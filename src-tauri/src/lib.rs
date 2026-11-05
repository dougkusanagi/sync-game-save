//! Camada fina do Tauri: cada comando chama o `sgs-service`. Nada de regra de negócio aqui.

use serde::{Deserialize, Serialize};
use sgs_catalog::recognize::{Recognition, ScanResult};
use sgs_library::{Game, NewGame};
use sgs_service::{ChoiceDto, Play, PlayOptions, Service};
use std::path::PathBuf;
use std::sync::Arc;
use tauri::{AppHandle, Emitter, Manager, State};

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
}

#[tauri::command]
fn sobre(app: AppHandle) -> Sobre {
    let info = app.package_info();
    Sobre { nome: info.name.clone(), versao: info.version.to_string() }
}

#[derive(Serialize)]
struct Estado {
    pasta_sync: Option<String>,
    nome_pc: String,
    catalogo_pronto: bool,
    operacoes_interrompidas: Vec<String>,
}

#[tauri::command]
async fn estado(svc: Svc<'_>) -> Result<Estado, String> {
    blocking(&svc, |s| {
        Ok(Estado {
            pasta_sync: s.sync_dir()?.map(|p| p.display().to_string()),
            nome_pc: s.device_name()?,
            catalogo_pronto: s.catalog_path().exists(),
            operacoes_interrompidas: s.interrupted_operations(),
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
        quiet_secs: 5,
        skip_sync: o.sem_sincronizar,
    };
    let result = blocking(&svc, move |s| {
        let emit = |estado: &str| {
            let _ = app.emit("jogo-estado", EstadoJogo { id, estado: estado.to_string() });
        };
        let r = s.play(id, &opts, &emit);
        emit(if r.is_ok() { "pronto" } else { "erro" });
        r
    })
    .await;
    result
}

/// Caminho da capa em cache (ou nada). A interface carrega por `convertFileSrc`.
#[tauri::command]
async fn capa_do_jogo(svc: Svc<'_>, id: i64) -> Result<Option<String>, String> {
    blocking(&svc, move |s| Ok(s.game_art(id)?.map(|p| p.display().to_string()))).await
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
        .setup(|app| {
            let dir = app.path().app_data_dir()?;
            let svc = Service::open(&dir).map_err(|e| e.to_string())?;
            app.manage(Arc::new(svc));
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
            listar_jogos,
            adicionar_jogo,
            editar_jogo,
            remover_jogo,
            jogar,
            capa_do_jogo,
            versoes,
            restaurar_versao,
        ])
        .run(tauri::generate_context!())
        .expect("erro ao iniciar o aplicativo");
}
