use sgs_core::env::system_env;
use sgs_core::journal;
use sgs_core::paths::resolve;
use sgs_core::snapshot::{create, list, prune, read_head};
use std::fs;

#[test]
fn journal_detecta_operacao_interrompida() {
    let t = tempfile::tempdir().unwrap();
    assert!(journal::pending(t.path()).unwrap().is_empty());
    journal::begin(t.path(), "jogo", "restore", 5).unwrap();
    assert_eq!(journal::pending(t.path()).unwrap().len(), 1);
    journal::end(t.path(), "jogo").unwrap();
    assert!(journal::pending(t.path()).unwrap().is_empty());
}

#[test]
fn retencao_preserva_head_e_os_mais_novos() {
    let t = tempfile::tempdir().unwrap();
    let (save, cloud) = (t.path().join("s"), t.path().join("c"));
    fs::create_dir_all(&save).unwrap();
    for n in [100u64, 200, 300, 400] {
        fs::write(save.join("a.sav"), n.to_string()).unwrap();
        create(&cloud, &save, &format!("{n}_PC"), "PC").unwrap();
    }
    // Snapshot incompleto (sem index.json) não conta nem é listado.
    fs::create_dir_all(cloud.join("snapshots/999_PC")).unwrap();
    assert_eq!(list(&cloud).unwrap().len(), 4);
    let removed = prune(&cloud, 2).unwrap();
    assert_eq!(removed, vec!["200_PC", "100_PC"]);
    assert_eq!(read_head(&cloud).unwrap().unwrap().snapshot, "400_PC");
    assert_eq!(list(&cloud).unwrap(), vec!["400_PC", "300_PC"]);
}

#[test]
fn env_do_sistema_tem_base_e_home() {
    let env = system_env(Some("C:/Games/X".into()));
    assert!(resolve("<base>/save", &env).is_some());
    assert!(resolve("<home>", &env).is_some());
}

#[test]
fn documentos_usa_a_pasta_do_sistema() {
    let env = system_env(None);
    if let Some(real) = dirs::document_dir() {
        assert_eq!(resolve("<winDocuments>", &env), Some(real));
    }
}
