use crate::{IoCtx, Result};
use serde::{Deserialize, Serialize};
use std::path::Path;

/// Trava com heartbeat mais novo que isto (em segundos) conta como "jogando agora".
pub const FRESH_SECS: u64 = 10 * 60;

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct Lock {
    pub device: String,
    pub started: u64,
    pub heartbeat: u64,
}

#[derive(Debug, PartialEq, Eq)]
pub enum LockState {
    Free,
    /// Trava deste mesmo PC (ex.: o app caiu): pode seguir.
    Own,
    /// Outro PC jogando agora: avisar e pedir confirmação.
    HeldByOther(Lock),
    /// Trava antiga de outro PC: tratada como abandonada.
    Stale(Lock),
}

fn path(game_dir: &Path) -> std::path::PathBuf {
    game_dir.join("lock.json")
}

pub fn read(game_dir: &Path) -> Result<Option<Lock>> {
    let p = path(game_dir);
    if !p.exists() {
        return Ok(None);
    }
    Ok(Some(serde_json::from_slice(&std::fs::read(&p).ctx(&p)?)?))
}

pub fn state(game_dir: &Path, device: &str, now: u64) -> Result<LockState> {
    Ok(match read(game_dir)? {
        None => LockState::Free,
        Some(l) if l.device == device => LockState::Own,
        Some(l) if now.saturating_sub(l.heartbeat) < FRESH_SECS => LockState::HeldByOther(l),
        Some(l) => LockState::Stale(l),
    })
}

pub fn acquire(game_dir: &Path, device: &str, now: u64) -> Result<()> {
    std::fs::create_dir_all(game_dir).ctx(game_dir)?;
    write(game_dir, &Lock { device: device.into(), started: now, heartbeat: now })
}

/// Renova o heartbeat sem mudar o início. Sem trava própria, não faz nada.
pub fn heartbeat(game_dir: &Path, device: &str, now: u64) -> Result<()> {
    match read(game_dir)? {
        Some(l) if l.device == device => write(game_dir, &Lock { heartbeat: now, ..l }),
        _ => Ok(()),
    }
}

/// Libera só a trava deste PC; a de outro PC nunca é apagada.
pub fn release(game_dir: &Path, device: &str) -> Result<()> {
    if matches!(read(game_dir)?, Some(l) if l.device == device) {
        let p = path(game_dir);
        std::fs::remove_file(&p).ctx(&p)?;
    }
    Ok(())
}

fn write(game_dir: &Path, l: &Lock) -> Result<()> {
    let tmp = game_dir.join("lock.json.tmp");
    std::fs::write(&tmp, serde_json::to_vec_pretty(l)?).ctx(&tmp)?;
    std::fs::rename(&tmp, path(game_dir)).ctx(&tmp)
}
