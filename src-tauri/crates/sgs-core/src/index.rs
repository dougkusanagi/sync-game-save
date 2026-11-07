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
    let data = crate::retry(|| std::fs::read(path)).ctx(path)?;
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

/// Uma pasta de save: o rótulo é o caminho lógico (`<winDocuments>/SavedGames/Jogo`) e vira
/// o prefixo dos arquivos no snapshot, o que o torna portátil entre PCs.
/// Rótulo vazio = pasta única, sem prefixo.
pub type Root = (String, std::path::PathBuf);

/// Prefixo seguro para nome de pasta (o Windows não aceita `<`, `>`, `:` etc.).
pub fn prefix(label: &str) -> String {
    label.chars().filter(|c| !"<>:\"|?*\\".contains(*c)).collect::<String>().trim_matches('/').to_string()
}

/// Indexa várias pastas de save num só índice, com o prefixo de cada uma nas chaves.
pub fn scan_roots(roots: &[Root]) -> Result<Index> {
    let mut all = Index::default();
    for (label, path) in roots {
        let pre = prefix(label);
        for (rel, entry) in scan(path)?.files {
            let key = if pre.is_empty() { rel } else { format!("{pre}/{rel}") };
            all.files.insert(key, entry);
        }
    }
    Ok(all)
}

/// Caminho real de uma chave do índice, pela pasta de prefixo mais longo.
pub fn locate(roots: &[Root], key: &str) -> Option<std::path::PathBuf> {
    roots
        .iter()
        .filter_map(|(label, path)| {
            let pre = prefix(label);
            if pre.is_empty() {
                Some((0, path.join(key)))
            } else {
                key.strip_prefix(&format!("{pre}/")).map(|rel| (pre.len(), path.join(rel)))
            }
        })
        .max_by_key(|(len, _)| *len)
        .map(|(_, p)| p)
}
