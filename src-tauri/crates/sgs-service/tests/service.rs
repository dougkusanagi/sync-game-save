use sgs_core::paths::Env;
use sgs_library::NewGame;
use sgs_service::{ChoiceDto, Error, Play, PlayOptions, Service};
use std::fs;
use std::path::{Path, PathBuf};
use std::sync::atomic::{AtomicU64, Ordering};
use std::sync::Arc;

/// Um "PC": dados próprios, pasta de save própria e relógio compartilhado.
fn pc(root: &Path, name: &str, clock: &Arc<AtomicU64>) -> Service {
    let saves = root.join(name).join("saves");
    let c = clock.clone();
    let s = Service::with(
        &root.join(name).join("dados"),
        Box::new(move |_| Env::new([("meusaves", saves.clone())])),
        Box::new(move || c.fetch_add(10, Ordering::SeqCst)),
    )
    .unwrap();
    s.set_device_name(name).unwrap();
    s
}

fn game() -> NewGame {
    NewGame {
        name: "Jogo: Teste".into(),
        exe: powershell(),
        install_dir: None,
        catalog_id: None,
        save_paths: vec!["<meusaves>/Jogo".into(), "<meusaves>/Config".into()],
    }
}

fn powershell() -> String {
    "powershell".into()
}

/// Opções que fazem o "jogo" (um comando do PowerShell) escrever no save e sair com 0.
fn opts(cmd: &str) -> PlayOptions {
    PlayOptions {
        args: vec!["-NoProfile".into(), "-Command".into(), cmd.into()],
        quiet_secs: 0,
        ..Default::default()
    }
}

fn write_cmd(root: &Path, pc: &str, rel: &str, body: &str) -> String {
    let p = root.join(pc).join("saves").join(rel);
    format!("New-Item -ItemType Directory -Force -Path '{}' | Out-Null; Set-Content -Path '{}' -Value {body}", p.parent().unwrap().display(), p.display())
}

fn read(root: &Path, pc: &str, rel: &str) -> String {
    fs::read_to_string(root.join(pc).join("saves").join(rel)).unwrap().trim().to_string()
}

fn clock() -> Arc<AtomicU64> {
    Arc::new(AtomicU64::new(1000))
}

fn silent(_: &str) {}

#[test]
fn assistente_valida_a_pasta_e_nao_apaga_outros_pcs() {
    let t = tempfile::tempdir().unwrap();
    let s = pc(t.path(), "A", &clock());
    assert!(s.check_sync_dir(&t.path().join("nao-existe")).message.is_some());
    assert!(matches!(s.set_sync_dir(&t.path().join("nao-existe")), Err(Error::Other(_))));

    // Pasta que já tem dados de outro PC.
    let sync = t.path().join("nuvem");
    fs::create_dir_all(sync.join("SyncGameSave/games/Outro Jogo")).unwrap();
    fs::write(sync.join("SyncGameSave/devices.json"), r#"["PC-ANTIGO"]"#).unwrap();
    let check = s.set_sync_dir(&sync).unwrap();
    assert!(check.has_data && check.writable);
    assert_eq!(check.games, vec!["Outro Jogo"]);
    let devices = fs::read_to_string(sync.join("SyncGameSave/devices.json")).unwrap();
    assert!(devices.contains("PC-ANTIGO") && devices.contains("\"A\""), "{devices}");
    assert_eq!(s.sync_dir().unwrap(), Some(sync));
}

#[test]
fn erros_claros_antes_de_jogar() {
    let t = tempfile::tempdir().unwrap();
    let s = pc(t.path(), "A", &clock());
    let id = s.add_game(&game()).unwrap();
    assert!(matches!(s.play(id, &opts("exit 0"), &silent), Err(Error::NoSyncDir)));

    s.set_sync_dir(&{
        let d = t.path().join("nuvem");
        fs::create_dir_all(&d).unwrap();
        d
    })
    .unwrap();
    let sem_save = s.add_game(&NewGame { name: "Sem save".into(), save_paths: vec![], ..game() }).unwrap();
    assert!(matches!(s.play(sem_save, &opts("exit 0"), &silent), Err(Error::NoSavePath)));
    let ruim = s.add_game(&NewGame { name: "Ruim".into(), save_paths: vec!["<naoexiste>/x".into()], ..game() }).unwrap();
    assert!(matches!(s.play(ruim, &opts("exit 0"), &silent), Err(Error::UnresolvedPath(_))));

    // Nuvem sumiu: erro claro e a opção de jogar sem sincronizar funciona.
    fs::remove_dir_all(t.path().join("nuvem")).unwrap();
    assert!(matches!(s.play(id, &opts("exit 0"), &silent), Err(Error::SyncDirUnavailable(_))));
    let mut o = opts("exit 3");
    o.skip_sync = true;
    assert!(matches!(s.play(id, &o, &silent).unwrap(), Play::Finished { exit_code: 3, synced: false, .. }));
}

fn setup_two_pcs() -> (tempfile::TempDir, Service, Service, i64, i64) {
    let t = tempfile::tempdir().unwrap();
    let ck = clock();
    let (a, b) = (pc(t.path(), "A", &ck), pc(t.path(), "B", &ck));
    let sync = t.path().join("nuvem");
    fs::create_dir_all(&sync).unwrap();
    a.set_sync_dir(&sync).unwrap();
    b.set_sync_dir(&sync).unwrap();
    let (ia, ib) = (a.add_game(&game()).unwrap(), b.add_game(&game()).unwrap());
    (t, a, b, ia, ib)
}

#[test]
fn joga_em_um_pc_e_recebe_no_outro() {
    let (t, a, b, ia, ib) = setup_two_pcs();
    let cmd = format!("{}; {}", write_cmd(t.path(), "A", "Jogo/slot1.sav", "um"), write_cmd(t.path(), "A", "Config/op.ini", "opcoes"));
    let r = a.play(ia, &opts(&cmd), &silent).unwrap();
    assert!(matches!(r, Play::Finished { exit_code: 0, synced: true, .. }), "{r:?}");
    assert_eq!(a.snapshots(ia).unwrap().len(), 1);

    // PC B começa sem save: recebe os dois e só então abre o jogo.
    let r = b.play(ib, &opts("exit 0"), &silent).unwrap();
    assert!(matches!(r, Play::Finished { synced: true, .. }));
    assert_eq!(read(t.path(), "B", "Jogo/slot1.sav"), "um");
    assert_eq!(read(t.path(), "B", "Config/op.ini"), "opcoes");
    assert!(a.games().unwrap()[0].last_sync_at.is_some());
}

#[test]
fn conflito_pede_escolha_e_guarda_o_lado_descartado() {
    let (t, a, b, ia, ib) = setup_two_pcs();
    a.play(ia, &opts(&write_cmd(t.path(), "A", "Jogo/s.sav", "base")), &silent).unwrap();
    b.play(ib, &opts("exit 0"), &silent).unwrap();
    // Os dois mudam o save e só A sincroniza.
    a.play(ia, &opts(&write_cmd(t.path(), "A", "Jogo/s.sav", "de-A")), &silent).unwrap();
    let ruim = opts(&write_cmd(t.path(), "B", "Jogo/s.sav", "de-B"));
    // B jogou fora do launcher: muda o arquivo direto.
    fs::write(t.path().join("B/saves/Jogo/s.sav"), "de-B").unwrap();
    let r = b.play(ib, &opts("exit 0"), &silent).unwrap();
    assert_eq!(r, Play::Conflict);
    assert_eq!(read(t.path(), "B", "Jogo/s.sav"), "de-B", "nada muda antes da escolha");

    let mut o = ruim;
    o.args = vec!["-NoProfile".into(), "-Command".into(), "exit 0".into()];
    o.conflict = Some(ChoiceDto::Cloud);
    let r = b.play(ib, &o, &silent).unwrap();
    assert!(matches!(r, Play::Finished { .. }), "{r:?}");
    assert_eq!(read(t.path(), "B", "Jogo/s.sav"), "de-A");
    // O save descartado de B continua recuperável em backup.
    let mut found = false;
    for e in walk(&t.path().join("B/dados/state")) {
        if e.ends_with("s.sav") && fs::read_to_string(&e).unwrap().trim() == "de-B" {
            found = true;
        }
    }
    assert!(found, "backup do save descartado não encontrado");
}

fn walk(dir: &Path) -> Vec<PathBuf> {
    let mut out = vec![];
    if let Ok(rd) = fs::read_dir(dir) {
        for e in rd.flatten() {
            let p = e.path();
            if p.is_dir() {
                out.extend(walk(&p));
            } else {
                out.push(p);
            }
        }
    }
    out
}

#[test]
fn outro_pc_jogando_pede_confirmacao() {
    let (t, a, b, ia, ib) = setup_two_pcs();
    a.play(ia, &opts(&write_cmd(t.path(), "A", "Jogo/s.sav", "v1")), &silent).unwrap();
    // Simula A ainda jogando agora (relógio compartilhado perto do heartbeat).
    let game_dir = t.path().join("nuvem/SyncGameSave/games/Jogo_ Teste");
    sgs_core::lock::acquire(&game_dir, "A", 1_000_000).unwrap();
    let mut o = opts("exit 0");
    assert!(matches!(b.play(ib, &o, &silent).unwrap(), Play::Locked { pc } if pc == "A"));
    o.force = true;
    assert!(matches!(b.play(ib, &o, &silent).unwrap(), Play::Finished { .. }));
}

#[test]
fn restaurar_versao_antiga_com_backup_e_remover_nao_apaga_saves() {
    let (t, a, _b, ia, _ib) = setup_two_pcs();
    a.play(ia, &opts(&write_cmd(t.path(), "A", "Jogo/s.sav", "v1")), &silent).unwrap();
    a.play(ia, &opts(&write_cmd(t.path(), "A", "Jogo/s.sav", "v2")), &silent).unwrap();
    let snaps = a.snapshots(ia).unwrap();
    assert_eq!(snaps.len(), 2);
    let oldest = snaps.last().unwrap();
    let backup = a.restore_snapshot(ia, oldest).unwrap();
    assert_eq!(read(t.path(), "A", "Jogo/s.sav"), "v1");
    assert!(!walk(&backup).is_empty(), "o save anterior deveria estar em backup");
    assert!(a.interrupted_operations().is_empty());

    a.remove_game(ia).unwrap();
    assert!(a.games().unwrap().is_empty());
    assert_eq!(read(t.path(), "A", "Jogo/s.sav"), "v1");
    assert!(t.path().join("nuvem/SyncGameSave/games/Jogo_ Teste/head.json").exists());
}

#[test]
fn capa_usa_o_cache_e_nao_quebra_sem_catalogo() {
    let t = tempfile::tempdir().unwrap();
    let s = pc(t.path(), "A", &clock());
    let id = s.add_game(&game()).unwrap();
    // Sem catálogo nem cache: nada de erro, só sem capa.
    assert_eq!(s.game_art(id).unwrap(), None);
    // Com uma capa já em cache, devolve o arquivo sem ir à rede.
    let cached = t.path().join("A/dados/art");
    fs::create_dir_all(&cached).unwrap();
    fs::write(cached.join(format!("{id}.jpg")), [0xFF, 0xD8, 0xFF]).unwrap();
    assert_eq!(s.game_art(id).unwrap(), Some(cached.join(format!("{id}.jpg"))));
    assert_eq!(sgs_service::steam_header_url(241600), "https://cdn.cloudflare.steamstatic.com/steam/apps/241600/header.jpg");
}

#[test]
fn heartbeat_da_trava_avanca_enquanto_o_jogo_roda() {
    let (t, a, _b, ia, _ib) = setup_two_pcs();
    let game_dir = t.path().join("nuvem/SyncGameSave/games/Jogo_ Teste");
    let mut o = opts("Start-Sleep -Seconds 4");
    o.heartbeat_secs = 1;
    let beats = std::thread::scope(|sc| {
        let watcher = sc.spawn(|| {
            let mut seen = std::collections::BTreeSet::new();
            for _ in 0..40 {
                if let Ok(Some(l)) = sgs_core::lock::read(&game_dir) {
                    seen.insert(l.heartbeat);
                }
                std::thread::sleep(std::time::Duration::from_millis(150));
            }
            seen
        });
        let r = a.play(ia, &o, &silent).unwrap();
        assert!(matches!(r, Play::Finished { .. }));
        watcher.join().unwrap()
    });
    assert!(beats.len() >= 2, "o heartbeat deveria mudar durante o jogo: {beats:?}");
    assert!(sgs_core::lock::read(&game_dir).unwrap().is_none(), "trava solta ao fechar");
}
