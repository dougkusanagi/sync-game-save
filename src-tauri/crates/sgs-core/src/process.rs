//! Abre um programa e espera ele e tudo que ele abriu terminarem.

use std::path::Path;
use std::time::Duration;

/// Acompanha um processo e todos os descendentes que ele abriu.
pub struct Tracker {
    sys: sysinfo::System,
    pids: std::collections::HashSet<u32>,
}

impl Tracker {
    pub fn new(root: u32) -> Option<Self> {
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
    pub fn wait_until_gone(&mut self, quiet: std::time::Duration, beat: &mut Beat) {
        let mut gone_since: Option<std::time::Instant> = None;
        loop {
            beat.maybe();
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

/// Chama `tick` de tempos em tempos enquanto se espera (heartbeat da trava).
pub struct Beat<'a> {
    every: Duration,
    last: std::time::Instant,
    tick: &'a mut dyn FnMut(),
}

impl<'a> Beat<'a> {
    pub fn new(every: Duration, tick: &'a mut dyn FnMut()) -> Self {
        Beat { every, last: std::time::Instant::now(), tick }
    }

    pub fn maybe(&mut self) {
        if self.last.elapsed() >= self.every {
            self.last = std::time::Instant::now();
            (self.tick)();
        }
    }
}

/// Abre `prog`, espera ele terminar e depois espera os processos que ele abriu (launchers).
/// O jogo abre na pasta do executável, como a Steam faz (muitos jogos acham os arquivos por ali).
/// Devolve o código de saída do programa aberto. `quiet` é o tempo sem nenhum processo do
/// grupo para considerar o jogo fechado.
pub fn run(prog: &str, args: &[String], cwd: Option<&Path>, quiet: Duration) -> Result<i32, String> {
    run_with_beat(prog, args, cwd, quiet, Duration::from_secs(3600), &mut || {})
}

/// Como `run`, chamando `tick` a cada `every` enquanto o jogo roda.
pub fn run_with_beat(
    prog: &str,
    args: &[String],
    cwd: Option<&Path>,
    quiet: Duration,
    every: Duration,
    tick: &mut dyn FnMut(),
) -> Result<i32, String> {
    let mut command = std::process::Command::new(prog);
    command.args(args);
    if let Some(dir) = cwd.map(Path::to_path_buf).or_else(|| Path::new(prog).parent().filter(|d| d.is_dir()).map(Path::to_path_buf)) {
        command.current_dir(dir);
    }
    let mut child = command.spawn().map_err(|x| format!("não consegui abrir {prog}: {x}"))?;
    let mut tracker = Tracker::new(child.id());
    let mut beat = Beat::new(every, tick);
    let status = loop {
        beat.maybe();
        match child.try_wait().map_err(|x| x.to_string())? {
            Some(st) => break st,
            None => std::thread::sleep(Duration::from_millis(200)),
        }
    };
    if let Some(t) = tracker.as_mut() {
        t.wait_until_gone(quiet, &mut beat);
    }
    Ok(status.code().unwrap_or(1))
}
