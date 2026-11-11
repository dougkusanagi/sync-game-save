use sgs_core::paths::Env;
use sgs_library::NewGame;
use std::fs;
use std::path::{Path, PathBuf};

fn env_fixo(raiz: &Path) -> Env {
    let home = raiz.to_path_buf().join("home");
    Env::new([
        ("base", raiz.to_path_buf().join("jogo")),
        ("home", home.clone()),
        ("winAppData", home.join("AppData/Roaming")),
        ("winLocalAppData", home.join("AppData/Local")),
        ("winPublic", raiz.to_path_buf().join("public")),
        ("winDocuments", home.join("Documents")),
    ])
}

fn svc_grande(t: &Path, raiz: &Path) -> sgs_service::Service {
    let raiz = raiz.to_path_buf();
    sgs_service::Service::with(
        t.to_path_buf().join("dados").as_path(),
        Box::new(move |_| env_fixo(&raiz)),
        Box::new(|| 0),
    )
    .unwrap()
}

#[test]
fn descobre_onde_o_jogo_guarda_o_save() {
    let t = tempfile::tempdir().unwrap();
    let raiz = t.path().to_path_buf();
    let svc = svc_grande(t.path(), &raiz);
    // Estado antes: o jogo já tinha um save e vai alterá-lo ao fechar.
    let save_dir = raiz.join("home/Documents/Meu Jogo");
    fs::create_dir_all(&save_dir).unwrap();
    fs::write(save_dir.join("slot0.dat"), "a").unwrap();

    // Falso jogo: um cmd.exe na pasta do jogo (o rastreamento por pasta só confia em
    // processos de dentro dela; um cmd de System32 combinaria com o sistema todo).
    let jogo = raiz.join("jogo");
    fs::create_dir_all(&jogo).unwrap();
    let clone = jogo.join("jogo-clon.exe");
    fs::copy(r"C:\Windows\System32\cmd.exe", &clone).unwrap();

    let lote = jogo.join("salva.bat");
    fs::write(
        &lote,
        format!(
            "@echo off\r\necho x >> \"{}\"\r\necho y > \"{}\"\r\n",
            save_dir.join("slot0.dat").display(),
            save_dir.join("slot1.dat").display()
        ),
    )
    .unwrap();

    let code = svc
        .discover_run(&clone, &["/c".into(), lote.to_string_lossy().into_owned()], 1)
        .expect("a abertura de descoberta deve funcionar");
    assert_eq!(code, 0);

    let cands = svc.discover_candidates(&clone).unwrap();
    assert_eq!(cands.len(), 1, "candidatos: {cands:?}");
    assert_eq!(cands[0].logical, "<winDocuments>/Meu Jogo");
    assert_eq!(cands[0].added, 2, "o save alterado e o novo contam");
    assert!(Path::new(&cands[0].resolved).is_dir(), "o caminho resolvido é a pasta real");

    // Confirmar não é automático: o caminho só vira save após compor no jogo.
    let id = svc.add_game(&NewGame { name: "Meu Jogo".into(), exe: "jogo-clon.exe".into(), install_dir: None, catalog_id: None, save_paths: vec![cands[0].logical.clone()] }).unwrap();
    assert_eq!(svc.games().unwrap()[0].id, id);
}

#[test]
fn descoberta_exige_executavel_existente_e_estado_capturado() {
    let t = tempfile::tempdir().unwrap();
    let svc = svc_grande(t.path(), t.path());
    let err = svc.discover_run(Path::new("c:/nao/existe.exe"), &[], 1).unwrap_err();
    assert!(err.to_string().contains("executável"), "erro: {err}");

    // Sem captura anterior, não há de onde comparar (candidatos só existem depois de `discover_run`).
    let sem = PathBuf::from(r"C:\Windows\System32\cmd.exe");
    let err = svc.discover_candidates(&sem).unwrap_err();
    assert!(err.to_string().contains("captura"), "erro: {err}");
}
