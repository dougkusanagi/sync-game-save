use crate::{IoCtx, Result};
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};
use std::collections::BTreeMap;
use std::path::Path;

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct Entry {
    pub size: u64,
    pub sha256: String,
}

/// Caminho lógico (com `/`) → arquivo. Ordenado, para o hash do índice ser estável.
#[derive(Debug, Clone, Default, PartialEq, Eq, Serialize, Deserialize)]
pub struct Index {
    pub files: BTreeMap<String, Entry>,
}

impl Index {
    /// Hash do conteúdo do índice, usado para comparar estados.
    pub fn digest(&self) -> String {
        let mut h = Sha256::new();
        for (p, e) in &self.files {
            h.update(format!("{p}\0{}\0{}\n", e.size, e.sha256));
        }
        hex(&h.finalize())
    }
}

pub fn hex(bytes: &[u8]) -> String {
    bytes.iter().map(|b| format!("{b:02x}")).collect()
}

pub fn hash_file(path: &Path) -> Result<Entry> {
    let data = std::fs::read(path).ctx(path)?;
    Ok(Entry {
        size: data.len() as u64,
        sha256: hex(&Sha256::digest(&data)),
    })
}

/// Indexa todos os arquivos de `root`, recursivamente. Pasta ausente = índice vazio.
pub fn scan(root: &Path) -> Result<Index> {
    let mut index = Index::default();
    if root.is_dir() {
        walk(root, root, &mut index)?;
    }
    Ok(index)
}

fn walk(root: &Path, dir: &Path, index: &mut Index) -> Result<()> {
    for item in std::fs::read_dir(dir).ctx(dir)? {
        let path = item.ctx(dir)?.path();
        if path.is_dir() {
            walk(root, &path, index)?;
        } else {
            let rel = path
                .strip_prefix(root)
                .expect("o caminho está dentro da raiz")
                .components()
                .map(|c| c.as_os_str().to_string_lossy().into_owned())
                .collect::<Vec<_>>()
                .join("/");
            index.files.insert(rel, hash_file(&path)?);
        }
    }
    Ok(())
}
