//! Fluxo de abrir/fechar um jogo: junta trava, comparação em três vias,
//! snapshots e backups. Não abre processos; quem chama cuida disso.

use crate::index::scan;
use crate::lock::{self, LockState};
use crate::snapshot::{self, Head};
use crate::sync::{decide, Action};
use crate::{IoCtx, Result};
use serde::{Deserialize, Serialize};
use std::path::PathBuf;

pub struct Ctx {
    /// Pasta do jogo na nuvem (`head.json`, `lock.json`, `snapshots/`).
    pub game_dir: PathBuf,
    /// Pasta do save neste PC.
    pub save_dir: PathBuf,
    /// Estado local, fora da nuvem (`base.json`, `backups/`).
    pub state_dir: PathBuf,
    pub device: String,
    /// Segundos desde 1970; passado de fora para facilitar testes.
    pub now: u64,
}

#[derive(Debug, PartialEq, Eq)]
pub enum Before {
    /// Pode abrir o jogo.
    Ready,
    RestoredFromCloud,
    UploadedLocal,
    /// Os dois lados mudaram: o usuário escolhe com `resolve`.
    Conflict,
    /// Outro PC está jogando: pedir confirmação e chamar de novo com `force`.
    LockedByOther(String),
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Choice {
    UseLocal,
    UseCloud,
}

#[derive(Serialize, Deserialize)]
struct Base {
    digest: Option<String>,
}

impl Ctx {
    fn base_path(&self) -> PathBuf {
        self.state_dir.join("base.json")
    }

    fn base(&self) -> Result<Option<String>> {
        let p = self.base_path();
        if !p.exists() {
            return Ok(None);
        }
        let b: Base = serde_json::from_slice(&std::fs::read(&p).ctx(&p)?)?;
        Ok(b.digest)
    }

    fn set_base(&self, digest: &str) -> Result<()> {
        std::fs::create_dir_all(&self.state_dir).ctx(&self.state_dir)?;
        let p = self.base_path();
        let b = Base { digest: Some(digest.to_string()) };
        std::fs::write(&p, serde_json::to_vec_pretty(&b)?).ctx(&p)
    }

    fn snapshot_id(&self) -> String {
        format!("{}_{}", self.now, self.device)
    }

    fn backup_dir(&self, reason: &str) -> PathBuf {
        self.state_dir.join("backups").join(format!("{}_{reason}", self.now))
    }

    fn upload(&self) -> Result<Head> {
        let head = snapshot::create(&self.game_dir, &self.save_dir, &self.snapshot_id(), &self.device)?;
        self.set_base(&head.index_digest)?;
        Ok(head)
    }

    fn restore_cloud(&self, head: &Head, reason: &str) -> Result<()> {
        let snap = self.game_dir.join("snapshots").join(&head.snapshot);
        snapshot::restore(&snap, &self.save_dir, &self.backup_dir(reason))?;
        self.set_base(&head.index_digest)
    }

    /// Passo antes de abrir o jogo.
    pub fn before_launch(&self, force: bool) -> Result<Before> {
        if let LockState::HeldByOther(l) = lock::state(&self.game_dir, &self.device, self.now)? {
            if !force {
                return Ok(Before::LockedByOther(l.device));
            }
        }
        let head = snapshot::read_head(&self.game_dir)?;
        let local = scan(&self.save_dir)?;
        let cloud = head.as_ref().map(|h| h.index_digest.as_str());
        let result = match decide(self.base()?.as_deref(), &local, cloud) {
            Action::Nothing => {
                // Primeiro contato com os dois lados iguais: guarda a base.
                if let Some(c) = cloud {
                    self.set_base(c)?;
                }
                Before::Ready
            }
            Action::RestoreFromCloud => {
                self.restore_cloud(head.as_ref().expect("nuvem mudou, logo há head"), "antes-de-restaurar")?;
                Before::RestoredFromCloud
            }
            Action::UploadLocal => {
                if local.files.is_empty() {
                    // Save local vazio e sem base: nada a enviar.
                    Before::Ready
                } else {
                    self.upload()?;
                    Before::UploadedLocal
                }
            }
            Action::Conflict => return Ok(Before::Conflict),
        };
        lock::acquire(&self.game_dir, &self.device, self.now)?;
        Ok(result)
    }

    /// Resolve um conflito. O lado descartado continua recuperável
    /// (snapshot na nuvem ou backup local).
    pub fn resolve(&self, choice: Choice) -> Result<()> {
        match choice {
            Choice::UseLocal => {
                self.upload()?;
            }
            Choice::UseCloud => {
                let head = snapshot::read_head(&self.game_dir)?.expect("conflito exige head");
                self.restore_cloud(&head, "conflito")?;
            }
        }
        lock::acquire(&self.game_dir, &self.device, self.now)
    }

    /// Passo depois de o jogo fechar (qualquer código de saída).
    /// Devolve `true` se criou um snapshot novo.
    pub fn after_exit(&self) -> Result<bool> {
        let local = scan(&self.save_dir)?;
        let changed = self.base()?.as_deref() != Some(local.digest().as_str());
        let created = if changed && !local.files.is_empty() {
            self.upload()?;
            true
        } else {
            false
        };
        lock::release(&self.game_dir, &self.device)?;
        Ok(created)
    }
}
