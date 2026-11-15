use sgs_catalog::recognize::{clean_folder_name, guess_exe, recognize, Confidence};
use sgs_catalog::{build, Catalog};
use sgs_core::paths::Env;
use std::fs;
use std::path::Path;

const YAML: &str = r#"
"241600":
  files:
    <winDocuments>/SavedGames/Aggelos:
      tags: [save]
      when:
        - os: windows
    <winAppData>/Aggelos:
      tags: [config]
      when:
        - os: windows
    <xdgData>/Aggelos:
      tags: [save]
      when:
        - os: linux
    <base>/Saves:
      tags: [save]
  installDir:
    Aggelos: {}
  launch:
    <base>/Aggelos.exe:
      - when:
          - os: windows
"#;

fn put(p: &Path, size: usize) {
    fs::create_dir_all(p.parent().unwrap()).unwrap();
    fs::write(p, vec![b'x'; size]).unwrap();
}

fn setup() -> (tempfile::TempDir, Catalog, Env) {
    let t = tempfile::tempdir().unwrap();
    let m = t.path().join("m.yaml");
    fs::write(&m, YAML).unwrap();
    let db = t.path().join("c.db");
    build(&m, &db).unwrap();
    let env = Env::new([
        ("home", t.path().join("home")),
        ("winDocuments", t.path().join("docs")),
        ("winAppData", t.path().join("appdata")),
    ]);
    (t, Catalog::open(&db).unwrap(), env)
}

#[test]
fn limpa_nome_da_pasta() {
    assert_eq!(clean_folder_name("The_Game.v1.2.3 [GOG] (x64)"), "The Game");
    assert_eq!(clean_folder_name("Aggelos"), "Aggelos");
}

#[test]
fn reconhece_pela_pasta_com_save_existente() {
    let (t, cat, env) = setup();
    let game = t.path().join("Games/Aggelos");
    put(&game.join("Aggelos.exe"), 10);
    put(&game.join("unins000.exe"), 999); // maior, mas é desinstalador
    put(&t.path().join("docs/SavedGames/Aggelos/slot1.sav"), 5);
    let r = recognize(&cat, &game, &env, true).unwrap();
    assert_eq!(r.confidence, Confidence::High);
    assert_eq!(r.name, "Aggelos", "o nome vem da pasta quando o catálogo só tem o id");
    assert_eq!(r.exe.unwrap().file_name().unwrap(), "Aggelos.exe");
    // A regra de config e a do Linux ficam de fora; só a de save do Windows aparece.
    assert_eq!(r.save_candidates.len(), 2);
    let c = r.save_candidates.iter().find(|c| c.logical.starts_with("<winDocuments>")).unwrap();
    assert_eq!((c.logical.as_str(), c.exists, c.files, c.selected), ("<winDocuments>/SavedGames/Aggelos", true, 1, true));
}

#[test]
fn save_inexistente_nao_vem_marcado() {
    let (t, cat, env) = setup();
    let game = t.path().join("Aggelos");
    put(&game.join("a.exe"), 3);
    let r = recognize(&cat, &game, &env, true).unwrap();
    assert!(r.save_candidates.iter().all(|c| !c.exists && !c.selected));
}

#[test]
fn varredura_resolve_base_para_cada_jogo_e_escolha_do_catalogo() {
    let (t, cat, env) = setup();
    let lib = t.path().join("Biblioteca");
    let game = lib.join("Aggelos");
    put(&game.join("Aggelos.exe"), 10);
    put(&game.join("Saves/slot.sav"), 4);
    let resultado = sgs_catalog::recognize::scan_folder(&cat, &lib, 1, &env, true).unwrap();
    let r = &resultado.recognized[0].1;
    let save = r.save_candidates.iter().find(|c| c.logical == "<base>/Saves").unwrap();
    assert_eq!(save.resolved, game.join("Saves"));
    assert!(save.selected);
    let escolhido = sgs_catalog::recognize::recognize_choice(&cat, &game, &env, true, r.catalog_id.unwrap()).unwrap().unwrap();
    assert_eq!(escolhido.name, r.name);
    assert!(escolhido.save_candidates.iter().any(|c| c.logical == "<base>/Saves" && c.selected));
    assert!(sgs_catalog::recognize::recognize_choice(&cat, &game, &env, true, -1).unwrap().is_none());
}

#[test]
fn jogo_desconhecido_cai_no_manual() {
    let (t, cat, env) = setup();
    let game = t.path().join("Jogo Que Nao Existe 99");
    put(&game.join("grande.exe"), 50);
    put(&game.join("redist/vcredist_x64.exe"), 500);
    let r = recognize(&cat, &game, &env, true).unwrap();
    assert_eq!(r.confidence, Confidence::None);
    assert_eq!(r.catalog_id, None);
    assert!(r.save_candidates.is_empty());
    assert_eq!(guess_exe(&game).unwrap().file_name().unwrap(), "grande.exe");
}

#[test]
fn varre_pasta_com_varios_jogos() {
    use sgs_catalog::recognize::scan_folder;
    let (t, cat, env) = setup();
    let lib = t.path().join("Biblioteca");
    put(&lib.join("Aggelos/Aggelos.exe"), 10);
    put(&lib.join("Desconhecido Total/x.exe"), 10);
    put(&lib.join("Loja/Dentro Do Nivel 2/y.exe"), 10);
    put(&lib.join("Sem Catalogo/bin/Win64/game.exe"), 10);
    put(&lib.join("Docs/leia.txt"), 1); // sem executável: ignorada
    let r = scan_folder(&cat, &lib, 2, &env, true).unwrap();
    assert_eq!(r.recognized.len(), 1);
    assert_eq!(r.unknown.len(), 3, "{:?}", r.unknown.iter().map(|u| &u.0).collect::<Vec<_>>());
    assert!(r.unknown.iter().any(|(p, _)| p == &lib.join("Loja/Dentro Do Nivel 2")));
    assert!(r.unknown.iter().any(|(p, _)| p == &lib.join("Sem Catalogo")));
    // O limite conta pastas de jogos, não a pasta técnica bin/.
    let r1 = scan_folder(&cat, &lib, 1, &env, true).unwrap();
    assert_eq!((r1.recognized.len(), r1.unknown.len()), (1, 2));
    assert!(!r1.unknown.iter().any(|(p, _)| p == &lib.join("Loja")));
}

#[test]
fn varredura_informa_pasta_inacessivel() {
    let (t, cat, env) = setup();
    let inexistente = t.path().join("nao-existe");
    let resultado = sgs_catalog::recognize::scan_folder(&cat, &inexistente, 2, &env, true).unwrap();
    assert_eq!(resultado.errors.len(), 1);
    assert_eq!(resultado.errors[0].0, inexistente);
}
