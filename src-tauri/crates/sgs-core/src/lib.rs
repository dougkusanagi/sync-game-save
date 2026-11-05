//! Núcleo do Sync Game Save: índice de arquivos, snapshots imutáveis,
//! comparação em três vias e restauração com backup.

pub mod index;
pub mod lock;
pub mod paths;
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
