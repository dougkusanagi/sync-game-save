//! Jogo da Steam: o executável aberto fecha e a Steam abre o jogo de novo (processo que não é filho).
//! O acompanhamento precisa esperar o processo que roda da pasta do jogo.
#![cfg(windows)]

use sgs_core::process::{Beat, Tracker};
use std::time::{Duration, Instant};

#[test]
fn acompanha_descendente_de_launcher_curto_fora_da_pasta() {
    let dir = tempfile::tempdir().unwrap();
    let launcher = dir.path().join("launcher.exe");
    std::fs::copy(std::env::current_exe().unwrap(), &launcher).unwrap();
    std::fs::write(dir.path().join("filho.txt"), std::env::current_exe().unwrap().to_str().unwrap()).unwrap();
    let inicio = Instant::now();
    let args = vec!["--exact".into(), "auxiliar_launcher".into(), "--nocapture".into()];
    assert_eq!(sgs_core::process::run(launcher.to_str().unwrap(), &args, None, Duration::from_millis(300)).unwrap(), 0);
    assert!(dir.path().join("filho-terminou").exists(), "o filho não terminou antes do retorno");
    assert!(inicio.elapsed() >= Duration::from_millis(1500), "parou de esperar cedo: {:?}", inicio.elapsed());
}

#[test]
fn auxiliar_launcher() {
    let Ok(exe) = std::fs::read_to_string("filho.txt") else { return };
    std::process::Command::new(exe).args(["--exact", "auxiliar_filho", "--nocapture"]).spawn().unwrap();
}

#[test]
fn auxiliar_filho() {
    if !std::path::Path::new("filho.txt").exists() { return }
    std::thread::sleep(Duration::from_secs(2));
    std::fs::write("filho-terminou", "ok").unwrap();
}

#[test]
fn espera_processo_da_pasta_do_jogo_mesmo_sem_ser_filho() {
    let dir = tempfile::tempdir().unwrap();
    let jogo = dir.path().join("Jogo");
    std::fs::create_dir_all(&jogo).unwrap();
    let exe = jogo.join("jogo.exe");
    std::fs::copy(r"C:\Windows\System32\PING.EXE", &exe).unwrap();
    // "Launcher" que sai na hora (o pid dele some logo).
    let launcher = std::process::Command::new("cmd").args(["/c", "exit"]).spawn().unwrap();
    let pid = launcher.id();
    // O jogo de verdade, aberto por outro processo (aqui, pelo teste), roda ~2 s.
    let mut real = std::process::Command::new(&exe).args(["-n", "3", "127.0.0.1"]).stdout(std::process::Stdio::null()).spawn().unwrap();
    let inicio = Instant::now();
    let mut nada = || {};
    let mut beat = Beat::new(Duration::from_secs(3600), &mut nada);
    Tracker::with_dir(pid, Some(&jogo)).unwrap().wait_until_gone(Duration::from_millis(300), &mut beat);
    assert!(inicio.elapsed() >= Duration::from_millis(1500), "parou de esperar cedo: {:?}", inicio.elapsed());
    let _ = real.wait();
}
