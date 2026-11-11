use sgs_core::discover::{capture, compare, Candidate};
use sgs_core::paths::Env;
use std::fs;
use std::path::PathBuf;
use std::time::{Duration, SystemTime};

/// Env de teste: raízes apontadas para pastas temporárias, para não tocar nos
/// dados reais deste PC durante os testes. `<base>` é a pasta do "jogo".
fn env_teste(t: &PathBuf) -> Env {
    let home = t.join("home");
    let pub_dir = t.join("public");
    Env::new([
        ("home", home.clone()),
        ("winAppData", home.join("AppData/Roaming")),
        ("winLocalAppData", home.join("AppData/Local")),
        ("winPublic", pub_dir.clone()),
        ("winDocuments", home.join("Documents")),
        ("base", t.join("jogo")),
    ])
}

fn montar(t: &PathBuf) -> Env {
    let env = env_teste(t);
    for p in ["home/Documents", "home/AppData/Roaming", "home/AppData/Local", "public/Documents", "jogo"] {
        fs::create_dir_all(t.join(p)).unwrap();
    }
    env
}

fn tocar_fstempo(p: &PathBuf) {
    // Diferença visível de mtime entre captura e comparação.
    let file = fs::OpenOptions::new().append(true).open(p).unwrap();
    let t = SystemTime::now() + Duration::from_secs(1);
    file.set_modified(t).unwrap();
}

fn candidato<'a>(out: &'a [Candidate], fim: &str) -> &'a Candidate {
    out.iter().find(|c| c.logical == fim).expect("esperava este candidato")
}

#[test]
fn novos_e_alterados_varias_raizes() {
    let t = tempfile::tempdir().unwrap().keep();
    let env = montar(&t);
    // Estado antes: um save que já existe e um que o jogo ainda vai criar.
    let doc = t.join("home/Documents/MeuJogo");
    fs::create_dir_all(&doc).unwrap();
    fs::write(doc.join("save0.dat"), "antigo").unwrap();

    let before = capture(&env);

    // Novo arquivo em Documents, alteração no AppData\\Roaming e nada em Local.
    fs::write(doc.join("save1.dat"), "novo").unwrap();
    tocar_fstempo(&doc.join("save0.dat"));
    fs::create_dir_all(t.join("home/AppData/Roaming/OutroJogo")).unwrap();
    fs::write(t.join("home/AppData/Roaming/OutroJogo/logo.png"), "x").unwrap();

    let out = compare(&before, &env);
    let doc_c = candidato(&out, "<winDocuments>/MeuJogo");
    // O save alterado e o novo contam como evidência da pasta.
    assert_eq!(doc_c.added, 2);
    assert!(doc_c.removed == 0 && doc_c.bytes > 0);
    assert_eq!(candidato(&out, "<winAppData>/OutroJogo").added, 1);
    assert!(out.iter().all(|c| !c.logical.contains("<home>/AppData/Local")));
    fs::remove_dir_all(&t).unwrap();
}

#[test]
fn arquivo_direto_na_raiz_mostra_a_raiz_inteira() {
    let t = tempfile::tempdir().unwrap().keep();
    let env = montar(&t);
    let before = capture(&env);
    fs::write(t.join("home/Documents/sessao.sav"), "s").unwrap();
    let out = compare(&before, &env);
    let c = candidato(&out, "<winDocuments>");
    assert_eq!(c.added, 1);
    fs::remove_dir_all(&t).unwrap();
}

#[test]
fn ruido_nao_vira_candidato() {
    let t = tempfile::tempdir().unwrap().keep();
    let env = montar(&t);
    let before = capture(&env);
    let jogo = t.join("jogo");
    for p in [
        "cache/a.bin",
        "logs/a.log",
        "shaders/a.bin",
        "crash.dmp",
        "salvou.tmp",
        "ShaderCache/x.data",
    ] {
        let alvo = jogo.join(p);
        fs::create_dir_all(alvo.parent().unwrap()).unwrap();
        fs::write(&alvo, "x").unwrap();
    }
    // O jogo também salvou de verdade, para conferir que a lista não fica vazia.
    fs::create_dir_all(jogo.join("saves")).unwrap();
    fs::write(jogo.join("saves/slot1.dat"), "s").unwrap();
    let out = compare(&before, &env);
    assert_eq!(out.len(), 1, "só a pasta de saves deve aparecer: {out:?}");
    assert_eq!(out[0].logical, "<base>/saves");
    fs::remove_dir_all(&t).unwrap();
}

#[test]
fn arquivos_apagados_contam_como_evidencia() {
    let t = tempfile::tempdir().unwrap().keep();
    let env = montar(&t);
    let dir = t.join("home/Documents/SomJogo2");
    fs::create_dir_all(&dir).unwrap();
    fs::write(dir.join("slot.dat"), "s").unwrap();
    let before = capture(&env);
    fs::remove_file(dir.join("slot.dat")).unwrap();
    let out = compare(&before, &env);
    let c = candidato(&out, "<winDocuments>/SomJogo2");
    assert_eq!(c.removed, 1);
    assert_eq!(c.added, 0);
    fs::remove_dir_all(&t).unwrap();
}

#[test]
fn ordena_da_mais_provavel_para_a_menos() {
    let t = tempfile::tempdir().unwrap().keep();
    let env = montar(&t);
    let before = capture(&env);
    let dir = t.join("home/Documents/Muito");
    fs::create_dir_all(&dir).unwrap();
    for i in 0..5 {
        fs::write(dir.join(format!("slot{i}.dat")), "s").unwrap();
    }
    fs::create_dir_all(t.join("home/Documents/Pouco")).unwrap();
    fs::write(t.join("home/Documents/Pouco/um.dat"), "s").unwrap();
    let out = compare(&before, &env);
    assert_eq!(out[0].logical, "<winDocuments>/Muito");
    assert_eq!(out[1].logical, "<winDocuments>/Pouco");
    fs::remove_dir_all(&t).unwrap();
}

#[test]
fn my_games_e_locallow_agrupam_ate_o_jogo() {
    let t = tempfile::tempdir().unwrap().keep();
    let env = montar(&t);
    let before = capture(&env);
    fs::create_dir_all(t.join("home/Documents/My Games/Um Jogo")).unwrap();
    fs::write(t.join("home/Documents/My Games/Um Jogo/save.dat"), "s").unwrap();
    let ll = t.join("home/AppData/LocalLow/Dev/GrauUmUm/save.dat");
    fs::create_dir_all(ll.parent().unwrap()).unwrap();
    fs::write(&ll, "s").unwrap();
    let out = compare(&before, &env);
    assert_eq!(candidato(&out, "<winDocuments>/My Games/Um Jogo").added, 1);
    assert_eq!(candidato(&out, "<home>/AppData/LocalLow/Dev/GrauUmUm").added, 1);
    fs::remove_dir_all(&t).unwrap();
}

#[test]
fn pasta_criada_ou_apagada_entre_capturas() {
    let t = tempfile::tempdir().unwrap().keep();
    let env = montar(&t);
    // Pasta do save só passa a existir quando o jogo "salvou" pela primeira vez.
    let novo = t.join("jogo/Saves Novos");
    fs::create_dir_all(&novo).unwrap();
    let some = t.join("jogo/Sumiu");
    fs::create_dir_all(&some).unwrap();
    fs::write(some.join("a.dat"), "s").unwrap();
    let before = capture(&env);
    fs::write(novo.join("x.dat"), "s").unwrap();
    fs::remove_dir_all(&some).unwrap();
    let out = compare(&before, &env);
    assert_eq!(candidato(&out, "<base>/Saves Novos").added, 1);
    fs::remove_dir_all(&t).unwrap();
}

#[cfg(windows)]
#[test]
fn pasta_sem_acesso_nao_derruba_captura() {
    // Simula o antivírus/permissão: pasta com acesso negado para o usuário atual.
    let saida = std::process::Command::new("whoami").output().unwrap();
    let usuario = String::from_utf8_lossy(&saida.stdout).trim_start_matches(r"\").trim().to_string();
    let t = tempfile::tempdir().unwrap().keep();
    let env = montar(&t);
    let protegida = t.join("jogo/protegida");
    fs::create_dir_all(&protegida).unwrap();
    fs::write(protegida.join("secret.dat"), "s").unwrap();
    let negar_alvo = format!("{usuario}:(OI)(CI)F");
    let negar = std::process::Command::new("icacls")
        .args([&protegida.to_string_lossy(), "/deny", &negar_alvo])
        .output()
        .unwrap();
    if !negar.status.success() {
        eprintln!("icacls não funcionou neste sistema; teste sem ele.");
        return;
    }
    // Não importa se a captura alcança 0 ou 1 arquivos: não pode falhar nem travar.
    let _ = capture(&env);
    let _ = std::process::Command::new("icacls")
        .args([&protegida.to_string_lossy(), "/reset"])
        .output();
    let _ = std::fs::remove_dir_all(&t);
}
