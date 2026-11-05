//! CLI `sgs`: usa o `sgs-core` para abrir um jogo com sync seguro e para
//! fazer backup, restaurar e consultar o estado.

use sgs_core::engine::{Before, Choice, Ctx};
use sgs_core::{journal, lock, snapshot};
use std::io::{BufRead, Write};
use std::path::PathBuf;

const HELP: &str = "\
Uso: sgs <comando> --sync-dir <pasta> --game <nome> --save-dir <pasta do save ou <marcador>/caminho> [opções]

Comandos:
  run [--force] [--on-conflict local|cloud|abort] -- <programa> [args...]
        Sincroniza, abre o programa, espera ele fechar e envia o save.
  backup     Cria um snapshot se o save mudou.
  restore [--snapshot <id>]   Restaura o snapshot (padrão: o atual), com backup antes.
  status [--json]   Mostra o estado do sync.
  list [--json]     Lista os snapshots.

Opções:
  --state-dir <pasta>  Estado local (padrão: %LOCALAPPDATA%/SyncGameSave/<jogo>)
  --device <nome>      Nome deste PC (padrão: nome do computador)
  --cwd <pasta>        Pasta de trabalho do jogo (padrão: a pasta do executável)
  --quiet-secs <n>     Segundos sem nenhum processo do jogo para considerar fechado (padrão: 5)
";

struct Opts {
    cmd: String,
    sync_dir: PathBuf,
    game: String,
    save_dir: PathBuf,
    state_dir: Option<PathBuf>,
    device: Option<String>,
    snapshot: Option<String>,
    on_conflict: Option<String>,
    force: bool,
    json: bool,
    program: Vec<String>,
    now: Option<u64>,
    quiet_secs: u64,
    cwd: Option<PathBuf>,
}

fn parse(args: Vec<String>) -> Result<Opts, String> {
    let mut it = args.into_iter();
    let cmd = it.next().ok_or("falta o comando")?;
    let (mut sync_dir, mut game, mut save_dir) = (None, None, None);
    let mut o = Opts {
        cmd,
        sync_dir: PathBuf::new(),
        game: String::new(),
        save_dir: PathBuf::new(),
        state_dir: None,
        device: None,
        snapshot: None,
        on_conflict: None,
        force: false,
        json: false,
        program: vec![],
        now: None,
        quiet_secs: 5,
        cwd: None,
    };
    while let Some(a) = it.next() {
        let mut val = |name: &str| it.next().ok_or(format!("falta o valor de {name}"));
        match a.as_str() {
            "--sync-dir" => sync_dir = Some(PathBuf::from(val(&a)?)),
            "--game" => game = Some(val(&a)?),
            "--save-dir" => save_dir = Some(PathBuf::from(val(&a)?)),
            "--state-dir" => o.state_dir = Some(PathBuf::from(val(&a)?)),
            "--device" => o.device = Some(val(&a)?),
            "--snapshot" => o.snapshot = Some(val(&a)?),
            "--on-conflict" => o.on_conflict = Some(val(&a)?),
            "--now" => o.now = Some(val(&a)?.parse().map_err(|_| "--now precisa ser um número")?),
            "--quiet-secs" => o.quiet_secs = val(&a)?.parse().map_err(|_| "--quiet-secs precisa ser um número")?,
            "--cwd" => o.cwd = Some(PathBuf::from(val(&a)?)),
            "--force" => o.force = true,
            "--json" => o.json = true,
            "--" => {
                o.program = it.by_ref().collect();
            }
            other => return Err(format!("opção desconhecida: {other}")),
        }
    }
    o.sync_dir = sync_dir.ok_or("falta --sync-dir")?;
    o.game = game.ok_or("falta --game")?;
    o.save_dir = save_dir.ok_or("falta --save-dir")?;
    Ok(o)
}

/// Nome seguro para pasta: troca caracteres inválidos no Windows.
fn slug(name: &str) -> String {
    let s: String = name
        .chars()
        .map(|c| if "<>:\"/\\|?*".contains(c) || c.is_control() { '_' } else { c })
        .collect();
    s.trim().trim_end_matches('.').to_string()
}

fn device_name() -> String {
    std::env::var("COMPUTERNAME")
        .or_else(|_| std::env::var("HOSTNAME"))
        .unwrap_or_else(|_| "PC".into())
}

fn unix_now() -> u64 {
    std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .map(|d| d.as_secs())
        .unwrap_or(0)
}

/// Aceita caminho comum ou lógico (`<winDocuments>/SavedGames/Jogo`).
fn resolve_save_dir(p: &std::path::Path) -> Result<PathBuf, String> {
    let text = p.to_string_lossy();
    if !text.starts_with('<') {
        return Ok(p.to_path_buf());
    }
    sgs_core::paths::resolve(&text, &sgs_core::env::system_env(None))
        .ok_or_else(|| format!("não consegui resolver o caminho {text} neste PC"))
}

fn context(o: &Opts) -> Result<Ctx, String> {
    let game = slug(&o.game);
    if game.is_empty() {
        return Err("o nome do jogo ficou vazio".into());
    }
    let state_dir = match &o.state_dir {
        Some(d) => d.clone(),
        None => {
            let base = std::env::var_os("LOCALAPPDATA")
                .map(PathBuf::from)
                .or_else(|| std::env::var_os("HOME").map(|h| PathBuf::from(h).join(".local/share")))
                .ok_or("não achei a pasta de dados local; use --state-dir")?;
            base.join("SyncGameSave").join(&game)
        }
    };
    Ok(Ctx {
        game_dir: o.sync_dir.join("SyncGameSave").join("games").join(&game),
        roots: vec![(String::new(), resolve_save_dir(&o.save_dir)?)],
        state_dir,
        device: o.device.clone().unwrap_or_else(device_name),
        now: o.now.unwrap_or_else(unix_now),
    })
}

fn e<T: std::fmt::Display>(x: T) -> String {
    x.to_string()
}

/// Executa um comando. Devolve o código de saída do processo.
pub fn exec(args: Vec<String>, out: &mut impl Write, input: &mut impl BufRead) -> Result<i32, String> {
    if args.first().is_none_or(|a| a == "--help" || a == "-h" || a == "help") {
        write!(out, "{HELP}").map_err(e)?;
        return Ok(0);
    }
    let o = parse(args)?;
    let ctx = context(&o)?;
    let jdir = ctx.state_dir.join("journal");
    let pending = journal::pending(&jdir).map_err(e)?;
    for p in &pending {
        writeln!(out, "Aviso: a operação \"{}\" de {} não terminou na última vez. Confira o save.", p.op, p.game).map_err(e)?;
    }
    match o.cmd.as_str() {
        "status" => status(&ctx, &o, out),
        "list" => {
            let ids = snapshot::list(&ctx.game_dir).map_err(e)?;
            if o.json {
                writeln!(out, "{}", serde_json::json!(ids)).map_err(e)?;
            } else if ids.is_empty() {
                writeln!(out, "Nenhum snapshot ainda.").map_err(e)?;
            } else {
                for id in ids {
                    writeln!(out, "{id}").map_err(e)?;
                }
            }
            Ok(0)
        }
        "backup" => {
            journal::begin(&jdir, &o.game, "backup", ctx.now).map_err(e)?;
            let created = ctx.after_exit().map_err(e)?;
            journal::end(&jdir, &o.game).map_err(e)?;
            writeln!(out, "{}", if created { "Snapshot criado." } else { "Nada mudou; nenhum snapshot novo." }).map_err(e)?;
            Ok(0)
        }
        "restore" => {
            let head = snapshot::read_head(&ctx.game_dir).map_err(e)?;
            let id = match (&o.snapshot, &head) {
                (Some(id), _) => id.clone(),
                (None, Some(h)) => h.snapshot.clone(),
                (None, None) => return Err("não há snapshot para restaurar".into()),
            };
            let snap = ctx.game_dir.join("snapshots").join(&id);
            let backup = ctx.state_dir.join("backups").join(format!("{}_restaurar", ctx.now));
            journal::begin(&jdir, &o.game, "restore", ctx.now).map_err(e)?;
            snapshot::restore_roots(&snap, &ctx.roots, &backup).map_err(e)?;
            journal::end(&jdir, &o.game).map_err(e)?;
            writeln!(out, "Snapshot {id} restaurado. O save anterior está em {}.", backup.display()).map_err(e)?;
            Ok(0)
        }
        "run" => run(&ctx, &o, &jdir, out, input),
        other => Err(format!("comando desconhecido: {other}. Use `sgs help`.")),
    }
}

fn status(ctx: &Ctx, o: &Opts, out: &mut impl Write) -> Result<i32, String> {
    let head = snapshot::read_head(&ctx.game_dir).map_err(e)?;
    let lk = lock::state(&ctx.game_dir, &ctx.device, ctx.now).map_err(e)?;
    let local = sgs_core::index::scan_roots(&ctx.roots).map_err(e)?;
    let lock_txt = match &lk {
        lock::LockState::Free => "livre".to_string(),
        lock::LockState::Own => "deste PC".to_string(),
        lock::LockState::HeldByOther(l) => format!("em uso em {}", l.device),
        lock::LockState::Stale(l) => format!("antiga de {} (abandonada)", l.device),
    };
    if o.json {
        let v = serde_json::json!({
            "head": head.as_ref().map(|h| &h.snapshot),
            "lock": lock_txt,
            "localFiles": local.files.len(),
            "localDigest": local.digest(),
            "cloudDigest": head.as_ref().map(|h| &h.index_digest),
        });
        writeln!(out, "{v}").map_err(e)?;
    } else {
        writeln!(out, "Snapshot atual: {}", head.as_ref().map_or("nenhum", |h| h.snapshot.as_str())).map_err(e)?;
        writeln!(out, "Trava: {lock_txt}").map_err(e)?;
        writeln!(out, "Arquivos de save locais: {}", local.files.len()).map_err(e)?;
    }
    Ok(0)
}

fn run(ctx: &Ctx, o: &Opts, jdir: &std::path::Path, out: &mut impl Write, input: &mut impl BufRead) -> Result<i32, String> {
    let (prog, rest) = o.program.split_first().ok_or("falta o programa depois de `--`")?;
    if !o.sync_dir.is_dir() {
        return Err(format!(
            "a pasta de sync {} não está disponível. Confira se a nuvem está montada, ou abra o jogo sem o sgs.",
            o.sync_dir.display()
        ));
    }
    journal::begin(jdir, &o.game, "abrir", ctx.now).map_err(e)?;
    let mut before = ctx.before_launch(o.force).map_err(e)?;
    if before == Before::Conflict {
        let choice = match o.on_conflict.as_deref() {
            Some("local") => Some(Choice::UseLocal),
            Some("cloud") => Some(Choice::UseCloud),
            Some("abort") => None,
            _ => {
                writeln!(out, "Conflito: o save deste PC e o da nuvem mudaram. Usar o [l]ocal, o da [n]uvem ou [c]ancelar?").map_err(e)?;
                let mut line = String::new();
                input.read_line(&mut line).map_err(e)?;
                match line.trim() {
                    "l" => Some(Choice::UseLocal),
                    "n" => Some(Choice::UseCloud),
                    _ => None,
                }
            }
        };
        match choice {
            Some(c) => {
                ctx.resolve(c).map_err(e)?;
                before = Before::Ready;
            }
            None => {
                journal::end(jdir, &o.game).map_err(e)?;
                writeln!(out, "Cancelado. Nada foi alterado.").map_err(e)?;
                return Ok(2);
            }
        }
    }
    if let Before::LockedByOther(pc) = &before {
        journal::end(jdir, &o.game).map_err(e)?;
        writeln!(out, "O jogo parece aberto em {pc}. Use --force para abrir mesmo assim.").map_err(e)?;
        return Ok(3);
    }
    journal::end(jdir, &o.game).map_err(e)?;
    match before {
        Before::RestoredFromCloud => writeln!(out, "Save da nuvem restaurado. Se havia um save neste PC, uma cópia ficou na pasta de backups."),
        Before::UploadedLocal => writeln!(out, "Save deste PC enviado para a nuvem."),
        _ => writeln!(out, "Save em dia."),
    }
    .map_err(e)?;

    let mut command = std::process::Command::new(prog);
    command.args(rest);
    // Como a Steam: abre o jogo na pasta do executável (muitos jogos procuram "Content" ali).
    if let Some(dir) = o.cwd.clone().or_else(|| std::path::Path::new(prog).parent().filter(|d| d.is_dir()).map(|d| d.to_path_buf())) {
        command.current_dir(dir);
    }
    let status = match command.spawn() {
        Ok(mut child) => {
            let pid = child.id();
            // Segue o programa e os processos que ele abriu, mesmo se ele fechar logo (launcher).
            let mut tracker = Tracker::new(pid);
            let st = child.wait();
            if let Some(t) = tracker.as_mut() {
                t.wait_until_gone(std::time::Duration::from_secs(o.quiet_secs));
            }
            st
        }
        Err(x) => Err(x),
    };
    // Mesmo se o programa nem abriu ou saiu com erro, o save é conferido e a trava solta.
    journal::begin(jdir, &o.game, "fechar", ctx.now).map_err(e)?;
    let created = ctx.after_exit().map_err(e)?;
    journal::end(jdir, &o.game).map_err(e)?;
    writeln!(out, "{}", if created { "Save enviado para a nuvem." } else { "Save sem mudanças." }).map_err(e)?;
    let status = status.map_err(|x| format!("não consegui abrir {prog}: {x}"))?;
    Ok(status.code().unwrap_or(1))
}

/// Acompanha um processo e todos os descendentes que ele abriu.
struct Tracker {
    sys: sysinfo::System,
    pids: std::collections::HashSet<u32>,
}

impl Tracker {
    fn new(root: u32) -> Option<Self> {
        // Sem acesso à lista de processos, só se espera o programa que foi aberto.
        Some(Tracker { sys: sysinfo::System::new().ok()?, pids: [root].into() })
    }

    /// Atualiza a lista; devolve se ainda há algum processo do grupo vivo.
    fn refresh(&mut self) -> bool {
        use sysinfo::{ProcessRefreshKind, ProcessesToUpdate};
        self.sys.refresh_processes_specifics(ProcessesToUpdate::All, true, ProcessRefreshKind::nothing());
        // Um filho novo entra no grupo se o pai (mesmo já encerrado) está no grupo.
        loop {
            let new: Vec<u32> = self
                .sys
                .processes()
                .iter()
                .filter(|(pid, p)| !self.pids.contains(&pid.as_u32()) && p.parent().is_some_and(|pp| self.pids.contains(&pp.as_u32())))
                .map(|(pid, _)| pid.as_u32())
                .collect();
            if new.is_empty() {
                break;
            }
            self.pids.extend(new);
        }
        self.pids.iter().any(|p| self.sys.process(sysinfo::Pid::from_u32(*p)).is_some())
    }

    /// Espera até nenhum processo do grupo existir por `quiet` seguido.
    fn wait_until_gone(&mut self, quiet: std::time::Duration) {
        let mut gone_since: Option<std::time::Instant> = None;
        loop {
            if self.refresh() {
                gone_since = None;
            } else {
                let t = *gone_since.get_or_insert_with(std::time::Instant::now);
                if t.elapsed() >= quiet {
                    return;
                }
            }
            std::thread::sleep(std::time::Duration::from_millis(200));
        }
    }
}
