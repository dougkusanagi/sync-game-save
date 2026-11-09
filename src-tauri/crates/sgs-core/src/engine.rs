//! Fluxo de abrir/fechar um jogo: junta trava, comparação em três vias,
//! snapshots e backups. Não abre processos; quem chama cuida disso.

use crate::index::{scan_roots, Root};
use crate::lock::{self, LockState};
use crate::snapshot::{self, Head};
use crate::sync::{decide, Action};
use crate::{IoCtx, Result};
use serde::{Deserialize, Serialize};
use std::path::PathBuf;

pub struct Ctx {
    /// Pasta do jogo na nuvem (`head.json`, `lock.json`, `snapshots/`).
    pub game_dir: PathBuf,
    /// Pastas de save neste PC (rótulo lógico + caminho). Rótulo vazio = pasta única.
    pub roots: Vec<Root>,
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
    /// Atalho para um jogo com uma só pasta de save.
    pub fn single(game_dir: PathBuf, save_dir: PathBuf, state_dir: PathBuf, device: &str, now: u64) -> Self {
        Ctx { game_dir, roots: vec![(String::new(), save_dir)], state_dir, device: device.into(), now }
    }

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
        let base = format!("{}_{}", self.now, self.device);
        let mut id = base.clone();
        let mut n = 1;
        while self.game_dir.join("snapshots").join(&id).exists() {
            id = format!("{base}~sgs-{n}");
            n += 1;
        }
        id
    }

    fn backup_dir(&self, reason: &str) -> PathBuf {
        self.state_dir.join("backups").join(format!("{}_{reason}", self.now))
    }

    fn upload(&self) -> Result<Head> {
        let head = snapshot::create_roots(&self.game_dir, &self.roots, &self.snapshot_id(), &self.device)?;
        self.set_base(&head.index_digest)?;
        Ok(head)
    }

    fn restore_cloud(&self, head: &Head, reason: &str) -> Result<()> {
        let snap = self.game_dir.join("snapshots").join(&head.snapshot);
        snapshot::restore_roots(&snap, &self.roots, &self.backup_dir(reason))?;
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
        let local = scan_roots(&self.roots)?;
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
        self.after_exit_with_beat(&mut || {})
    }

    /// Aguarda os arquivos estabilizarem e mantém a trava ativa durante a espera.
    pub fn after_exit_with_beat(&self, beat: &mut dyn FnMut()) -> Result<bool> {
        for _ in 0..3 {
            std::thread::sleep(std::time::Duration::from_secs(1));
            beat();
        }
        let mut local = scan_roots(&self.roots)?;
        let mut estavel = false;
        for _ in 0..60 {
            std::thread::sleep(std::time::Duration::from_secs(1));
            beat();
            let proximo = scan_roots(&self.roots)?;
            if local.digest() == proximo.digest() {
                local = proximo;
                estavel = true;
                break;
            }
            local = proximo;
        }
        if !estavel {
            return Err(crate::Error::Io {
                path: self.roots.first().map(|(_, p)| p.clone()).unwrap_or_default(),
                source: std::io::Error::new(std::io::ErrorKind::TimedOut, "os arquivos de save continuam mudando; feche o jogo e tente sincronizar novamente"),
            });
        }
        let changed = self.base()?.as_deref() != Some(local.digest().as_str());
        let created = if changed && !local.files.is_empty() {
            let head = snapshot::create_roots_checked(&self.game_dir, &self.roots, &self.snapshot_id(), &self.device, &local)?;
            self.set_base(&head.index_digest)?;
            true
        } else {
            false
        };
        lock::release(&self.game_dir, &self.device)?;
        Ok(created)
    }
}
