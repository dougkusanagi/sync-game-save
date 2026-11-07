//! Diário de operações perigosas. Grava a intenção antes e apaga depois;
//! se o app cair no meio, o arquivo sobra e o próximo início avisa.

use crate::{IoCtx, Result};
use serde::{Deserialize, Serialize};
use std::path::{Path, PathBuf};

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct Entry {
    pub op: String,
    pub game: String,
    pub started: u64,
}

fn file(dir: &Path, game: &str) -> PathBuf {
    dir.join(format!("{game}.json"))
}

pub fn begin(dir: &Path, game: &str, op: &str, now: u64) -> Result<()> {
    std::fs::create_dir_all(dir).ctx(dir)?;
    let e = Entry { op: op.into(), game: game.into(), started: now };
    let p = file(dir, game);
    std::fs::write(&p, serde_json::to_vec_pretty(&e)?).ctx(&p)
}

pub fn end(dir: &Path, game: &str) -> Result<()> {
    let p = file(dir, game);
    if p.exists() {
        std::fs::remove_file(&p).ctx(&p)?;
    }
    Ok(())
}

/// Operações que começaram e não terminaram.
pub fn pending(dir: &Path) -> Result<Vec<Entry>> {
    let mut out = Vec::new();
    if dir.is_dir() {
        for item in std::fs::read_dir(dir).ctx(dir)? {
            let p = item.ctx(dir)?.path();
            if p.extension().is_some_and(|e| e == "json") {
                out.push(serde_json::from_slice(&std::fs::read(&p).ctx(&p)?)?);
            }
        }
    }
    Ok(out)
}
