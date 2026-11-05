use sgs_cli::exec;
use std::fs;
use std::path::Path;

fn sgs(args: &[&str], stdin: &str) -> (Result<i32, String>, String) {
    let mut out = Vec::new();
    let r = exec(args.iter().map(|s| s.to_string()).collect(), &mut out, &mut stdin.as_bytes());
    (r, String::from_utf8(out).unwrap())
}

fn base<'a>(root: &'a Path, pc: &'a str, now: &'a str) -> Vec<String> {
    let r = |s: &str| root.join(s).to_string_lossy().into_owned();
    vec![
        "--sync-dir".into(), r("nuvem"),
        "--game".into(), "Jogo: Teste".into(),
        "--save-dir".into(), r(&format!("{pc}/save")),
        "--state-dir".into(), r(&format!("{pc}/state")),
        "--device".into(), pc.into(),
        "--now".into(), now.into(),
    ]
}

fn call(cmd: &str, extra: &[&str], b: &[String], stdin: &str) -> (Result<i32, String>, String) {
    let mut v: Vec<&str> = vec![cmd];
    v.extend(b.iter().map(String::as_str));
    v.extend(extra);
    sgs(&v, stdin)
}

/// Programa que grava no save e sai com o código dado.
fn writer(save: &Path, code: i32) -> Vec<String> {
    let file = save.join("slot.sav");
    if cfg!(windows) {
        vec!["powershell".into(), "-NoProfile".into(), "-Command".into(), format!("Set-Content -Path '{}' -Value novo; exit {code}", file.display())]
    } else {
        vec!["sh".into(), "-c".into(), format!("echo novo > '{}'; exit {code}", file.display())]
    }
}

#[test]
fn run_salva_mesmo_com_codigo_de_erro_e_outro_pc_recebe() {
    let t = tempfile::tempdir().unwrap();
    let a = base(t.path(), "A", "1000");
    fs::create_dir_all(t.path().join("A/save")).unwrap();
    fs::create_dir_all(t.path().join("nuvem")).unwrap();
    let prog = writer(&t.path().join("A/save"), 7);
    let mut extra: Vec<&str> = vec!["--"];
    extra.extend(prog.iter().map(String::as_str));
    let (r, out) = call("run", &extra, &a, "");
    assert_eq!(r.unwrap(), 7, "{out}");
    assert!(out.contains("Save enviado para a nuvem"), "{out}");

    let (r, out) = call("list", &[], &a, "");
    assert_eq!(r.unwrap(), 0);
    assert!(out.contains("1000_A"), "{out}");

    // Outro PC restaura pelo comando restore.
    let b = base(t.path(), "B", "2000");
    fs::create_dir_all(t.path().join("B/save")).unwrap();
    let (r, _) = call("restore", &[], &b, "");
    assert_eq!(r.unwrap(), 0);
    assert!(fs::read_to_string(t.path().join("B/save/slot.sav")).unwrap().contains("novo"));
    // Sem journal sobrando.
    let (_, out) = call("status", &["--json"], &b, "");
    assert!(out.contains("\"head\":\"1000_A\""), "{out}");
}

#[test]
fn run_recusa_quando_outro_pc_esta_jogando() {
    let t = tempfile::tempdir().unwrap();
    let a = base(t.path(), "A", "1000");
    fs::create_dir_all(t.path().join("A/save")).unwrap();
    fs::write(t.path().join("A/save/s.sav"), "a").unwrap();
    fs::create_dir_all(t.path().join("B/save")).unwrap();
    fs::create_dir_all(t.path().join("nuvem")).unwrap();
    let ok: Vec<&str> = if cfg!(windows) { vec!["--", "cmd", "/C", "exit 0"] } else { vec!["--", "true"] };
    assert_eq!(call("run", &ok, &a, "").0.unwrap(), 0);
    // Simula A ainda jogando: trava de A fresca.
    sgs_core::lock::acquire(&t.path().join("nuvem/SyncGameSave/games/Jogo_ Teste"), "A", 1500).unwrap();
    let b = base(t.path(), "B", "1600");
    let (r, out) = call("run", &ok, &b, "");
    assert_eq!(r.unwrap(), 3, "{out}");
    assert!(out.contains("--force"));
}

#[test]
fn erros_claros() {
    assert!(sgs(&["run"], "").0.unwrap_err().contains("--sync-dir"));
    assert!(sgs(&["voar", "--sync-dir", "x", "--game", "g", "--save-dir", "s"], "").0.unwrap_err().contains("desconhecido"));
    assert_eq!(sgs(&["help"], "").0.unwrap(), 0);
}

/// O launcher abre o jogo em outro processo e fecha na hora; o save só é gravado 2 s depois.
#[cfg(windows)]
#[test]
fn run_espera_o_jogo_aberto_por_launcher_intermediario() {
    let t = tempfile::tempdir().unwrap();
    let a = base(t.path(), "A", "1000");
    fs::create_dir_all(t.path().join("A/save")).unwrap();
    fs::create_dir_all(t.path().join("nuvem")).unwrap();
    let file = t.path().join("A/save/slot.sav");
    let inner = format!("Start-Sleep -Seconds 2; Set-Content -Path '{}' -Value tardio", file.display());
    let launcher = format!(
        "Start-Process powershell -ArgumentList '-NoProfile','-Command',\"{}\" -WindowStyle Hidden",
        inner.replace('"', "'")
    );
    let extra = ["--quiet-secs", "1", "--", "powershell", "-NoProfile", "-Command", launcher.as_str()];
    let (r, out) = call("run", &extra, &a, "");
    assert_eq!(r.unwrap(), 0, "{out}");
    assert!(out.contains("Save enviado para a nuvem"), "o save tardio ficou de fora: {out}");
    assert!(fs::read_to_string(&file).unwrap().contains("tardio"));
}

/// O jogo deve abrir na pasta pedida (muitos jogos acham seus arquivos pelo diretório atual).
#[cfg(windows)]
#[test]
fn run_abre_o_jogo_na_pasta_de_trabalho() {
    let t = tempfile::tempdir().unwrap();
    let a = base(t.path(), "A", "1000");
    fs::create_dir_all(t.path().join("A/save")).unwrap();
    fs::create_dir_all(t.path().join("nuvem")).unwrap();
    let cwd = t.path().join("A/save");
    let cwd = cwd.to_string_lossy().into_owned();
    let extra = ["--cwd", cwd.as_str(), "--", "powershell", "-NoProfile", "-Command", "Set-Content -Path relativo.sav -Value x"];
    let (r, out) = call("run", &extra, &a, "");
    assert_eq!(r.unwrap(), 0, "{out}");
    assert!(t.path().join("A/save/relativo.sav").exists());
    assert!(out.contains("Save enviado para a nuvem"), "{out}");
}
