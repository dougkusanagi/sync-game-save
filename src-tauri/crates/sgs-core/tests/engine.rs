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
