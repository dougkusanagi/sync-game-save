//! Gera dados no formato da versão Rust para os testes de compatibilidade do porte em Go.
//! Uso: cargo run -p sgs-service --example gerar_fixtures -- <pasta de saída>

use sgs_core::discover::{capture_at, hex_key};
use sgs_core::paths::Env;
use sgs_library::{Library, NewGame};
use std::path::PathBuf;

const MANIFESTO: &str = r#"
"Aggelos":
  files:
    <winAppData>/Aggelos:
      tags: [save]
      when:
        - os: windows
    <base>/config.ini:
      tags: [config]
  installDir:
    Aggelos: {}
  launch:
    <base>/LAUNCHER_x64.exe:
      - when:
          - os: windows
  steam:
    id: 123
"Half-Life 2: Épisode One":
  installDir:
    Half-Life 2: {}
"#;

const CONQUISTAS: &str = r#"{"achievements":[{"id":"A","name":"Um","description":"d","icon":"i","icon_gray":"g","hidden":false,"unlocked":true,"unlock_time":1700000000,"percent":91.5,"tier":"Common"}],"progress_known":true}"#;

fn main() {
    let out = PathBuf::from(std::env::args().nth(1).expect("falta a pasta de saída"));
    std::fs::create_dir_all(&out).unwrap();

    let db = out.join("library.db");
    let _ = std::fs::remove_file(&db);
    let mut lib = Library::open(&db).unwrap();
    let a = lib
        .add_game(
            &NewGame { name: "Ação & Aventura".into(), exe: "C:/Jogos/aa.exe".into(), install_dir: Some("C:/Jogos".into()), catalog_id: Some(7), save_paths: vec!["<winDocuments>/AA".into(), "D:/Saves/AA".into()] },
            1_700_000_000,
        )
        .unwrap();
    let b = lib
        .add_game(&NewGame { name: "Outro".into(), exe: "outro.exe".into(), install_dir: None, catalog_id: None, save_paths: vec![] }, 1_700_000_100)
        .unwrap();
    lib.record_sync(a, "abc123", 1_700_000_500).unwrap();
    lib.add_playtime(a, 3725, 1_700_001_000).unwrap();
    lib.save_achievements(a, CONQUISTAS, 1_700_002_000).unwrap();
    lib.set_setting("sync_dir", "C:/Nuvem").unwrap();
    lib.set_setting("device", "PC-RUST").unwrap();
    lib.set_setting("sync_ok_streak", "3").unwrap();
    lib.set_setting("theme", "claro").unwrap();
    println!("jogos: {a} {b}");

    let manifest = out.join("manifest.yaml");
    std::fs::write(&manifest, MANIFESTO).unwrap();
    let stats = sgs_catalog::build(&manifest, &out.join("catalog.db")).unwrap();
    println!("catálogo: {} jogos, {} arquivos", stats.games, stats.files);
    std::fs::remove_file(&manifest).unwrap();

    // Captura da descoberta de save: pasta com um arquivo conhecido.
    let base = out.join("jogo");
    std::fs::create_dir_all(base.join("Saves")).unwrap();
    std::fs::write(base.join("Saves/slot.sav"), b"12345").unwrap();
    let env = Env::new([("base", base.clone())]);
    let cap = capture_at(&env, 1_700_003_000);
    let exe = "C:/Jogos/aa.exe";
    std::fs::write(out.join(format!("captura-{}.json", hex_key(exe))), serde_json::to_vec(&cap).unwrap()).unwrap();
    std::fs::remove_dir_all(&base).unwrap();
    println!("chave: {}", hex_key(exe));
}
