//! A força só fica disponível após o pedido normal e a espera.
#![cfg(windows)]
use std::time::{Duration, Instant};

#[test]
fn janela_fecha_normalmente_antes_do_prazo() {
    let dir = tempfile::tempdir().unwrap();
    let exe = dir.path().join("launcher.exe");
    std::fs::copy(r"C:\Windows\System32\cmd.exe", &exe).unwrap();
    let script = dir.path().join("janela.ps1");
    std::fs::write(&script, "Add-Type -AssemblyName System.Windows.Forms\n$f = New-Object System.Windows.Forms.Form\n$f.Add_Shown({ [IO.File]::WriteAllText('pronto', 'sim') })\n$f.Add_FormClosing({ Start-Process ping.exe -ArgumentList '-n','4','127.0.0.1' -WindowStyle Hidden })\n[void]$f.ShowDialog()\n").unwrap();
    let prog = exe.to_string_lossy().to_string();
    let p = prog.clone();
    let cwd = dir.path().to_path_buf();
    let jogo = std::thread::spawn(move || sgs_core::process::run(&p, &["/c".into(), "powershell.exe".into(), "-NoProfile".into(), "-STA".into(), "-File".into(), script.to_string_lossy().into()], Some(&cwd), Duration::from_millis(300)));
    let limite = Instant::now() + Duration::from_secs(15);
    while !dir.path().join("pronto").exists() && Instant::now() < limite { std::thread::sleep(Duration::from_millis(50)); }
    assert!(dir.path().join("pronto").exists(), "janela não iniciou");
    let inicio = Instant::now();
    assert_eq!(sgs_core::process::parar(&prog).unwrap(), Some(false));
    assert!(inicio.elapsed() >= Duration::from_secs(2), "voltou antes de o filho terminar");
    assert!(inicio.elapsed() < Duration::from_secs(10));
    assert!(jogo.join().unwrap().is_ok());
}

#[test]
fn janela_que_recusa_fechamento_so_termina_com_forca_confirmada() {
    let dir = tempfile::tempdir().unwrap();
    let exe = dir.path().join("launcher.exe");
    std::fs::copy(r"C:\Windows\System32\cmd.exe", &exe).unwrap();
    let script = dir.path().join("janela.ps1");
    std::fs::write(&script, "Add-Type -AssemblyName System.Windows.Forms\n$f = New-Object System.Windows.Forms.Form\n$f.Add_Shown({ [IO.File]::WriteAllText('pronto', 'sim') })\n$f.Add_FormClosing({ param($s,$e) [IO.File]::WriteAllText('recusou', 'sim'); $e.Cancel = $true })\n[void]$f.ShowDialog()\n").unwrap();
    let prog = exe.to_string_lossy().to_string();
    let p = prog.clone();
    let cwd = dir.path().to_path_buf();
    let jogo = std::thread::spawn(move || sgs_core::process::run(&p, &["/c".into(), "powershell.exe".into(), "-NoProfile".into(), "-STA".into(), "-File".into(), script.to_string_lossy().into()], Some(&cwd), Duration::from_millis(300)));
    let limite = Instant::now() + Duration::from_secs(15);
    while !dir.path().join("pronto").exists() && Instant::now() < limite { std::thread::sleep(Duration::from_millis(50)); }
    assert!(dir.path().join("pronto").exists());
    let inicio = Instant::now();
    assert_eq!(sgs_core::process::parar(&prog).unwrap(), Some(true));
    assert!(inicio.elapsed() >= Duration::from_secs(10));
    assert!(dir.path().join("recusou").exists());
    assert!(!jogo.is_finished());
    assert!(sgs_core::process::forcar_parada(&prog).unwrap());
    assert!(jogo.join().unwrap().is_ok());
}

#[test]
fn jogo_sem_janela_espera_antes_de_permitir_forca() {
    let dir = tempfile::tempdir().unwrap();
    let exe = dir.path().join("jogo.exe");
    std::fs::copy(r"C:\Windows\System32\PING.EXE", &exe).unwrap();
    let prog = exe.to_string_lossy().to_string();
    let p = prog.clone();
    let jogo = std::thread::spawn(move || sgs_core::process::run(&p, &["-n".into(), "60".into(), "127.0.0.1".into()], None, Duration::from_millis(300)));
    let limite = Instant::now() + Duration::from_secs(5);
    loop {
        match sgs_core::process::forcar_parada(&prog) {
            Err(_) => break,
            Ok(false) if Instant::now() < limite => std::thread::sleep(Duration::from_millis(50)),
            outro => panic!("estado inesperado: {outro:?}"),
        }
    }
    let inicio = Instant::now();
    let p = prog.clone();
    let pedido = std::thread::spawn(move || sgs_core::process::parar(&p));
    std::thread::sleep(Duration::from_millis(300));
    assert!(sgs_core::process::forcar_parada(&prog).is_err());
    assert_eq!(pedido.join().unwrap().unwrap(), Some(true));
    assert!(inicio.elapsed() >= Duration::from_secs(10));
    assert!(!jogo.is_finished());
    assert!(sgs_core::process::forcar_parada(&prog).unwrap());
    assert!(jogo.join().unwrap().is_ok());
    assert_eq!(sgs_core::process::parar(&prog).unwrap(), None);
}
