//! Camada fina do Tauri: cada comando chama o `sgs-service`. Nada de regra de negócio aqui.

mod controle;

use serde::{Deserialize, Serialize};
use sgs_catalog::recognize::{Recognition, ScanResult};
use sgs_library::{Game, NewGame};
use sgs_service::{ChoiceDto, Play, PlayOptions, Service};
use std::path::PathBuf;
use std::sync::Arc;
use tauri::menu::{Menu, MenuItem};
use tauri::tray::{MouseButton, MouseButtonState, TrayIconBuilder, TrayIconEvent};
use tauri::{AppHandle, Emitter, Manager, State, WindowEvent};
use tauri_plugin_notification::NotificationExt;

type Svc<'a> = State<'a, Arc<Service>>;

/// Roda trabalho que bloqueia (disco, rede, jogo aberto) fora da thread da interface.
async fn blocking<T: Send + 'static>(
    svc: &Arc<Service>,
    f: impl FnOnce(&Service) -> sgs_service::Result<T> + Send + 'static,
) -> Result<T, String> {
    let svc = svc.clone();
    tauri::async_runtime::spawn_blocking(move || f(&svc))
        .await
        .map_err(|_| "A operação foi interrompida de forma inesperada.".to_string())?
        .map_err(|e| e.to_string())
}

#[derive(Serialize)]
struct Sobre {
    nome: String,
    versao: String,
    build: String,
}

#[tauri::command]
fn sobre(app: AppHandle) -> Sobre {
    let info = app.package_info();
    Sobre {
        nome: info.name.clone(),
        versao: info.version.to_string(),
        build: option_env!("SGS_BUILD").unwrap_or("local").to_string(),
    }
}

#[derive(Serialize)]
struct Estado {
    pasta_sync: Option<String>,
    nome_pc: String,
    catalogo_pronto: bool,
    operacoes_interrompidas: Vec<String>,
    steam_configurada: bool,
    steam_id: Option<String>,
    steamgriddb_configurada: bool,
}

#[tauri::command]
async fn estado(svc: Svc<'_>) -> Result<Estado, String> {
    blocking(&svc, |s| {
        Ok(Estado {
            pasta_sync: s.sync_dir()?.map(|p| p.display().to_string()),
            nome_pc: s.device_name()?,
            catalogo_pronto: s.catalog_path().exists(),
            operacoes_interrompidas: s.interrupted_operations(),
            steam_configurada: s.steam_configured(),
            steam_id: s.steam_id()?,
            steamgriddb_configurada: s.steamgriddb_configured(),
        })
    })
    .await
}

#[tauri::command]
async fn verificar_pasta_sync(svc: Svc<'_>, pasta: String) -> Result<sgs_service::SyncDirCheck, String> {
    blocking(&svc, move |s| Ok(s.check_sync_dir(&PathBuf::from(pasta)))).await
}

#[tauri::command]
async fn definir_pasta_sync(svc: Svc<'_>, pasta: String) -> Result<sgs_service::SyncDirCheck, String> {
    blocking(&svc, move |s| s.set_sync_dir(&PathBuf::from(pasta))).await
}

#[tauri::command]
async fn definir_nome_pc(svc: Svc<'_>, nome: String) -> Result<(), String> {
    blocking(&svc, move |s| s.set_device_name(&nome)).await
}

#[tauri::command]
async fn atualizar_catalogo(svc: Svc<'_>) -> Result<usize, String> {
    blocking(&svc, |s| Ok(s.update_catalog()?.games)).await
}

#[tauri::command]
async fn reconhecer_pasta(svc: Svc<'_>, pasta: String) -> Result<Recognition, String> {
    blocking(&svc, move |s| s.recognize_folder(&PathBuf::from(pasta))).await
}

#[tauri::command]
async fn varrer_pasta(svc: Svc<'_>, pasta: String, profundidade: u32) -> Result<ScanResult, String> {
    blocking(&svc, move |s| s.scan_library_folder(&PathBuf::from(pasta), profundidade)).await
}

#[tauri::command]
async fn converter_caminho(svc: Svc<'_>, caminho: String) -> Result<String, String> {
    blocking(&svc, move |s| Ok(s.logical_path(&PathBuf::from(caminho)))).await
}

#[tauri::command]
async fn listar_jogos(svc: Svc<'_>) -> Result<Vec<Game>, String> {
    blocking(&svc, |s| s.games()).await
}

#[tauri::command]
async fn adicionar_jogo(svc: Svc<'_>, jogo: NewGame) -> Result<i64, String> {
    blocking(&svc, move |s| s.add_game(&jogo)).await
}

#[tauri::command]
async fn editar_jogo(svc: Svc<'_>, id: i64, jogo: NewGame) -> Result<(), String> {
    blocking(&svc, move |s| s.update_game(id, &jogo)).await
}

#[tauri::command]
async fn remover_jogo(svc: Svc<'_>, id: i64) -> Result<(), String> {
    blocking(&svc, move |s| s.remove_game(id)).await
}

#[derive(Deserialize, Default)]
struct OpcoesJogar {
    #[serde(default)]
    forcar: bool,
    /// "local" ou "nuvem", para resolver um conflito.
    #[serde(default)]
    conflito: Option<String>,
    #[serde(default)]
    sem_sincronizar: bool,
}

#[derive(Serialize, Clone)]
struct EstadoJogo {
    id: i64,
    estado: String,
}

/// Abre o jogo com sync e emite `jogo-estado` a cada etapa. Só volta quando o jogo fecha.
static ABERTOS: std::sync::Mutex<std::collections::BTreeSet<i64>> = std::sync::Mutex::new(std::collections::BTreeSet::new());

#[tauri::command]
async fn jogar(app: AppHandle, svc: Svc<'_>, id: i64, opcoes: Option<OpcoesJogar>) -> Result<Play, String> {
    let o = opcoes.unwrap_or_default();
    let opts = PlayOptions {
        force: o.forcar,
        conflict: match o.conflito.as_deref() {
            Some("local") => Some(ChoiceDto::Local),
            Some("nuvem") => Some(ChoiceDto::Cloud),
            _ => None,
        },
        args: vec![],
        // Jogos da Steam fecham e voltam pela Steam: a espera cobre esse intervalo.
        quiet_secs: 15,
        skip_sync: o.sem_sincronizar,
        heartbeat_secs: 0,
    };
    // Um jogo de cada vez: apertar Jogar de novo não abre outra cópia.
    if !ABERTOS.lock().map_err(|e| e.to_string())?.insert(id) {
        return Err("Este jogo já está aberto.".into());
    }
    struct Libera(i64);
    impl Drop for Libera {
        fn drop(&mut self) {
            if let Ok(mut a) = ABERTOS.lock() {
                a.remove(&self.0);
            }
        }
    }
    let _libera = Libera(id);
    let notify = app.clone();
    let result = blocking(&svc, move |s| {
        let emit = |estado: &str| {
            let _ = app.emit("jogo-estado", EstadoJogo { id, estado: estado.to_string() });
        };
        let r = s.play(id, &opts, &emit);
        emit(if r.is_ok() { "pronto" } else { "erro" });
        r
    })
    .await;
    if matches!(result, Ok(Play::Finished { .. })) {
        let svc = svc.inner().clone();
        tauri::async_runtime::spawn(async move {
            let _ = tauri::async_runtime::spawn_blocking(move || {
                let _ = svc.achievements(id, true);
            })
            .await;
        });
    }
    match &result {
        // Deu tudo certo: sem notificação. Só avisa quando o save não foi sincronizado.
        Ok(Play::Finished { notice, synced: false, .. }) => notificar(&notify, "Sync Game Save", notice),
        Ok(Play::Conflict) => notificar(&notify, "Sync Game Save", "O save deste PC e o da pasta de sincronização mudaram. Abra o app para escolher qual manter."),
        Err(msg) => notificar(&notify, "Sync Game Save", msg),
        _ => {}
    }
    result
}

/// Fecha o jogo aberto pelo app. O save é sincronizado quando o jogo termina de fechar.
#[tauri::command]
async fn parar_jogo(svc: Svc<'_>, id: i64) -> Result<Option<bool>, String> {
    blocking(&svc, move |s| s.stop_game(id)).await
}

#[tauri::command]
async fn forcar_parada_jogo(svc: Svc<'_>, id: i64) -> Result<bool, String> {
    blocking(&svc, move |s| s.force_stop_game(id)).await
}

fn notificar(app: &AppHandle, titulo: &str, texto: &str) {
    let _ = app.notification().builder().title(titulo).body(texto).show();
}

fn mostrar_janela(app: &AppHandle) {
    if let Some(w) = app.get_webview_window("main") {
        let _ = w.show();
        let _ = w.unminimize();
        let _ = w.set_focus();
    }
}

/// Caminho da capa em cache (ou nada). A interface carrega por `convertFileSrc`.
#[tauri::command]
async fn capa_do_jogo(svc: Svc<'_>, id: i64) -> Result<Option<String>, String> {
    blocking(&svc, move |s| Ok(s.game_art(id)?.map(|p| p.display().to_string()))).await
}

/// Arte de um tipo (`header`, `hero`, `logo` ou `cover`) em cache, ou nada.
#[tauri::command]
async fn arte_do_jogo(svc: Svc<'_>, id: i64, tipo: String) -> Result<Option<String>, String> {
    let kind = sgs_service::ArtKind::parse(&tipo).ok_or_else(|| format!("Tipo de arte desconhecido: {tipo}"))?;
    blocking(&svc, move |s| Ok(s.game_art_kind(id, kind)?.map(|p| p.display().to_string()))).await
}

#[tauri::command]
async fn definir_credenciais_steam(svc: Svc<'_>, chave: String, steam_id: String) -> Result<(), String> {
    blocking(&svc, move |s| s.set_steam_credentials(&chave, &steam_id)).await
}

#[tauri::command]
async fn conquistas(svc: Svc<'_>, id: i64, forcar: Option<bool>) -> Result<sgs_service::AchievementsView, String> {
    blocking(&svc, move |s| s.achievements(id, forcar.unwrap_or(false))).await
}

#[tauri::command]
async fn definir_chave_steamgriddb(svc: Svc<'_>, chave: String) -> Result<(), String> {
    blocking(&svc, move |s| s.set_steamgriddb_key(&chave)).await
}

#[tauri::command]
async fn conquistas_hub(svc: Svc<'_>) -> Result<sgs_service::HubAchievements, String> {
    blocking(&svc, |s| s.hub_achievements()).await
}

#[tauri::command]
async fn temas(svc: Svc<'_>) -> Result<sgs_service::ThemeList, String> {
    blocking(&svc, |s| Ok(s.themes())).await
}

#[tauri::command]
async fn definir_tema(svc: Svc<'_>, id: String) -> Result<sgs_service::ThemeList, String> {
    blocking(&svc, move |s| s.set_theme(&id)).await
}

#[tauri::command]
async fn versoes(svc: Svc<'_>, id: i64) -> Result<Vec<String>, String> {
    blocking(&svc, move |s| s.snapshots(id)).await
}

#[tauri::command]
async fn restaurar_versao(svc: Svc<'_>, id: i64, versao: String) -> Result<String, String> {
    blocking(&svc, move |s| s.restore_snapshot(id, &versao).map(|p| p.display().to_string())).await
}

pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_dialog::init())
        .plugin(tauri_plugin_notification::init())
        .plugin(tauri_plugin_process::init())
        .plugin(tauri_plugin_updater::Builder::new().build())
        .on_window_event(|window, event| {
            // Fechar a janela só esconde o app na bandeja; ele continua cuidando dos saves.
            match event {
                WindowEvent::CloseRequested { api, .. } => {
                    api.prevent_close();
                    let _ = window.hide();
                }
                WindowEvent::Focused(f) => controle::definir_foco(*f),
                _ => {}
            }
        })
        .setup(|app| {
            // `SGS_DATA_DIR` isola os dados (usado nos testes do app real); sem ele, vale a pasta de dados do app.
            let dir = match std::env::var_os("SGS_DATA_DIR") {
                Some(d) => PathBuf::from(d),
                None => app.path().app_data_dir()?,
            };
            let svc = Service::open(&dir).map_err(|e| e.to_string())?;
            // A interface só pode ler as artes baixadas (pasta `art`), não o resto dos dados.
            let art = dir.join("art");
            let _ = std::fs::create_dir_all(&art);
            app.asset_protocol_scope().allow_directory(&art, false)?;
            let svc = Arc::new(svc);
            app.manage(svc.clone());
            // Atualização periódica com cache de 10 minutos; rede fora do ar
            // não afeta a interface nem o restante do serviço.
            tauri::async_runtime::spawn(async move {
                loop {
                    let svc = svc.clone();
                    let _ = tauri::async_runtime::spawn_blocking(move || svc.refresh_achievements()).await;
                    tokio::time::sleep(std::time::Duration::from_secs(60 * 60)).await;
                }
            });
            controle::iniciar(app.handle().clone());

            let abrir = MenuItem::with_id(app, "abrir", "Abrir Sync Game Save", true, None::<&str>)?;
            let sair = MenuItem::with_id(app, "sair", "Sair", true, None::<&str>)?;
            let menu = Menu::with_items(app, &[&abrir, &sair])?;
            let mut tray = TrayIconBuilder::new()
                .tooltip("Sync Game Save")
                .menu(&menu)
                .show_menu_on_left_click(false)
                .on_menu_event(|app, event| match event.id.as_ref() {
                    "abrir" => mostrar_janela(app),
                    "sair" => app.exit(0),
                    _ => {}
                })
                .on_tray_icon_event(|tray, event| {
                    if let TrayIconEvent::Click { button: MouseButton::Left, button_state: MouseButtonState::Up, .. } = event {
                        mostrar_janela(tray.app_handle());
                    }
                });
            if let Some(icon) = app.default_window_icon() {
                tray = tray.icon(icon.clone());
            }
            tray.build(app)?;
            Ok(())
        })
        .invoke_handler(tauri::generate_handler![
            sobre,
            estado,
            verificar_pasta_sync,
            definir_pasta_sync,
            definir_nome_pc,
            atualizar_catalogo,
            reconhecer_pasta,
            varrer_pasta,
            converter_caminho,
            listar_jogos,
            adicionar_jogo,
            editar_jogo,
            remover_jogo,
            jogar,
            parar_jogo,
            forcar_parada_jogo,
            capa_do_jogo,
            arte_do_jogo,
            definir_credenciais_steam,
            conquistas,
            conquistas_hub,
            definir_chave_steamgriddb,
            temas,
            definir_tema,
            versoes,
            restaurar_versao,
        ])
        .run(tauri::generate_context!())
        .expect("erro ao iniciar o aplicativo");
}
