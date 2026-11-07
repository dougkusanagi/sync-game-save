//! Parar o jogo: `parar` encerra o processo aberto e `run` volta sem esperar o tempo de silêncio.
#![cfg(windows)]

use std::time::{Duration, Instant};

#[test]
fn parar_encerra_o_jogo_aberto() {
    let dir = tempfile::tempdir().unwrap();
    let exe = dir.path().join("jogo.exe");
    std::fs::copy(r"C:\Windows\System32\PING.EXE", &exe).unwrap();
    let prog = exe.to_string_lossy().to_string();

    let p = prog.clone();
    let t = std::thread::spawn(move || {
        // Espera o jogo aparecer na lista e pede para parar.
        let fim = Instant::now() + Duration::from_secs(5);
        while !sgs_core::process::parar(&p) && Instant::now() < fim {
            std::thread::sleep(Duration::from_millis(50));
        }
    });
    let inicio = Instant::now();
    // Sem parar, este ping duraria ~60 s; o silêncio final seria de 30 s.
    let r = sgs_core::process::run(&prog, &["-n".into(), "60".into(), "127.0.0.1".into()], None, Duration::from_secs(30));
    t.join().unwrap();
    assert!(r.is_ok());
    assert!(inicio.elapsed() < Duration::from_secs(10), "não parou: {:?}", inicio.elapsed());
    assert!(!sgs_core::process::parar(&prog), "jogo parado sai da lista");
}
