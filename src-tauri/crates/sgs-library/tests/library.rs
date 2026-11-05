use sgs_library::{Error, Library, NewGame};

fn game(name: &str) -> NewGame {
    NewGame {
        name: name.into(),
        exe: "C:/Games/x/x.exe".into(),
        install_dir: Some("C:/Games/x".into()),
        catalog_id: None,
        save_paths: vec!["<winAppData>/X".into(), "<home>/Saved Games/X".into()],
    }
}

#[test]
fn adiciona_lista_edita_e_remove() {
    let mut lib = Library::open_in_memory().unwrap();
    let id = lib.add_game(&game("  Árvore & Ação  "), 10).unwrap();
    let g = lib.get(id).unwrap();
    assert_eq!(g.name, "Árvore & Ação");
    assert_eq!(g.save_paths.len(), 2);
    assert_eq!(g.base_digest, None);

    let mut novo = game("Outro nome");
    novo.save_paths = vec!["<base>/saves".into()];
    lib.update_game(id, &novo).unwrap();
    let g = lib.get(id).unwrap();
    assert_eq!((g.name.as_str(), g.save_paths.clone()), ("Outro nome", vec!["<base>/saves".to_string()]));

    lib.record_sync(id, "abc", 20).unwrap();
    assert_eq!(lib.get(id).unwrap().base_digest.as_deref(), Some("abc"));

    lib.remove_game(id).unwrap();
    assert!(matches!(lib.get(id), Err(Error::NotFound)));
    assert!(matches!(lib.remove_game(id), Err(Error::NotFound)));
    assert!(lib.list().unwrap().is_empty());
}

#[test]
fn nome_repetido_ou_vazio_e_recusado() {
    let mut lib = Library::open_in_memory().unwrap();
    lib.add_game(&game("Jogo"), 1).unwrap();
    assert!(matches!(lib.add_game(&game("jogo"), 2), Err(Error::DuplicateName)));
    assert!(matches!(lib.add_game(&game("   "), 3), Err(Error::EmptyName)));
    // A tentativa recusada não deixa jogo pela metade.
    assert_eq!(lib.list().unwrap().len(), 1);
    let outro = lib.add_game(&game("Outro"), 4).unwrap();
    assert!(matches!(lib.update_game(outro, &game("JOGO")), Err(Error::DuplicateName)));
    assert_eq!(lib.get(outro).unwrap().name, "Outro");
}

#[test]
fn configuracoes_e_persistencia_com_migracao() {
    let t = tempfile::tempdir().unwrap();
    let p = t.path().join("dados/library.db");
    {
        let mut lib = Library::open(&p).unwrap();
        assert_eq!(lib.schema_version().unwrap(), 1);
        lib.set_setting("sync_dir", "D:/OneDrive").unwrap();
        lib.set_setting("sync_dir", "E:/Nuvem").unwrap();
        lib.add_game(&game("Persistente"), 1).unwrap();
    }
    let lib = Library::open(&p).unwrap();
    assert_eq!(lib.setting("sync_dir").unwrap().as_deref(), Some("E:/Nuvem"));
    assert_eq!(lib.setting("nada").unwrap(), None);
    assert_eq!(lib.list().unwrap()[0].name, "Persistente");
}

#[test]
fn recusa_banco_de_versao_mais_nova() {
    let t = tempfile::tempdir().unwrap();
    let p = t.path().join("library.db");
    drop(Library::open(&p).unwrap());
    let c = rusqlite::Connection::open(&p).unwrap();
    c.pragma_update(None, "user_version", 99).unwrap();
    drop(c);
    assert!(matches!(Library::open(&p), Err(Error::TooNew(99))));
}
