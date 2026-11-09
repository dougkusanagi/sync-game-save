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
    create_roots_with(game_dir, roots, id, device, None, &mut copy_into)
}

pub(crate) fn create_roots_checked(game_dir: &Path, roots: &[Root], id: &str, device: &str, expected: &Index) -> Result<Head> {
    create_roots_with(game_dir, roots, id, device, Some(expected), &mut copy_into)
}

fn create_roots_with(game_dir: &Path, roots: &[Root], id: &str, device: &str, expected: Option<&Index>, copy: &mut dyn FnMut(&Path, &Path) -> Result<()>) -> Result<Head> {
    let index = scan_roots(roots)?;
    if expected.is_some_and(|e| e.digest() != index.digest()) { return Err(Error::SaveChanged); }
    let snap = game_dir.join("snapshots").join(id);
    let files = snap.join("files");
    let snapshots = game_dir.join("snapshots");
    std::fs::create_dir_all(&snapshots).ctx(&snapshots)?;
    // Reservar o diretório de forma atômica impede sobrescrever um snapshot existente.
    std::fs::create_dir(&snap).ctx(&snap)?;
    std::fs::create_dir(&files).ctx(&files)?;
    for rel in index.files.keys() {
        let src = locate(roots, rel).expect("toda chave vem de uma pasta");
        copy(&src, &files.join(rel))?;
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

#[cfg(test)]
mod tests {
    use super::*;

    fn disco_cheio(path: &Path) -> Error {
        let codigo = if cfg!(windows) { 112 } else { 28 };
        Error::Io { path: path.into(), source: std::io::Error::from_raw_os_error(codigo) }
    }

    #[test]
    fn disco_cheio_durante_backup_ou_restauracao_preserva_copia_local() {
        let temp = tempfile::tempdir().unwrap();
        let save = temp.path().join("save");
        let cloud = temp.path().join("cloud");
        std::fs::create_dir(&save).unwrap();
        std::fs::write(save.join("a"), "nuvem a").unwrap();
        std::fs::write(save.join("b"), "nuvem b").unwrap();
        create(&cloud, &save, "s1", "PC").unwrap();
        std::fs::write(save.join("a"), "local a").unwrap();
        std::fs::write(save.join("b"), "local b").unwrap();
        let roots = [(String::new(), save.clone())];
        let snap = cloud.join("snapshots/s1");
        let mut copias = 0;
        let r = restore_roots_with(&snap, &roots, &temp.path().join("backup-falha"), &mut |src, dst| {
            copias += 1;
            if copias == 2 { return Err(disco_cheio(dst)); }
            copy_into(src, dst)
        });
        assert!(matches!(r, Err(Error::Io { source, .. }) if source.raw_os_error() == Some(if cfg!(windows) { 112 } else { 28 })));
        assert_eq!(std::fs::read_to_string(save.join("a")).unwrap(), "local a");
        assert_eq!(std::fs::read_to_string(save.join("b")).unwrap(), "local b");

        let backup = temp.path().join("backup-completo");
        copias = 0;
        let r = restore_roots_with(&snap, &roots, &backup, &mut |src, dst| {
            copias += 1;
            if copias == 4 { return Err(disco_cheio(dst)); }
            copy_into(src, dst)
        });
        assert!(r.is_err());
        assert_eq!(std::fs::read_to_string(backup.join("a")).unwrap(), "local a");
        assert_eq!(std::fs::read_to_string(backup.join("b")).unwrap(), "local b");
        assert!(verify(&snap).is_ok());
    }

    #[test]
    fn recusa_snapshot_se_save_mudar_apos_verificacao() {
        let temp = tempfile::tempdir().unwrap();
        let save = temp.path().join("save");
        std::fs::create_dir(&save).unwrap();
        std::fs::write(save.join("slot"), "antes").unwrap();
        let roots = [(String::new(), save.clone())];
        let esperado = scan_roots(&roots).unwrap();
        std::fs::write(save.join("slot"), "depois").unwrap();
        let cloud = temp.path().join("cloud");
        assert!(matches!(create_roots_checked(&cloud, &roots, "id", "PC", &esperado), Err(Error::SaveChanged)));
        assert!(read_head(&cloud).unwrap().is_none());
        assert_eq!(std::fs::read_to_string(save.join("slot")).unwrap(), "depois");
    }

    #[test]
    fn falha_depois_de_uma_copia_preserva_head_e_arquivos_locais() {
        let temp = tempfile::tempdir().unwrap();
        let save = temp.path().join("save");
        let cloud = temp.path().join("cloud");
        std::fs::create_dir(&save).unwrap();
        std::fs::write(save.join("a"), "um").unwrap();
        std::fs::write(save.join("b"), "dois").unwrap();
        let head = create(&cloud, &save, "antigo", "PC").unwrap();
        let mut count = 0;
        let result = create_roots_with(&cloud, &[(String::new(), save.clone())], "novo", "PC", None, &mut |src, dst| {
            count += 1;
            if count == 2 { return Err(disco_cheio(dst)); }
            copy_into(src, dst)
        });
        assert!(result.is_err());
        assert_eq!(count, 2);
        assert_eq!(read_head(&cloud).unwrap(), Some(head));
        assert_eq!(std::fs::read_to_string(save.join("a")).unwrap(), "um");
        assert_eq!(std::fs::read_to_string(save.join("b")).unwrap(), "dois");
        assert_eq!(std::fs::read_to_string(cloud.join("snapshots/novo/files/a")).unwrap(), "um");
    }
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
    restore_roots_with(snap, roots, backup_dir, &mut copy_into)
}

fn restore_roots_with(snap: &Path, roots: &[Root], backup_dir: &Path, copy: &mut dyn FnMut(&Path, &Path) -> Result<()>) -> Result<()> {
    let index = verify(snap)?;
    let current = scan_roots(roots)?;
    let parent = backup_dir.parent().expect("backup tem pasta pai");
    std::fs::create_dir_all(parent).ctx(parent)?;
    // Nunca misture backups de restaurações distintas, mesmo no mesmo segundo.
    std::fs::create_dir(backup_dir).ctx(backup_dir)?;
    for rel in current.files.keys() {
        let src = locate(roots, rel).expect("toda chave vem de uma pasta");
        copy(&src, &backup_dir.join(rel))?;
    }
    for rel in current.files.keys().filter(|r| !index.files.contains_key(*r)) {
        let p = locate(roots, rel).expect("toda chave vem de uma pasta");
        std::fs::remove_file(&p).ctx(&p)?;
    }
    for rel in index.files.keys() {
        let Some(dst) = locate(roots, rel) else { continue };
        copy(&snap.join("files").join(rel), &dst)?;
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
    let head = read_head(game_dir)?.map(|h| h.snapshot);
    ids.sort_by_key(|id| {
        let timestamp = id.split('_').next().and_then(|n| n.parse::<u64>().ok()).unwrap_or(0);
        let sequencia = id.rsplit_once("~sgs-").and_then(|(_, n)| n.parse::<u64>().ok()).unwrap_or(0);
        std::cmp::Reverse((head.as_deref() == Some(id.as_str()), timestamp, sequencia))
    });
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
