//! Cenários de teste do plano (seção "Cenários de teste do sync").

use sgs_core::engine::{Before, Ctx};
use sgs_core::index::scan;
use sgs_core::snapshot::{create, restore};
use std::fs;
use std::path::Path;

fn write(dir: &Path, rel: &str, body: &str) {
    let p = dir.join(rel);
    fs::create_dir_all(p.parent().unwrap()).unwrap();
    fs::write(p, body).unwrap();
}

fn ctx(root: &Path, pc: &str, now: u64) -> Ctx {
    Ctx {
        game_dir: root.join("nuvem/jogo"),
        save_dir: root.join(pc).join("save"),
        state_dir: root.join(pc).join("state"),
        device: pc.into(),
        now,
    }
}

#[test]
fn nuvem_parcialmente_sincronizada_nao_toca_no_save_local() {
    let t = tempfile::tempdir().unwrap();
    let a = ctx(t.path(), "A", 1000);
    write(&a.save_dir, "s.sav", "v1");
    a.before_launch(false).unwrap();
    a.after_exit().unwrap();
    // O arquivo chega pela metade no PC B (hash não bate).
    fs::write(a.game_dir.join("snapshots/1000_A/files/s.sav"), "v").unwrap();
    let b = ctx(t.path(), "B", 2000);
    write(&b.save_dir, "s.sav", "do B");
    let r = b.before_launch(false);
    // Sem base no B e dois lados diferentes: conflito ou erro, mas nunca sobrescreve.
    assert!(matches!(r, Ok(Before::Conflict) | Err(_)));
    assert_eq!(fs::read_to_string(b.save_dir.join("s.sav")).unwrap(), "do B");

    // PC novo sem save: a restauração falha na verificação e nada é criado.
    let c = ctx(t.path(), "C", 3000);
    fs::create_dir_all(&c.save_dir).unwrap();
    assert!(c.before_launch(false).is_err());
    assert!(scan(&c.save_dir).unwrap().files.is_empty());
    assert!(!a.game_dir.join("lock.json").exists() || fs::read_to_string(a.game_dir.join("lock.json")).unwrap().contains("\"A\""));
}

#[test]
fn falha_no_backup_nao_sobrescreve_o_save() {
    let t = tempfile::tempdir().unwrap();
    let (save, cloud) = (t.path().join("save"), t.path().join("cloud"));
    write(&save, "s.sav", "v1");
    create(&cloud, &save, "1_A", "A").unwrap();
    write(&save, "s.sav", "meu progresso");
    // O destino do backup é um arquivo: não dá para criar a pasta (simula disco/permissão).
    let bkp = t.path().join("bkp");
    fs::write(&bkp, "x").unwrap();
    assert!(restore(&cloud.join("snapshots/1_A"), &save, &bkp).is_err());
    assert_eq!(fs::read_to_string(save.join("s.sav")).unwrap(), "meu progresso");
}

#[test]
fn caminho_profundo_acima_de_260_caracteres() {
    let t = tempfile::tempdir().unwrap();
    let (save, cloud, out) = (t.path().join("save"), t.path().join("cloud"), t.path().join("out"));
    let deep = format!("{}/{}/{}/slot.sav", "a".repeat(90), "b".repeat(90), "c".repeat(90));
    write(&save, &deep, "fundo");
    create(&cloud, &save, "1_A", "A").unwrap();
    restore(&cloud.join("snapshots/1_A"), &out, &t.path().join("bkp")).unwrap();
    assert_eq!(fs::read_to_string(out.join(&deep)).unwrap(), "fundo");
}

#[test]
fn arquivos_apagados_entre_sessoes_voltam_na_restauracao() {
    let t = tempfile::tempdir().unwrap();
    let (save, cloud, bkp) = (t.path().join("save"), t.path().join("cloud"), t.path().join("bkp"));
    write(&save, "slot1.sav", "1");
    write(&save, "pasta/slot2.sav", "2");
    create(&cloud, &save, "1_A", "A").unwrap();
    fs::remove_file(save.join("slot1.sav")).unwrap();
    create(&cloud, &save, "2_A", "A").unwrap();
    restore(&cloud.join("snapshots/1_A"), &save, &bkp).unwrap();
    assert_eq!(fs::read_to_string(save.join("slot1.sav")).unwrap(), "1");
    assert_eq!(fs::read_to_string(save.join("pasta/slot2.sav")).unwrap(), "2");
}

#[test]
fn save_vazio_ou_ausente_nao_gera_snapshot_nem_apaga_a_nuvem() {
    let t = tempfile::tempdir().unwrap();
    let a = ctx(t.path(), "A", 1000);
    write(&a.save_dir, "s.sav", "v1");
    a.before_launch(false).unwrap();
    a.after_exit().unwrap();
    // O save some do PC (pasta ausente): não cria snapshot vazio por cima.
    fs::remove_dir_all(&a.save_dir).unwrap();
    let a2 = ctx(t.path(), "A", 2000);
    a2.before_launch(false).unwrap();
    assert!(!a2.after_exit().unwrap());
    assert!(a.game_dir.join("snapshots/1000_A/files/s.sav").exists());
}

#[test]
fn maiusculas_e_minusculas_ficam_como_estao_no_indice() {
    let t = tempfile::tempdir().unwrap();
    write(t.path(), "Save/Slot.SAV", "x");
    let idx = scan(t.path()).unwrap();
    assert!(idx.files.contains_key("Save/Slot.SAV"));
}

#[cfg(windows)]
#[test]
fn arquivo_preso_por_pouco_tempo_e_copiado_depois_da_espera() {
    use std::os::windows::fs::OpenOptionsExt;
    let t = tempfile::tempdir().unwrap();
    let (save, cloud) = (t.path().join("save"), t.path().join("cloud"));
    write(&save, "s.sav", "v1");
    // Abre sem compartilhamento: leitura por outro processo falha.
    let held = fs::OpenOptions::new().read(true).share_mode(0).open(save.join("s.sav")).unwrap();
    let h = std::thread::spawn(move || {
        std::thread::sleep(std::time::Duration::from_millis(600));
        drop(held);
    });
    create(&cloud, &save, "1_A", "A").unwrap();
    h.join().unwrap();
    assert_eq!(fs::read_to_string(cloud.join("snapshots/1_A/files/s.sav")).unwrap(), "v1");
}

#[cfg(windows)]
#[test]
fn arquivo_preso_para_sempre_da_erro_sem_mexer_no_head() {
    use std::os::windows::fs::OpenOptionsExt;
    let t = tempfile::tempdir().unwrap();
    let (save, cloud) = (t.path().join("save"), t.path().join("cloud"));
    write(&save, "s.sav", "v1");
    let _held = fs::OpenOptions::new().read(true).share_mode(0).open(save.join("s.sav")).unwrap();
    assert!(create(&cloud, &save, "1_A", "A").is_err());
    assert!(!cloud.join("head.json").exists());
}
