use sgs_core::lock::{acquire, heartbeat, release, state, LockState, FRESH_SECS};
use sgs_core::paths::{resolve, to_logical, Env};
use std::path::PathBuf;

#[test]
fn trava_de_outro_pc() {
    let t = tempfile::tempdir().unwrap();
    let g = t.path();
    assert_eq!(state(g, "A", 100).unwrap(), LockState::Free);
    acquire(g, "A", 100).unwrap();
    assert_eq!(state(g, "A", 5000).unwrap(), LockState::Own);
    assert!(matches!(state(g, "B", 100 + FRESH_SECS - 1).unwrap(), LockState::HeldByOther(_)));
    assert!(matches!(state(g, "B", 100 + FRESH_SECS).unwrap(), LockState::Stale(_)));
    heartbeat(g, "A", 1000).unwrap();
    assert!(matches!(state(g, "B", 1000 + 60).unwrap(), LockState::HeldByOther(l) if l.started == 100));
}

#[test]
fn release_nao_apaga_trava_alheia() {
    let t = tempfile::tempdir().unwrap();
    acquire(t.path(), "A", 1).unwrap();
    release(t.path(), "B").unwrap();
    assert!(matches!(state(t.path(), "B", 2).unwrap(), LockState::HeldByOther(_)));
    release(t.path(), "A").unwrap();
    assert_eq!(state(t.path(), "B", 2).unwrap(), LockState::Free);
}

#[test]
fn caminho_logico_entre_pcs() {
    let pc1 = Env::new([
        ("home", PathBuf::from("C:/Users/ana")),
        ("winAppData", PathBuf::from("C:/Users/ana/AppData/Roaming")),
    ]);
    let pc2 = Env::new([
        ("home", PathBuf::from("D:/Usuarios/beto")),
        ("winAppData", PathBuf::from("D:/Usuarios/beto/AppData/Roaming")),
    ]);
    let real = PathBuf::from("C:/Users/ana/AppData/Roaming/Jogo Ç/saves");
    let logical = to_logical(&real, &pc1).unwrap();
    assert_eq!(logical, "<winAppData>/Jogo Ç/saves");
    assert_eq!(
        resolve(&logical, &pc2).unwrap(),
        PathBuf::from("D:/Usuarios/beto/AppData/Roaming/Jogo Ç/saves")
    );
}

#[test]
fn resolucao_recusa_marcador_desconhecido_e_ponto_ponto() {
    let env = Env::new([("base", PathBuf::from("C:/Games/X"))]);
    assert_eq!(resolve("<base>", &env), Some(PathBuf::from("C:/Games/X")));
    assert_eq!(resolve("<home>/a", &env), None);
    assert_eq!(resolve("<base>/../../Windows", &env), None);
    assert_eq!(to_logical(&PathBuf::from("E:/outro"), &env), None);
}
