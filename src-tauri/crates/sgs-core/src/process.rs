//! Abre um programa e espera ele e tudo que ele abriu terminarem.

use std::path::Path;
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::{Arc, Mutex};
use std::time::{Duration, Instant};

#[cfg(windows)]
mod job {
    use std::os::windows::io::AsRawHandle;
    use windows_sys::Win32::Foundation::{CloseHandle, HANDLE, INVALID_HANDLE_VALUE};
    use windows_sys::Win32::System::JobObjects::*;
    use windows_sys::Win32::System::Diagnostics::ToolHelp::*;
    use windows_sys::Win32::System::Threading::{OpenThread, ResumeThread, THREAD_SUSPEND_RESUME};

    pub struct Job(HANDLE);
    impl Drop for Job {
        fn drop(&mut self) { unsafe { CloseHandle(self.0); } }
    }
    impl Job {
        pub fn contains(&self, pid: u32) -> bool {
            use windows_sys::Win32::System::Threading::{OpenProcess, PROCESS_QUERY_LIMITED_INFORMATION};
            let process = unsafe { OpenProcess(PROCESS_QUERY_LIMITED_INFORMATION, 0, pid) };
            if process.is_null() { return false; }
            let mut member = 0;
            let ok = unsafe { IsProcessInJob(process, self.0, &mut member) };
            unsafe { CloseHandle(process); }
            ok != 0 && member != 0
        }
        pub fn create() -> Result<Self, String> {
            let h = unsafe { CreateJobObjectW(std::ptr::null(), std::ptr::null()) };
            if h.is_null() { Err(format!("não foi possível criar o grupo do jogo: {}", std::io::Error::last_os_error())) } else { Ok(Self(h)) }
        }
        pub fn attach_and_resume(&self, child: &mut std::process::Child) -> Result<(), String> {
            let process = child.as_raw_handle() as HANDLE;
            if unsafe { AssignProcessToJobObject(self.0, process) } == 0 {
                let err = std::io::Error::last_os_error();
                let _ = child.kill();
                let _ = child.wait();
                return Err(format!("não foi possível acompanhar o jogo: {err}"));
            }
            if let Err(err) = resume_primary(child.id()) {
                let _ = child.kill();
                let _ = child.wait();
                return Err(format!("não foi possível iniciar o jogo: {err}"));
            }
            Ok(())
        }
        pub fn active(&self) -> Result<bool, String> {
            let mut info: JOBOBJECT_BASIC_ACCOUNTING_INFORMATION = unsafe { std::mem::zeroed() };
            let ok = unsafe { QueryInformationJobObject(self.0, JobObjectBasicAccountingInformation, &mut info as *mut _ as *mut _, std::mem::size_of_val(&info) as u32, std::ptr::null_mut()) };
            if ok == 0 { Err(format!("não foi possível verificar o jogo: {}", std::io::Error::last_os_error())) } else { Ok(info.ActiveProcesses > 0) }
        }
        pub fn terminate(&self) -> Result<(), String> {
            if unsafe { TerminateJobObject(self.0, 1) } == 0 {
                Err(format!("não foi possível parar o jogo: {}", std::io::Error::last_os_error()))
            } else { Ok(()) }
        }
    }

    fn resume_primary(pid: u32) -> Result<(), String> {
        unsafe {
            let snapshot = CreateToolhelp32Snapshot(TH32CS_SNAPTHREAD, 0);
            if snapshot == INVALID_HANDLE_VALUE { return Err(std::io::Error::last_os_error().to_string()); }
            let mut entry: THREADENTRY32 = std::mem::zeroed();
            entry.dwSize = std::mem::size_of::<THREADENTRY32>() as u32;
            let mut found = Thread32First(snapshot, &mut entry) != 0;
            while found && entry.th32OwnerProcessID != pid {
                found = Thread32Next(snapshot, &mut entry) != 0;
            }
            CloseHandle(snapshot);
            if !found { return Err("thread inicial não encontrada".into()); }
            let thread = OpenThread(THREAD_SUSPEND_RESUME, 0, entry.th32ThreadID);
            if thread.is_null() { return Err(std::io::Error::last_os_error().to_string()); }
            let result = ResumeThread(thread);
            let error = if result == u32::MAX { Some(std::io::Error::last_os_error()) } else { None };
            CloseHandle(thread);
            match error { Some(err) => Err(err.to_string()), None => Ok(()) }
        }
    }

    #[cfg(test)]
    mod tests {
        use super::*;
        use std::os::windows::process::CommandExt;

        #[test]
        fn falha_de_associacao_encerra_processo_suspenso() {
            let mut child = std::process::Command::new(r"C:\Windows\System32\cmd.exe")
                .args(["/c", "exit", "0"])
                .creation_flags(0x00000004)
                .spawn().unwrap();
            let job = Job(std::ptr::null_mut());
            assert!(job.attach_and_resume(&mut child).is_err());
            assert!(child.try_wait().unwrap().is_some());
        }
    }
}

/// Acompanha um processo e todos os descendentes que ele abriu.
pub struct Tracker {
    sys: sysinfo::System,
    pids: std::collections::HashSet<u32>,
    inicios: std::collections::HashMap<u32, u64>,
    /// Pasta do jogo: processos que rodam dela também contam, mesmo sem serem filhos.
    /// Cobre jogos da Steam que fecham e pedem para a Steam abrir de novo.
    dir: Option<String>,
}

struct Controle {
    pedido: AtomicBool,
    enviado: Mutex<Option<Instant>>,
    forcar: AtomicBool,
    vivo: AtomicBool,
}

static ATIVOS: Mutex<Vec<(String, Arc<Controle>)>> = Mutex::new(Vec::new());

/// Inclui o intervalo de espera após o encerramento do grupo.
pub fn em_execucao(prog: &str) -> Result<bool, String> {
    Ok(ATIVOS.lock().map_err(|_| "não foi possível verificar se o jogo está aberto")?.iter().any(|(p, _)| p == prog))
}

/// Pede fechamento normal e aguarda até dez segundos pelo grupo.
/// O resultado indica se ainda há processos que podem precisar de força.
pub fn parar(prog: &str) -> Result<Option<bool>, String> {
    let controle = ATIVOS.lock().map_err(|_| "não foi possível verificar o jogo")?
        .iter().find(|(p, _)| p == prog).map(|(_, c)| c.clone());
    let Some(controle) = controle else { return Ok(None) };
    controle.pedido.store(true, Ordering::SeqCst);
    while controle.vivo.load(Ordering::SeqCst) {
        if controle.enviado.lock().map_err(|_| "não foi possível verificar o pedido")?
            .is_some_and(|t| t.elapsed() >= Duration::from_secs(10)) { break; }
        std::thread::sleep(Duration::from_millis(100));
    }
    Ok(Some(controle.vivo.load(Ordering::SeqCst)))
}

/// Só aceita a força depois de um pedido normal com espera de dez segundos.
pub fn forcar_parada(prog: &str) -> Result<bool, String> {
    let controle = ATIVOS.lock().map_err(|_| "não foi possível verificar o jogo")?
        .iter().find(|(p, _)| p == prog).map(|(_, c)| c.clone());
    let Some(controle) = controle else { return Ok(false) };
    let pedido = controle.enviado.lock().map_err(|_| "não foi possível verificar o pedido")?;
    if !pedido.is_some_and(|t| t.elapsed() >= Duration::from_secs(10)) {
        return Err("Aguarde o jogo responder ao pedido de fechamento.".into());
    }
    if !controle.vivo.load(Ordering::SeqCst) { return Ok(false); }
    controle.forcar.store(true, Ordering::SeqCst);
    Ok(true)
}

fn chave_de_caminho(p: &Path) -> String {
    let s = p.to_string_lossy().replace('\\', "/");
    let s = s.trim_end_matches('/');
    if cfg!(windows) { s.to_lowercase() } else { s.to_string() }
}

#[cfg(windows)]
fn pedir_fechamento(tracker: Option<&Tracker>, job: &job::Job) {
    use windows_sys::Win32::Foundation::{HWND, LPARAM};
    use windows_sys::Win32::UI::WindowsAndMessaging::{EnumWindows, GetWindowThreadProcessId, PostMessageW, WM_CLOSE};
    struct Contexto<'a> { tracker: Option<&'a Tracker>, job: &'a job::Job }
    unsafe extern "system" fn janela(hwnd: HWND, data: LPARAM) -> i32 {
        let contexto = unsafe { &*(data as *const Contexto<'_>) };
        let mut pid = 0;
        unsafe { GetWindowThreadProcessId(hwnd, &mut pid); }
        if contexto.tracker.is_some_and(|t| t.pids.contains(&pid)) || contexto.job.contains(pid) {
            unsafe { PostMessageW(hwnd, WM_CLOSE, 0, 0); }
        }
        1
    }
    let contexto = Contexto { tracker, job };
    unsafe { EnumWindows(Some(janela), &contexto as *const _ as LPARAM); }
}

impl Tracker {
    pub fn new(root: u32) -> Option<Self> {
        // Sem acesso à lista de processos, só se espera o programa que foi aberto.
        let mut tracker = Tracker { sys: sysinfo::System::new().ok()?, pids: [root].into(), inicios: Default::default(), dir: None };
        tracker.refresh();
        Some(tracker)
    }

    /// Como `new`, contando também os processos cujo executável fica dentro de `dir`.
    pub fn with_dir(root: u32, dir: Option<&Path>) -> Option<Self> {
        let mut t = Self::new(root)?;
        t.dir = dir.map(chave_de_caminho).filter(|d| d.matches('/').count() >= 2);
        Some(t)
    }

    /// Atualiza a lista; devolve se ainda há algum processo do grupo vivo.
    fn refresh(&mut self) -> bool {
        use sysinfo::{ProcessRefreshKind, ProcessesToUpdate, UpdateKind};
        let tipo = if self.dir.is_some() { ProcessRefreshKind::nothing().with_exe(UpdateKind::OnlyIfNotSet) } else { ProcessRefreshKind::nothing() };
        self.sys.refresh_processes_specifics(ProcessesToUpdate::All, true, tipo);
        // O sistema pode reutilizar o PID de um processo encerrado. Ele deixa
        // de pertencer ao jogo quando a data de início muda.
        self.pids.retain(|pid| {
            match (self.inicios.get(pid), self.sys.process(sysinfo::Pid::from_u32(*pid))) {
                (Some(inicio), Some(processo)) => *inicio == processo.start_time(),
                _ => true,
            }
        });
        if let Some(dir) = &self.dir {
            let eu = std::process::id();
            let prefixo = format!("{dir}/");
            let da_pasta: Vec<u32> = self
                .sys
                .processes()
                .iter()
                .filter(|(pid, p)| pid.as_u32() != eu && p.exe().is_some_and(|e| chave_de_caminho(e).starts_with(&prefixo)))
                .map(|(pid, _)| pid.as_u32())
                .collect();
            self.pids.extend(da_pasta);
        }
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
        for pid in &self.pids {
            if let Some(processo) = self.sys.process(sysinfo::Pid::from_u32(*pid)) {
                self.inicios.insert(*pid, processo.start_time());
            }
        }
        self.pids.iter().any(|p| self.sys.process(sysinfo::Pid::from_u32(*p)).is_some())
    }

    /// Encerra todos os processos do grupo.
    fn matar(&mut self) {
        self.refresh();
        for p in &self.pids {
            if let Some(proc) = self.sys.process(sysinfo::Pid::from_u32(*p)) {
                proc.kill();
            }
        }
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
    #[cfg(windows)]
    let job = {
        use std::os::windows::process::CommandExt;
        command.creation_flags(0x00000004); // CREATE_SUSPENDED
        job::Job::create()?
    };
    command.args(args);
    let pasta = Path::new(prog).parent().filter(|d| d.is_dir()).map(Path::to_path_buf);
    if let Some(dir) = cwd.map(Path::to_path_buf).or_else(|| pasta.clone()) {
        command.current_dir(dir);
    }
    let mut child = command.spawn().map_err(|x| format!("não consegui abrir {prog}: {x}"))?;
    #[cfg(windows)]
    job.attach_and_resume(&mut child)?;
    let mut tracker = Tracker::with_dir(child.id(), pasta.as_deref());
    let controle = Arc::new(Controle { pedido: AtomicBool::new(false), enviado: Mutex::new(None), forcar: AtomicBool::new(false), vivo: AtomicBool::new(true) });
    if let Ok(mut a) = ATIVOS.lock() {
        a.push((prog.to_string(), controle.clone()));
    }
    // Sai da lista ao terminar, de qualquer jeito.
    struct Sai(Arc<Controle>);
    impl Drop for Sai {
        fn drop(&mut self) {
            self.0.vivo.store(false, Ordering::SeqCst);
            if let Ok(mut a) = ATIVOS.lock() {
                a.retain(|(_, f)| !Arc::ptr_eq(f, &self.0));
            }
        }
    }
    let _sai = Sai(controle.clone());
    let mut beat = Beat::new(every, tick);
    let mut status = None;
    let mut gone_since: Option<Instant> = None;
    let mut pediu = false;
    loop {
        beat.maybe();
        if status.is_none() {
            status = child.try_wait().map_err(|x| x.to_string())?;
        }
        let rastreado = tracker.as_mut().is_some_and(|t| t.refresh());
        #[cfg(windows)]
        let no_job = job.active()?;
        #[cfg(not(windows))]
        let no_job = false;
        let vivo = status.is_none() || rastreado || no_job;
        controle.vivo.store(vivo, Ordering::SeqCst);
        if vivo && !pediu && controle.pedido.load(Ordering::SeqCst) {
            #[cfg(windows)]
            pedir_fechamento(tracker.as_ref(), &job);
            *controle.enviado.lock().map_err(|_| "não foi possível registrar o pedido")? = Some(Instant::now());
            pediu = true;
        }
        if vivo && controle.forcar.load(Ordering::SeqCst) {
            #[cfg(windows)]
            if no_job { job.terminate()?; }
            if let Some(t) = tracker.as_mut() { t.matar(); }
            if status.is_none() { let _ = child.kill(); }
        }
        if vivo {
            gone_since = None;
        } else if gone_since.get_or_insert_with(Instant::now).elapsed() >= quiet {
            break;
        }
        std::thread::sleep(Duration::from_millis(200));
    }
    Ok(status.expect("processo terminou").code().unwrap_or(1))
}
