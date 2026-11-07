use crate::index::{hash_file, locate, scan_roots, Index, Root};
use crate::{Error, IoCtx, Result};
use serde::{Deserialize, Serialize};
use std::path::{Path, PathBuf};

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct Head {
    pub snapshot: String,
    pub index_digest: String,
    pub device: String,
}

fn head_path(game_dir: &Path) -> PathBuf {
    game_dir.join("head.json")
}

pub fn read_head(game_dir: &Path) -> Result<Option<Head>> {
    let p = head_path(game_dir);
    if !p.exists() {
        return Ok(None);
    }
    Ok(Some(serde_json::from_slice(&std::fs::read(&p).ctx(&p)?)?))
}

fn copy_into(src: &Path, dst: &Path) -> Result<()> {
    std::fs::create_dir_all(dst.parent().expect("destino tem pasta pai")).ctx(dst)?;
    crate::retry(|| std::fs::copy(src, dst)).ctx(dst)?;
    Ok(())
}

/// Cria um snapshot imutável de `save_dir` em `game_dir/snapshots/<id>/`.
/// Ordem: copia os arquivos, grava `index.json`, relê e confere os hashes e
/// só então atualiza `head.json`. Uma falha antes disso deixa o `head` intacto.
pub fn create(game_dir: &Path, save_dir: &Path, id: &str, device: &str) -> Result<Head> {
    create_roots(game_dir, &[(String::new(), save_dir.to_path_buf())], id, device)
}

/// Igual a `create`, para várias pastas de save.
pub fn create_roots(game_dir: &Path, roots: &[Root], id: &str, device: &str) -> Result<Head> {
    let index = scan_roots(roots)?;
    let snap = game_dir.join("snapshots").join(id);
    let files = snap.join("files");
    std::fs::create_dir_all(&files).ctx(&files)?;
    for rel in index.files.keys() {
        let src = locate(roots, rel).expect("toda chave vem de uma pasta");
        copy_into(&src, &files.join(rel))?;
    }
    let idx_path = snap.join("index.json");
    std::fs::write(&idx_path, serde_json::to_vec_pretty(&index)?).ctx(&idx_path)?;
    verify(&snap)?;
    let head = Head {
        snapshot: id.to_string(),
        index_digest: index.digest(),
        device: device.to_string(),
    };
    // Grava em arquivo temporário e renomeia, para o `head` nunca ficar pela metade.
    let tmp = game_dir.join("head.json.tmp");
    std::fs::write(&tmp, serde_json::to_vec_pretty(&head)?).ctx(&tmp)?;
    std::fs::rename(&tmp, head_path(game_dir)).ctx(&tmp)?;
    Ok(head)
}

/// Confere se todos os arquivos do snapshot existem e batem com o `index.json`.
pub fn verify(snap: &Path) -> Result<Index> {
    let p = snap.join("index.json");
    let index: Index = serde_json::from_slice(&std::fs::read(&p).ctx(&p)?)?;
    for (rel, entry) in &index.files {
        let got = hash_file(&snap.join("files").join(rel))?;
        if &got != entry {
            return Err(Error::HashMismatch(rel.clone()));
        }
    }
    Ok(index)
}

/// Restaura um snapshot em `save_dir`. Antes, copia o save atual para `backup_dir`.
/// Se o backup falhar, nada é sobrescrito.
pub fn restore(snap: &Path, save_dir: &Path, backup_dir: &Path) -> Result<()> {
    restore_roots(snap, &[(String::new(), save_dir.to_path_buf())], backup_dir)
}

/// Igual a `restore`, para várias pastas de save.
pub fn restore_roots(snap: &Path, roots: &[Root], backup_dir: &Path) -> Result<()> {
    let index = verify(snap)?;
    let current = scan_roots(roots)?;
    for rel in current.files.keys() {
        let src = locate(roots, rel).expect("toda chave vem de uma pasta");
        copy_into(&src, &backup_dir.join(rel))?;
    }
    for rel in current.files.keys().filter(|r| !index.files.contains_key(*r)) {
        let p = locate(roots, rel).expect("toda chave vem de uma pasta");
        std::fs::remove_file(&p).ctx(&p)?;
    }
    for rel in index.files.keys() {
        let Some(dst) = locate(roots, rel) else { continue };
        copy_into(&snap.join("files").join(rel), &dst)?;
    }
    Ok(())
}

/// Snapshots completos (com `index.json`), do mais novo para o mais antigo (pelo nome do id).
/// Os incompletos, de uma queda no meio da escrita, ficam de fora.
pub fn list(game_dir: &Path) -> Result<Vec<String>> {
    let dir = game_dir.join("snapshots");
    let mut ids = Vec::new();
    if dir.is_dir() {
        for item in std::fs::read_dir(&dir).ctx(&dir)? {
            let p = item.ctx(&dir)?.path();
            if p.join("index.json").is_file() {
                ids.push(p.file_name().unwrap().to_string_lossy().into_owned());
            }
        }
    }
    ids.sort_by_key(|id| std::cmp::Reverse(id.split('_').next().and_then(|n| n.parse::<u64>().ok()).unwrap_or(0)));
    Ok(ids)
}

/// Mantém os `keep` mais novos (mínimo 1) e o do `head`; apaga o resto.
/// Devolve os ids apagados.
pub fn prune(game_dir: &Path, keep: usize) -> Result<Vec<String>> {
    let head = read_head(game_dir)?.map(|h| h.snapshot);
    let mut removed = Vec::new();
    for id in list(game_dir)?.into_iter().skip(keep.max(1)) {
        if Some(&id) == head.as_ref() {
            continue;
        }
        let p = game_dir.join("snapshots").join(&id);
        std::fs::remove_dir_all(&p).ctx(&p)?;
        removed.push(id);
    }
    Ok(removed)
}
