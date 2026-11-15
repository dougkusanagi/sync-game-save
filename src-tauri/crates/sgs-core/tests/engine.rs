use sgs_core::engine::{Before, Choice, Ctx};
use sgs_core::snapshot::read_head;
use std::fs;
use std::path::Path;

fn pc(root: &Path, name: &str, now: u64) -> Ctx {
    Ctx {
        game_dir: root.join("cloud/jogo"),
        roots: vec![(String::new(), root.join(name).join("save"))],
        state_dir: root.join(name).join("state"),
        device: name.to_string(),
        now,
    }
}

fn put(c: &Ctx, body: &str) {
    fs::create_dir_all(&c.roots[0].1).unwrap();
    fs::write(c.roots[0].1.join("s.sav"), body).unwrap();
}

fn get(c: &Ctx) -> String {
    fs::read_to_string(c.roots[0].1.join("s.sav")).unwrap()
}

#[test]
fn espera_o_save_parar_de_mudar_antes_de_criar_snapshot() {
    let t = tempfile::tempdir().unwrap();
    let a = pc(t.path(), "A", 1000);
    put(&a, "antes");
    a.before_launch(false).unwrap();
    let save = a.roots[0].1.join("s.sav");
    let escritor = std::thread::spawn(move || {
        std::thread::sleep(std::time::Duration::from_millis(3500));
        fs::write(save, "final").unwrap();
    });
    let inicio = std::time::Instant::now();
    assert!(a.after_exit().unwrap());
    escritor.join().unwrap();
    assert!(inicio.elapsed() >= std::time::Duration::from_secs(5));
    let head = read_head(&a.game_dir).unwrap().unwrap();
    let versoes = sgs_core::snapshot::list(&a.game_dir).unwrap();
    assert_eq!(versoes.len(), 2);
    assert_eq!(versoes[0], head.snapshot);
    assert_eq!(fs::read_to_string(a.game_dir.join("snapshots").join(&versoes[0]).join("files/s.sav")).unwrap(), "final");
    assert_eq!(fs::read_to_string(a.game_dir.join("snapshots").join(&versoes[1]).join("files/s.sav")).unwrap(), "antes");
}

#[test]
fn dois_pcs_em_sequencia() {
    let t = tempfile::tempdir().unwrap();
    let a = pc(t.path(), "A", 1000);
    put(&a, "v1");
    assert_eq!(a.before_launch(false).unwrap(), Before::UploadedLocal);
    put(&a, "v2");
    assert!(a.after_exit().unwrap());

    // PC B ainda sem save: recebe o da nuvem sem conflito.
    let b = pc(t.path(), "B", 2000);
    assert_eq!(b.before_launch(false).unwrap(), Before::RestoredFromCloud);
    assert_eq!(get(&b), "v2");
    put(&b, "v3");
    assert!(b.after_exit().unwrap());

    // PC A volta: só a nuvem mudou.
    let a2 = pc(t.path(), "A", 3000);
    assert_eq!(a2.before_launch(false).unwrap(), Before::RestoredFromCloud);
    assert_eq!(get(&a2), "v3");
    // Fechou sem mudar o save: não cria snapshot.
    assert!(!a2.after_exit().unwrap());
}

#[test]
fn conflito_nao_perde_nenhum_lado() {
    let t = tempfile::tempdir().unwrap();
    let a = pc(t.path(), "A", 1000);
    put(&a, "base");
    a.before_launch(false).unwrap();
    a.after_exit().unwrap();
    let b = pc(t.path(), "B", 2000);
    b.before_launch(false).unwrap();
    b.after_exit().unwrap();

    // Os dois jogam offline e mudam o save.
    put(&a, "de A");
    put(&b, "de B");
    assert!(b.after_exit().unwrap());
    let a2 = pc(t.path(), "A", 4000);
    assert_eq!(a2.before_launch(false).unwrap(), Before::Conflict);
    assert_eq!(get(&a2), "de A", "nada muda antes da escolha");

    a2.resolve(Choice::UseCloud).unwrap();
    assert_eq!(get(&a2), "de B");
    let bkp = a2.state_dir.join("backups").join("4000_conflito/s.sav");
    assert_eq!(fs::read_to_string(bkp).unwrap(), "de A");
}

#[test]
fn trava_de_outro_pc_pede_confirmacao() {
    let t = tempfile::tempdir().unwrap();
    let a = pc(t.path(), "A", 1000);
    put(&a, "v1");
    a.before_launch(false).unwrap(); // A fica jogando, com trava
    let b = pc(t.path(), "B", 1100);
    assert_eq!(b.before_launch(false).unwrap(), Before::LockedByOther("A".into()));
    assert_ne!(b.before_launch(true).unwrap(), Before::LockedByOther("A".into()));
}

#[test]
fn fechar_com_erro_ainda_salva_e_solta_a_trava() {
    let t = tempfile::tempdir().unwrap();
    let a = pc(t.path(), "A", 1000);
    put(&a, "v1");
    a.before_launch(false).unwrap();
    put(&a, "v2 antes do crash");
    assert!(a.after_exit().unwrap());
    assert!(!a.game_dir.join("lock.json").exists());
    assert!(read_head(&a.game_dir).unwrap().is_some());
}

#[test]
fn primeiro_backup_cria_a_primeira_versao_e_solta_a_trava() {
    use sgs_core::engine::FirstBackup;
    let t = tempfile::tempdir().unwrap();
    let a = pc(t.path(), "A", 1000);
    assert_eq!(a.first_backup().unwrap(), FirstBackup::NothingToBackup);
    put(&a, "v1");
    assert_eq!(a.first_backup().unwrap(), FirstBackup::Created);
    assert!(read_head(&a.game_dir).unwrap().is_some());
    assert!(!a.game_dir.join("lock.json").exists());
    // Repetir não cria outra versão.
    assert_eq!(a.first_backup().unwrap(), FirstBackup::AlreadySynced);
    assert_eq!(sgs_core::snapshot::list(&a.game_dir).unwrap().len(), 1);
}

#[test]
fn primeiro_backup_nao_substitui_versao_existente_de_outro_pc() {
    use sgs_core::engine::FirstBackup;
    let t = tempfile::tempdir().unwrap();
    let a = pc(t.path(), "A", 1000);
    put(&a, "da-nuvem");
    assert_eq!(a.first_backup().unwrap(), FirstBackup::Created);
    let b = pc(t.path(), "B", 2000);
    put(&b, "local-diferente");
    assert_eq!(b.first_backup().unwrap(), FirstBackup::NeedsDecision);
    assert_eq!(get(&b), "local-diferente");
    assert_eq!(sgs_core::snapshot::list(&b.game_dir).unwrap().len(), 1);
    // Mesmo conteúdo nos dois lados: só liga o PC à versão existente.
    let c = pc(t.path(), "C", 3000);
    put(&c, "da-nuvem");
    assert_eq!(c.first_backup().unwrap(), FirstBackup::AlreadySynced);
}

#[test]
fn primeiro_backup_recusa_com_trava_de_outro_pc_ou_deste() {
    use sgs_core::engine::FirstBackup;
    let t = tempfile::tempdir().unwrap();
    let a = pc(t.path(), "A", 1000);
    put(&a, "v1");
    sgs_core::lock::acquire(&a.game_dir, "A", 1000).unwrap();
    assert_eq!(a.first_backup().unwrap(), FirstBackup::InUse);
    let b = pc(t.path(), "B", 1100);
    put(&b, "v1");
    assert_eq!(b.first_backup().unwrap(), FirstBackup::LockedByOther("A".into()));
    assert!(read_head(&b.game_dir).unwrap().is_none());
    // Trava antiga de outro PC é tratada como abandonada.
    let c = pc(t.path(), "C", 1000 + 3600);
    put(&c, "v1");
    assert_eq!(c.first_backup().unwrap(), FirstBackup::Created);
}
