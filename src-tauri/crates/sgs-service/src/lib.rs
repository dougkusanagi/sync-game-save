//! Serviço que a interface chama: biblioteca, catálogo, assistente de primeiro uso,
//! abrir/fechar jogo com sync e restauração de versões. Não depende do Tauri.

use serde::Serialize;
use sgs_catalog::recognize::{recognize, scan_folder, Recognition, ScanResult};
use sgs_catalog::Catalog;
use sgs_core::engine::{Before, Choice, Ctx};
use sgs_core::index::Root;
use sgs_core::paths::{resolve, Env};
use sgs_core::{journal, lock, snapshot};
use sgs_library::{Game, Library, NewGame};
use std::path::{Path, PathBuf};
use std::sync::Mutex;
use std::time::Duration;

/// Erros já escritos para o usuário final.
#[derive(Debug, thiserror::Error)]
pub enum Error {
    #[error("Escolha primeiro a pasta de sincronização, em Configurações.")]
    NoSyncDir,
    #[error("A pasta de sincronização ({0}) não está disponível. Confira se o OneDrive ou o serviço de nuvem está ativo, ou jogue sem sincronizar.")]
    SyncDirUnavailable(String),
    #[error("Este jogo ainda não tem uma pasta de save. Indique onde ele guarda o progresso para poder sincronizar.")]
    NoSavePath,
    #[error("Não consegui encontrar a pasta de save \"{0}\" neste PC.")]
    UnresolvedPath(String),
    #[error("O catálogo de jogos ainda não foi baixado. Atualize o catálogo em Configurações.")]
    NoCatalog,
    #[error("{0}")]
    Other(String),
}

impl From<sgs_core::Error> for Error {
    fn from(e: sgs_core::Error) -> Self {
        Error::Other(format!("Não foi possível concluir a operação com o save: {e}"))
    }
}
impl From<sgs_library::Error> for Error {
    fn from(e: sgs_library::Error) -> Self {
        Error::Other(e.to_string())
    }
}
impl From<sgs_catalog::Error> for Error {
    fn from(e: sgs_catalog::Error) -> Self {
        Error::Other(format!("Problema com o catálogo de jogos: {e}"))
    }
}

pub type Result<T> = std::result::Result<T, Error>;

/// Resultado de conferir a pasta de sync no assistente de primeiro uso.
#[derive(Debug, Serialize, PartialEq, Eq)]
pub struct SyncDirCheck {
    pub exists: bool,
    pub writable: bool,
    /// Já tem dados do Sync Game Save (de outro PC).
    pub has_data: bool,
    pub games: Vec<String>,
    /// A pasta parece estar só na nuvem (arquivos sob demanda), sem cópia local.
    pub message: Option<String>,
}

#[derive(Debug, Serialize, PartialEq, Eq)]
#[serde(tag = "kind", rename_all = "snake_case")]
pub enum Play {
    /// Os dois lados mudaram: pedir ao usuário e chamar de novo com a escolha.
    Conflict,
    /// Outro PC está jogando: pedir confirmação e chamar de novo com `force`.
    Locked { pc: String },
    Finished { exit_code: i32, synced: bool, notice: String },
}

#[derive(Debug, Clone, Default)]
pub struct PlayOptions {
    pub force: bool,
    pub conflict: Option<ChoiceDto>,
    pub args: Vec<String>,
    pub quiet_secs: u64,
    /// Joga sem tocar na pasta de sync.
    pub skip_sync: bool,
    /// Intervalo do heartbeat da trava; 0 usa o padrão (2 minutos).
    pub heartbeat_secs: u64,
}

impl PlayOptions {
    fn heartbeat_every(&self) -> Duration {
        Duration::from_secs(if self.heartbeat_secs == 0 { 120 } else { self.heartbeat_secs })
    }
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ChoiceDto {
    Local,
    Cloud,
}

type EnvFn = Box<dyn Fn(Option<PathBuf>) -> Env + Send + Sync>;

pub struct Service {
    lib: Mutex<Library>,
    data_dir: PathBuf,
    env: EnvFn,
    now: Box<dyn Fn() -> u64 + Send + Sync>,
}

fn slug(name: &str) -> String {
    let s: String = name
        .chars()
        .map(|c| if "<>:\"/\\|?*".contains(c) || c.is_control() { '_' } else { c })
        .collect();
    s.trim().trim_end_matches('.').to_string()
}

impl Service {
    pub fn open(data_dir: &Path) -> Result<Self> {
        Self::with(data_dir, Box::new(sgs_core::env::system_env), Box::new(unix_now))
    }

    pub fn with(data_dir: &Path, env: EnvFn, now: Box<dyn Fn() -> u64 + Send + Sync>) -> Result<Self> {
        std::fs::create_dir_all(data_dir).map_err(|e| Error::Other(format!("Não consegui criar a pasta de dados: {e}")))?;
        let lib = Library::open(&data_dir.join("library.db"))?;
        Ok(Service { lib: Mutex::new(lib), data_dir: data_dir.to_path_buf(), env, now })
    }

    fn lib(&self) -> std::sync::MutexGuard<'_, Library> {
        self.lib.lock().unwrap_or_else(|p| p.into_inner())
    }

    // ---- Configurações e assistente de primeiro uso ----

    pub fn sync_dir(&self) -> Result<Option<PathBuf>> {
        Ok(self.lib().setting("sync_dir")?.map(PathBuf::from))
    }

    pub fn device_name(&self) -> Result<String> {
        Ok(self.lib().setting("device")?.unwrap_or_else(|| {
            std::env::var("COMPUTERNAME").or_else(|_| std::env::var("HOSTNAME")).unwrap_or_else(|_| "PC".into())
        }))
    }

    pub fn set_device_name(&self, name: &str) -> Result<()> {
        let name = name.trim();
        if name.is_empty() {
            return Err(Error::Other("O nome deste PC não pode ficar vazio.".into()));
        }
        Ok(self.lib().set_setting("device", name)?)
    }

    /// Confere a pasta antes de usar: existe, dá para gravar e ler, e já tem dados de outro PC?
    pub fn check_sync_dir(&self, dir: &Path) -> SyncDirCheck {
        let exists = dir.is_dir();
        let probe = dir.join(".sgs-teste");
        let writable = exists && std::fs::write(&probe, b"ok").is_ok() && std::fs::read(&probe).is_ok_and(|b| b == b"ok");
        let _ = std::fs::remove_file(&probe);
        let games_dir = dir.join("SyncGameSave").join("games");
        let games: Vec<String> = std::fs::read_dir(&games_dir)
            .map(|rd| rd.flatten().filter(|e| e.path().is_dir()).map(|e| e.file_name().to_string_lossy().into_owned()).collect())
            .unwrap_or_default();
        let has_data = dir.join("SyncGameSave").join("devices.json").exists() || !games.is_empty();
        let message = if !exists {
            Some("Esta pasta não existe.".into())
        } else if !writable {
            Some("Não consegui gravar nesta pasta. Escolha outra ou confira as permissões.".into())
        } else {
            None
        };
        SyncDirCheck { exists, writable, has_data, games, message }
    }

    /// Guarda a pasta de sync e registra este PC. Nada existente é alterado ou movido.
    pub fn set_sync_dir(&self, dir: &Path) -> Result<SyncDirCheck> {
        let check = self.check_sync_dir(dir);
        if let Some(m) = &check.message {
            return Err(Error::Other(m.clone()));
        }
        let root = dir.join("SyncGameSave");
        std::fs::create_dir_all(root.join("games")).map_err(|e| Error::Other(format!("Não consegui preparar a pasta: {e}")))?;
        self.register_device(&root)?;
        self.lib().set_setting("sync_dir", &dir.to_string_lossy())?;
        Ok(check)
    }

    /// Acrescenta este PC em `devices.json` sem apagar os outros.
    fn register_device(&self, root: &Path) -> Result<()> {
        let path = root.join("devices.json");
        let mut names: Vec<String> = std::fs::read(&path).ok().and_then(|b| serde_json::from_slice(&b).ok()).unwrap_or_default();
        let me = self.device_name()?;
        if !names.contains(&me) {
            names.push(me);
        }
        let json = serde_json::to_vec_pretty(&names).unwrap_or_default();
        std::fs::write(&path, json).map_err(|e| Error::Other(format!("Não consegui registrar este PC: {e}")))
    }

    // ---- Catálogo e reconhecimento ----

    pub fn catalog_path(&self) -> PathBuf {
        self.data_dir.join("catalog.db")
    }

    /// Baixa o manifesto (se mudou) e remonta o catálogo.
    pub fn update_catalog(&self) -> Result<sgs_catalog::Stats> {
        let manifest = self.data_dir.join("manifest.yaml");
        let changed = sgs_catalog::fetch(&manifest)?;
        if !changed && self.catalog_path().exists() {
            return Ok(sgs_catalog::Stats::default());
        }
        Ok(sgs_catalog::build(&manifest, &self.catalog_path())?)
    }

    fn catalog(&self) -> Result<Catalog> {
        if !self.catalog_path().exists() {
            return Err(Error::NoCatalog);
        }
        Ok(Catalog::open(&self.catalog_path())?)
    }

    /// Sem o catálogo baixado, devolve só o que dá para saber pela pasta (nome e executável),
    /// e o usuário indica o save à mão.
    pub fn recognize_folder(&self, dir: &Path) -> Result<Recognition> {
        let env = (self.env)(Some(dir.to_path_buf()));
        match self.catalog() {
            Ok(cat) => Ok(recognize(&cat, dir, &env, cfg!(windows))?),
            Err(Error::NoCatalog) => {
                let folder = dir.file_name().map(|n| n.to_string_lossy().into_owned()).unwrap_or_default();
                let cleaned = sgs_catalog::recognize::clean_folder_name(&folder);
                Ok(Recognition {
                    name: if cleaned.is_empty() { folder } else { cleaned },
                    catalog_id: None,
                    confidence: sgs_catalog::recognize::Confidence::None,
                    exe: sgs_catalog::recognize::guess_exe(dir),
                    save_candidates: vec![],
                    alternatives: vec![],
                })
            }
            Err(e) => Err(e),
        }
    }

    /// Troca um caminho real por um caminho com marcador (portátil entre PCs), se houver marcador.
    pub fn logical_path(&self, path: &Path) -> String {
        let env = (self.env)(None);
        sgs_core::paths::to_logical(path, &env).unwrap_or_else(|| path.to_string_lossy().into_owned())
    }

    pub fn scan_library_folder(&self, dir: &Path, depth: u32) -> Result<ScanResult> {
        let env = (self.env)(None);
        Ok(scan_folder(&self.catalog()?, dir, depth, &env, cfg!(windows))?)
    }

    // ---- Biblioteca ----

    pub fn games(&self) -> Result<Vec<Game>> {
        Ok(self.lib().list()?)
    }

    pub fn add_game(&self, g: &NewGame) -> Result<i64> {
        Ok(self.lib().add_game(g, (self.now)() as i64)?)
    }

    pub fn update_game(&self, id: i64, g: &NewGame) -> Result<()> {
        Ok(self.lib().update_game(id, g)?)
    }

    /// Tira da biblioteca. Os saves, locais e na nuvem, continuam onde estão.
    pub fn remove_game(&self, id: i64) -> Result<()> {
        Ok(self.lib().remove_game(id)?)
    }

    // ---- Arte ----

    /// Capa do jogo no cache local. Baixa da Steam na primeira vez; offline ou sem id, devolve `None`
    /// e a interface mostra um espaço reservado.
    pub fn game_art(&self, id: i64) -> Result<Option<PathBuf>> {
        let game = self.lib().get(id)?;
        let cached = self.data_dir.join("art").join(format!("{id}.jpg"));
        if cached.is_file() {
            return Ok(Some(cached));
        }
        let (Some(cat_id), true) = (game.catalog_id, self.catalog_path().exists()) else { return Ok(None) };
        let Some(steam) = self.catalog()?.steam_id(cat_id)? else { return Ok(None) };
        Ok(download_art(&steam_header_url(steam), &cached))
    }

    // ---- Jogar ----

    fn roots(&self, game: &Game) -> Result<Vec<Root>> {
        if game.save_paths.is_empty() {
            return Err(Error::NoSavePath);
        }
        let env = (self.env)(game.install_dir.clone().map(PathBuf::from));
        game.save_paths
            .iter()
            .map(|logical| {
                // Pasta escolhida à mão e sem marcador conhecido: o caminho absoluto vale como está.
                if !logical.starts_with('<') {
                    return Ok((logical.clone(), PathBuf::from(logical)));
                }
                resolve(logical, &env).map(|p| (logical.clone(), p)).ok_or_else(|| Error::UnresolvedPath(logical.clone()))
            })
            .collect()
    }

    fn ctx(&self, game: &Game) -> Result<Ctx> {
        let sync = self.sync_dir()?.ok_or(Error::NoSyncDir)?;
        if !sync.is_dir() {
            return Err(Error::SyncDirUnavailable(sync.display().to_string()));
        }
        Ok(Ctx {
            game_dir: sync.join("SyncGameSave").join("games").join(slug(&game.name)),
            roots: self.roots(game)?,
            state_dir: self.data_dir.join("state").join(game.id.to_string()),
            device: self.device_name()?,
            now: (self.now)(),
        })
    }

    /// Abre o jogo com sync. `on_state` recebe o andamento ("sincronizando", "jogando", "salvando").
    /// Bloqueia até o jogo fechar; chame de uma thread, não da interface.
    pub fn play(&self, id: i64, opts: &PlayOptions, on_state: &dyn Fn(&str)) -> Result<Play> {
        let game = self.lib().get(id)?;
        let launch = |args: &[String], beat: &mut dyn FnMut()| -> Result<i32> {
            on_state("jogando");
            let started = std::time::Instant::now();
            let r = sgs_core::process::run_with_beat(&game.exe, args, None, Duration::from_secs(opts.quiet_secs), opts.heartbeat_every(), beat).map_err(|m| {
                Error::Other(format!("Não consegui abrir o jogo: {m}. Confira o caminho do executável."))
            });
            // O tempo conta mesmo se o jogo caiu; a espera final por processos restantes não entra.
            let secs = started.elapsed().as_secs().saturating_sub(opts.quiet_secs) as i64;
            let _ = self.lib().add_playtime(id, secs, (self.now)() as i64);
            r
        };
        if opts.skip_sync {
            let code = launch(&opts.args, &mut || {})?;
            return Ok(Play::Finished { exit_code: code, synced: false, notice: "Jogo fechado. O save não foi sincronizado.".into() });
        }
        let ctx = self.ctx(&game)?;
        let jdir = ctx.state_dir.join("journal");
        on_state("sincronizando");
        journal::begin(&jdir, &game.name, "abrir", ctx.now)?;
        let mut before = ctx.before_launch(opts.force);
        if matches!(before, Ok(Before::Conflict)) {
            if let Some(choice) = opts.conflict {
                let c = if choice == ChoiceDto::Local { Choice::UseLocal } else { Choice::UseCloud };
                before = ctx.resolve(c).map(|_| Before::Ready);
            }
        }
        journal::end(&jdir, &game.name)?;
        // O que aconteceu ao abrir também entra no aviso final.
        let ao_abrir = match before? {
            Before::Conflict => return Ok(Play::Conflict),
            Before::LockedByOther(pc) => return Ok(Play::Locked { pc }),
            Before::RestoredFromCloud => "O save da pasta de sincronização foi restaurado antes de abrir (o anterior ficou em backup). ",
            Before::UploadedLocal => "O save deste PC foi enviado antes de abrir. ",
            Before::Ready => "",
        };
        let game_dir = ctx.game_dir.clone();
        let device = ctx.device.clone();
        let result = launch(&opts.args, &mut || {
            let _ = lock::heartbeat(&game_dir, &device, (self.now)());
        });
        // Mesmo se o jogo nem abriu ou caiu, o save é conferido e a trava solta.
        on_state("salvando");
        journal::begin(&jdir, &game.name, "fechar", ctx.now)?;
        let created = ctx.after_exit();
        journal::end(&jdir, &game.name)?;
        let created = created?;
        let code = result?;
        let digest = sgs_core::index::scan_roots(&ctx.roots).map(|i| i.digest()).unwrap_or_default();
        self.lib().record_sync(id, &digest, (self.now)() as i64)?;
        let ao_fechar = if created { "Save enviado para a pasta de sincronização." } else { "Save sem mudanças depois de jogar." };
        Ok(Play::Finished { exit_code: code, synced: true, notice: format!("{ao_abrir}{ao_fechar}") })
    }

    // ---- Versões ----

    pub fn snapshots(&self, id: i64) -> Result<Vec<String>> {
        let game = self.lib().get(id)?;
        let ctx = self.ctx(&game)?;
        Ok(snapshot::list(&ctx.game_dir)?)
    }

    /// Restaura uma versão. O save atual vai antes para a pasta de backups. Devolve essa pasta.
    pub fn restore_snapshot(&self, id: i64, snapshot_id: &str) -> Result<PathBuf> {
        let game = self.lib().get(id)?;
        let ctx = self.ctx(&game)?;
        if let Ok(lock::LockState::HeldByOther(l)) = lock::state(&ctx.game_dir, &ctx.device, ctx.now) {
            return Err(Error::Other(format!("O jogo parece estar aberto em {}. Feche-o antes de restaurar.", l.device)));
        }
        let snap = ctx.game_dir.join("snapshots").join(snapshot_id);
        let backup = ctx.state_dir.join("backups").join(format!("{}_restaurar", ctx.now));
        let jdir = ctx.state_dir.join("journal");
        journal::begin(&jdir, &game.name, "restaurar", ctx.now)?;
        let r = snapshot::restore_roots(&snap, &ctx.roots, &backup);
        journal::end(&jdir, &game.name)?;
        r?;
        Ok(backup)
    }

    /// Operações que começaram e não terminaram (queda de energia, crash).
    pub fn interrupted_operations(&self) -> Vec<String> {
        let dir = self.data_dir.join("state");
        let mut out = Vec::new();
        if let Ok(rd) = std::fs::read_dir(dir) {
            for e in rd.flatten() {
                if let Ok(p) = journal::pending(&e.path().join("journal")) {
                    out.extend(p.into_iter().map(|x| format!("{} ({})", x.game, x.op)));
                }
            }
        }
        out
    }
}

fn unix_now() -> u64 {
    std::time::SystemTime::now().duration_since(std::time::UNIX_EPOCH).map(|d| d.as_secs()).unwrap_or(0)
}


pub fn steam_header_url(steam_id: u64) -> String {
    format!("https://cdn.cloudflare.steamstatic.com/steam/apps/{steam_id}/header.jpg")
}

/// Baixa uma imagem para o cache. Só guarda se for mesmo uma imagem (JPEG/PNG) e de tamanho razoável.
fn download_art(url: &str, dest: &Path) -> Option<PathBuf> {
    let mut resp = ureq::get(url).config().http_status_as_error(false).timeout_global(Some(Duration::from_secs(10))).build().call().ok()?;
    if resp.status().as_u16() != 200 {
        return None;
    }
    let body = resp.body_mut().with_config().limit(5 * 1024 * 1024).read_to_vec().ok()?;
    let is_image = body.starts_with(&[0xFF, 0xD8, 0xFF]) || body.starts_with(&[0x89, b'P', b'N', b'G']);
    if !is_image {
        return None;
    }
    std::fs::create_dir_all(dest.parent()?).ok()?;
    let tmp = dest.with_extension("tmp");
    std::fs::write(&tmp, body).ok()?;
    std::fs::rename(&tmp, dest).ok()?;
    Some(dest.to_path_buf())
}
