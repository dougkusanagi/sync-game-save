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
        assert_eq!(lib.schema_version().unwrap(), 2);
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

#[test]
fn soma_tempo_de_jogo() {
    let mut lib = Library::open_in_memory().unwrap();
    let id = lib.add_game(&game("Tempo"), 1).unwrap();
    assert_eq!(lib.get(id).unwrap().playtime_secs, 0);
    lib.add_playtime(id, 600, 100).unwrap();
    lib.add_playtime(id, 0, 200).unwrap(); // não conta
    lib.add_playtime(id, 30, 300).unwrap();
    let g = lib.get(id).unwrap();
    assert_eq!((g.playtime_secs, g.last_played_at), (630, Some(300)));
    assert!(matches!(lib.add_playtime(999, 5, 1), Err(Error::NotFound)));
}

#[test]
fn migra_banco_da_versao_1_sem_perder_jogos() {
    let t = tempfile::tempdir().unwrap();
    let p = t.path().join("library.db");
    {
        // Banco como o da primeira versão (esquema 1).
        let c = rusqlite::Connection::open(&p).unwrap();
        c.execute_batch(
            "CREATE TABLE games(id INTEGER PRIMARY KEY, name TEXT NOT NULL UNIQUE COLLATE NOCASE, exe TEXT NOT NULL, install_dir TEXT, catalog_id INTEGER, created_at INTEGER NOT NULL);
             CREATE TABLE save_paths(id INTEGER PRIMARY KEY, game_id INTEGER NOT NULL REFERENCES games(id) ON DELETE CASCADE, path TEXT NOT NULL, enabled INTEGER NOT NULL DEFAULT 1, UNIQUE(game_id, path));
             CREATE TABLE sync_state(game_id INTEGER PRIMARY KEY REFERENCES games(id) ON DELETE CASCADE, base_digest TEXT, last_sync_at INTEGER);
             CREATE TABLE settings(key TEXT PRIMARY KEY, value TEXT NOT NULL);
             INSERT INTO games(name,exe,created_at) VALUES('Antigo','a.exe',1);
             INSERT INTO sync_state(game_id) VALUES(1);
             PRAGMA user_version = 1;",
        )
        .unwrap();
    }
    let lib = Library::open(&p).unwrap();
    assert_eq!(lib.schema_version().unwrap(), 2);
    let g = lib.list().unwrap().remove(0);
    assert_eq!((g.name.as_str(), g.playtime_secs), ("Antigo", 0));
}
