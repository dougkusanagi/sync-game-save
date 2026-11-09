use sgs_core::index::scan;
use sgs_core::snapshot::{create, read_head, restore, verify};
use sgs_core::sync::{decide, Action};
use std::fs;

fn write(dir: &std::path::Path, rel: &str, body: &str) {
    let p = dir.join(rel);
    fs::create_dir_all(p.parent().unwrap()).unwrap();
    fs::write(p, body).unwrap();
}

#[test]
fn tres_vias() {
    let t = tempfile::tempdir().unwrap();
    write(t.path(), "a/slot1.sav", "um");
    let a = scan(t.path()).unwrap();
    write(t.path(), "a/slot1.sav", "dois");
    let b = scan(t.path()).unwrap();
    let (da, db) = (a.digest(), b.digest());
    assert_eq!(decide(Some(&da), &a, Some(&da)), Action::Nothing);
    assert_eq!(decide(Some(&da), &a, Some(&db)), Action::RestoreFromCloud);
    assert_eq!(decide(Some(&da), &b, Some(&da)), Action::UploadLocal);
    assert_eq!(decide(Some("x"), &b, Some(&da)), Action::Conflict);
    assert_eq!(decide(Some("x"), &b, Some(&db)), Action::Nothing);
    assert_eq!(decide(None, &b, None), Action::UploadLocal);
}

#[test]
fn snapshot_e_restauracao_com_backup() {
    let t = tempfile::tempdir().unwrap();
    let (save, cloud, bkp) = (t.path().join("save"), t.path().join("cloud"), t.path().join("bkp"));
    write(&save, "dir/ação ç.sav", "v1");
    let head = create(&cloud, &save, "s1", "PC-A").unwrap();
    assert_eq!(read_head(&cloud).unwrap(), Some(head));
    write(&save, "dir/ação ç.sav", "v2 local");
    write(&save, "extra.sav", "novo");
    restore(&cloud.join("snapshots/s1"), &save, &bkp).unwrap();
    assert_eq!(fs::read_to_string(save.join("dir/ação ç.sav")).unwrap(), "v1");
    assert!(!save.join("extra.sav").exists());
    // O que existia antes da restauração continua recuperável.
    assert_eq!(fs::read_to_string(bkp.join("dir/ação ç.sav")).unwrap(), "v2 local");
    assert_eq!(fs::read_to_string(bkp.join("extra.sav")).unwrap(), "novo");
    write(&save, "dir/ação ç.sav", "outro progresso");
    assert!(restore(&cloud.join("snapshots/s1"), &save, &bkp).is_err());
    assert_eq!(fs::read_to_string(save.join("dir/ação ç.sav")).unwrap(), "outro progresso");
    assert_eq!(fs::read_to_string(bkp.join("dir/ação ç.sav")).unwrap(), "v2 local");
}

#[test]
fn falha_no_snapshot_nao_move_o_head() {
    let t = tempfile::tempdir().unwrap();
    let (save, cloud) = (t.path().join("save"), t.path().join("cloud"));
    write(&save, "a.sav", "v1");
    let h1 = create(&cloud, &save, "s1", "PC-A").unwrap();
    write(&save, "a.sav", "v2");
    // Um arquivo no lugar da pasta do snapshot força erro na gravação.
    fs::write(cloud.join("snapshots/s2"), "x").unwrap();
    assert!(create(&cloud, &save, "s2", "PC-A").is_err());
    assert_eq!(read_head(&cloud).unwrap(), Some(h1));
}

#[test]
fn snapshot_existente_nao_pode_ser_sobrescrito() {
    let t = tempfile::tempdir().unwrap();
    let (save, cloud) = (t.path().join("save"), t.path().join("cloud"));
    write(&save, "a.sav", "original a");
    write(&save, "z/slot.sav", "original z");
    let anterior = create(&cloud, &save, "s1", "PC-A").unwrap();
    write(&save, "a.sav", "novo a");
    write(&save, "z/slot.sav", "novo z");
    assert!(create(&cloud, &save, "s1", "PC-A").is_err());
    assert_eq!(fs::read_to_string(cloud.join("snapshots/s1/files/a.sav")).unwrap(), "original a");
    assert_eq!(fs::read_to_string(cloud.join("snapshots/s1/files/z/slot.sav")).unwrap(), "original z");
    assert_eq!(read_head(&cloud).unwrap(), Some(anterior));
    assert_eq!(fs::read_to_string(save.join("a.sav")).unwrap(), "novo a");
    assert_eq!(fs::read_to_string(save.join("z/slot.sav")).unwrap(), "novo z");
}

#[test]
fn hash_divergente_e_detectado() {
    let t = tempfile::tempdir().unwrap();
    let (save, cloud) = (t.path().join("save"), t.path().join("cloud"));
    write(&save, "a.sav", "v1");
    create(&cloud, &save, "s1", "PC-A").unwrap();
    fs::write(cloud.join("snapshots/s1/files/a.sav"), "corrompido").unwrap();
    assert!(verify(&cloud.join("snapshots/s1")).is_err());
}
