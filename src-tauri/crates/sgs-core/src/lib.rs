//! Núcleo do Sync Game Save: índice de arquivos, snapshots imutáveis,
//! comparação em três vias e restauração com backup.

pub mod discover;
pub mod engine;
pub mod env;
pub mod journal;
pub mod index;
pub mod lock;
pub mod paths;
pub mod process;
pub mod snapshot;
pub mod sync;

use thiserror::Error;

#[derive(Debug, Error)]
pub enum Error {
    #[error("erro de arquivo em {path}: {source}")]
    Io {
        path: std::path::PathBuf,
        source: std::io::Error,
    },
    #[error("dados inválidos: {0}")]
    Json(#[from] serde_json::Error),
    #[error("o arquivo {0} não confere com o hash do índice")]
    HashMismatch(String),
    #[error("os arquivos de save mudaram durante a sincronização; feche o jogo e tente novamente")]
    SaveChanged,
}

pub type Result<T> = std::result::Result<T, Error>;

pub(crate) trait IoCtx<T> {
    fn ctx(self, path: &std::path::Path) -> Result<T>;
}

impl<T> IoCtx<T> for std::io::Result<T> {
    fn ctx(self, path: &std::path::Path) -> Result<T> {
        self.map_err(|source| Error::Io {
            path: path.to_path_buf(),
            source,
        })
    }
}

/// Tentativas e espera para arquivos presos pelo jogo ou pelo antivírus.
const RETRIES: u32 = 8;
const RETRY_WAIT: std::time::Duration = std::time::Duration::from_millis(250);

fn is_busy(e: &std::io::Error) -> bool {
    // 32 e 33: violação de compartilhamento e de trava no Windows.
    e.kind() == std::io::ErrorKind::PermissionDenied || matches!(e.raw_os_error(), Some(32) | Some(33))
}

/// Repete a operação enquanto o arquivo estiver ocupado. Outros erros saem na hora.
pub(crate) fn retry<T>(mut op: impl FnMut() -> std::io::Result<T>) -> std::io::Result<T> {
    let mut left = RETRIES;
    loop {
        match op() {
            Err(e) if left > 0 && is_busy(&e) => {
                left -= 1;
                std::thread::sleep(RETRY_WAIT);
            }
            r => return r,
        }
    }
}
