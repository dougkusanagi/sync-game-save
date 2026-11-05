use sgs_catalog::{build, normalize, Catalog};

const YAML: &str = r#"
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

fn catalog() -> (tempfile::TempDir, Catalog) {
    let t = tempfile::tempdir().unwrap();
    let m = t.path().join("m.yaml");
    std::fs::write(&m, YAML).unwrap();
    let db = t.path().join("catalog.db");
    let stats = build(&m, &db).unwrap();
    assert_eq!((stats.games, stats.files), (2, 2));
    let c = Catalog::open(&db).unwrap();
    (t, c)
}

#[test]
fn normaliza() {
    assert_eq!(normalize("Épisode One!"), "episodeone");
}

#[test]
fn acha_pela_pasta_e_pelo_nome() {
    let (_t, c) = catalog();
    let by_dir = c.by_install_dir("AGGELOS").unwrap();
    assert_eq!(by_dir.len(), 1);
    assert_eq!(c.by_install_dir("half-life 2").unwrap()[0].name, "Half-Life 2: Épisode One");
    assert_eq!(c.search("agge", 5).unwrap()[0].name, "Aggelos");
    assert_eq!(c.search("Ag", 5).unwrap()[0].name, "Aggelos", "prefixo curto");
    assert!(c.search("zzzzz", 5).unwrap().is_empty());
    // Entrada com aspas não pode quebrar a consulta.
    assert!(c.search("\"agg", 5).is_ok());
}

#[test]
fn caminhos_de_save_e_launch() {
    let (_t, c) = catalog();
    let id = c.by_install_dir("Aggelos").unwrap()[0].id;
    let paths = c.save_paths(id).unwrap();
    assert_eq!(paths.len(), 2);
    let save = paths.iter().find(|p| p.tags == "save").unwrap();
    assert_eq!((save.path.as_str(), save.os.as_deref()), ("<winAppData>/Aggelos", Some("windows")));
    assert_eq!(c.launch_paths(id).unwrap(), vec!["<base>/LAUNCHER_x64.exe"]);
}

#[test]
fn aceita_caractere_de_controle_do_manifesto_real() {
    let t = tempfile::tempdir().unwrap();
    let m = t.path().join("m.yaml");
    std::fs::write(&m, "\"Jogo\":\n  launch:\n    <base>/a.exe:\n      - arguments: \"-x\u{7f}\"\n").unwrap();
    assert_eq!(build(&m, &t.path().join("c.db")).unwrap().games, 1);
}
