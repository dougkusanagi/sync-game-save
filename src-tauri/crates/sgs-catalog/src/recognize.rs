//! Reconhecimento de um jogo pela pasta de instalação: nome, executável e caminhos de save.

use crate::{Catalog, Hit, Result};
use serde::Serialize;
use sgs_core::paths::{resolve, Env};
use std::path::{Path, PathBuf};

#[derive(Debug, Clone, PartialEq, Eq, Serialize)]
pub enum Confidence {
    /// Nome da pasta igual a uma pasta de instalação do catálogo.
    High,
    /// Achado só pela busca; o usuário deve conferir.
    Low,
    /// Nada no catálogo.
    None,
}

#[derive(Debug, Clone, PartialEq, Eq, Serialize)]
pub struct SaveCandidate {
    /// Caminho com marcador, como no catálogo (`<winDocuments>/SavedGames/Jogo`).
    pub logical: String,
    pub resolved: PathBuf,
    pub exists: bool,
    pub files: usize,
    /// Marcado por padrão: regra de save (não config) cujo caminho existe.
    pub selected: bool,
}

#[derive(Debug, Clone, PartialEq, Eq, Serialize)]
pub struct Recognition {
    pub name: String,
    pub catalog_id: Option<i64>,
    pub confidence: Confidence,
    pub exe: Option<PathBuf>,
    pub save_candidates: Vec<SaveCandidate>,
    /// Outros jogos possíveis, para o usuário trocar.
    pub alternatives: Vec<Hit>,
}

/// Tira o que não faz parte do nome: `[GOG]`, `(x64)`, `v1.2.3`, `_` e `.`.
pub fn clean_folder_name(name: &str) -> String {
    let mut out = String::new();
    let mut depth = 0;
    for c in name.chars() {
        match c {
            '[' | '(' => depth += 1,
            ']' | ')' => depth = (depth - 1).max(0),
            _ if depth == 0 => out.push(if c == '_' { ' ' } else { c }),
            _ => {}
        }
    }
    let words: Vec<String> = out
        .split_whitespace()
        .filter(|w| {
            let l = w.to_lowercase();
            // Versões como v1.2.3 saem antes de os pontos virarem espaço.
            let is_version = l.len() > 1 && l.starts_with('v') && l[1..].chars().all(|c| c.is_ascii_digit() || c == '.');
            !is_version && !matches!(l.as_str(), "gog" | "repack" | "portable")
        })
        .map(|w| {
            // "Game.v1.2.3": corta na versão colada ao nome.
            let cut = w.char_indices().find(|(i, c)| *c == '.' && w[i + 1..].starts_with('v') && w[i + 2..].starts_with(|d: char| d.is_ascii_digit()));
            let w: &str = match cut {
                Some((i, _)) => &w[..i],
                None => &w[..],
            };
            w.replace('.', " ")
        })
        .collect();
    words.join(" ").split_whitespace().collect::<Vec<_>>().join(" ")
}

const BAD_EXE: &[&str] = &[
    "unins", "setup", "install", "crash", "redist", "vcredist", "dxsetup", "dotnet", "launcherhelper", "updater",
];

fn is_bad_exe(file: &str) -> bool {
    let l = file.to_lowercase();
    BAD_EXE.iter().any(|b| l.contains(b))
}

/// Maior `.exe` da pasta (até 2 níveis), ignorando instaladores e ajudantes.
pub fn guess_exe(dir: &Path) -> Option<PathBuf> {
    let mut best: Option<(u64, PathBuf)> = None;
    fn walk(dir: &Path, depth: u32, best: &mut Option<(u64, PathBuf)>) {
        let Ok(rd) = std::fs::read_dir(dir) else { return };
        for e in rd.flatten() {
            let p = e.path();
            if p.is_dir() && depth < 2 {
                walk(&p, depth + 1, best);
            } else if p.extension().is_some_and(|x| x.eq_ignore_ascii_case("exe")) {
                let name = p.file_name().unwrap().to_string_lossy().into_owned();
                if is_bad_exe(&name) {
                    continue;
                }
                let size = e.metadata().map(|m| m.len()).unwrap_or(0);
                if best.as_ref().is_none_or(|(s, _)| size > *s) {
                    *best = Some((size, p));
                }
            }
        }
    }
    walk(dir, 0, &mut best);
    best.map(|(_, p)| p)
}

fn count_files(p: &Path) -> usize {
    if p.is_file() {
        return 1;
    }
    std::fs::read_dir(p)
        .map(|rd| rd.flatten().map(|e| count_files(&e.path())).sum())
        .unwrap_or(0)
}

/// `dir` é a pasta do jogo. `windows` escolhe as regras do Windows (senão, Linux).
pub fn recognize(cat: &Catalog, dir: &Path, env: &Env, windows: bool) -> Result<Recognition> {
    let folder = dir.file_name().map(|n| n.to_string_lossy().into_owned()).unwrap_or_default();
    let cleaned = clean_folder_name(&folder);
    let mut hits = cat.by_install_dir(&folder)?;
    let mut confidence = Confidence::High;
    if hits.is_empty() && !cleaned.is_empty() {
        hits = cat.by_install_dir(&cleaned)?;
    }
    if hits.is_empty() {
        hits = cat.search(&cleaned, 5)?;
        confidence = if hits.is_empty() { Confidence::None } else { Confidence::Low };
    }

    // Entre vários jogos com a mesma pasta, prefere o que tem save existente no disco.
    let mut base_env = env.clone();
    base_env.vars.retain(|(k, _)| k != "base");
    base_env.vars.push(("base".into(), dir.to_path_buf()));
    let mut scored: Vec<(usize, Hit, Vec<SaveCandidate>)> = Vec::new();
    for h in &hits {
        let cands = save_candidates(cat, h.id, &base_env, windows)?;
        let score = cands.iter().filter(|c| c.selected).count();
        scored.push((score, h.clone(), cands));
    }
    scored.sort_by(|a, b| b.0.cmp(&a.0));
    let best = scored.first().cloned();

    let (name, catalog_id, save_candidates) = match &best {
        Some((_, h, c)) => (h.name.clone(), Some(h.id), c.clone()),
        None => (if cleaned.is_empty() { folder.clone() } else { cleaned.clone() }, None, vec![]),
    };
    // O catálogo às vezes nomeia o jogo só pelo id da Steam; nesse caso a pasta dá o nome melhor.
    let name = if name.chars().all(|c| c.is_ascii_digit()) && !cleaned.is_empty() { cleaned } else { name };

    // Executável: o do `launch` do catálogo, se existir; senão, o maior .exe.
    let mut exe = None;
    if let Some(id) = catalog_id {
        let mut base_env = env.clone();
        base_env.vars.retain(|(k, _)| k != "base");
        base_env.vars.push(("base".into(), dir.to_path_buf()));
        for l in cat.launch_paths(id)? {
            if let Some(p) = resolve(&l, &base_env) {
                let is_exe = p.extension().is_some_and(|x| x.eq_ignore_ascii_case("exe"));
                if p.is_file() && (is_exe || !windows) {
                    exe = Some(p);
                    break;
                }
            }
        }
    }
    let exe = exe.or_else(|| guess_exe(dir));
    let alternatives = scored.into_iter().skip(1).map(|(_, h, _)| h).collect();
    Ok(Recognition { name, catalog_id, confidence, exe, save_candidates, alternatives })
}

/// Aplica uma escolha explícita do catálogo à pasta selecionada.
pub fn recognize_choice(cat: &Catalog, dir: &Path, env: &Env, windows: bool, id: i64) -> Result<Option<Recognition>> {
    let Some(hit) = cat.game(id)? else { return Ok(None) };
    let mut base_env = env.clone();
    base_env.vars.retain(|(k, _)| k != "base");
    base_env.vars.push(("base".into(), dir.to_path_buf()));
    let mut exe = None;
    for path in cat.launch_paths(id)? {
        if let Some(p) = resolve(&path, &base_env) {
            if p.is_file() && (p.extension().is_some_and(|x| x.eq_ignore_ascii_case("exe")) || !windows) {
                exe = Some(p);
                break;
            }
        }
    }
    let folder = dir.file_name().map(|n| n.to_string_lossy().into_owned()).unwrap_or_default();
    let cleaned = clean_folder_name(&folder);
    let name = if hit.name.chars().all(|c| c.is_ascii_digit()) && !cleaned.is_empty() { cleaned } else { hit.name };
    Ok(Some(Recognition { name, catalog_id: Some(id), confidence: Confidence::High,
        exe: exe.or_else(|| guess_exe(dir)), save_candidates: save_candidates(cat, id, &base_env, windows)?, alternatives: vec![] }))
}

fn save_candidates(cat: &Catalog, id: i64, env: &Env, windows: bool) -> Result<Vec<SaveCandidate>> {
    let os = if windows { "windows" } else { "linux" };
    let mut out: Vec<SaveCandidate> = Vec::new();
    for sp in cat.save_paths(id)? {
        if sp.os.as_deref().is_some_and(|o| o != os) {
            continue;
        }
        let is_save = sp.tags.split(',').any(|t| t == "save");
        let is_config_only = sp.tags.split(',').any(|t| t == "config") && !is_save;
        if is_config_only {
            continue;
        }
        // Marcadores que variam por usuário da loja ou curingas não resolvem para um caminho único.
        if sp.path.contains('*') || sp.path.contains("<storeUserId>") {
            continue;
        }
        let Some(resolved) = resolve(&sp.path, env) else { continue };
        if out.iter().any(|c| c.logical == sp.path) {
            continue;
        }
        let exists = resolved.exists();
        // Pastas de emuladores/cracks (regra sem tag) que não existem só poluem a lista.
        if !exists && sp.tags.is_empty() {
            continue;
        }
        let files = if exists { count_files(&resolved) } else { 0 };
        out.push(SaveCandidate {
            logical: sp.path,
            resolved,
            exists,
            files,
            selected: is_save && exists && files > 0,
        });
    }
    // Os que existem primeiro.
    out.sort_by_key(|c| (!c.selected, !c.exists));
    Ok(out)
}


#[derive(Debug, Default, Serialize)]
pub struct ScanResult {
    /// Reconhecidos com certeza alta.
    pub recognized: Vec<(PathBuf, Recognition)>,
    /// Achados só pela busca: precisam de confirmação.
    pub needs_confirmation: Vec<(PathBuf, Recognition)>,
    /// Pastas com executável que o catálogo não conhece.
    pub unknown: Vec<(PathBuf, Recognition)>,
    /// Pastas que não puderam ser examinadas ou reconhecidas.
    pub errors: Vec<(PathBuf, String)>,
}

fn has_exe(dir: &Path, depth: u32, errors: &mut Vec<(PathBuf, String)>) -> bool {
    let rd = match std::fs::read_dir(dir) {
        Ok(rd) => rd,
        Err(e) => { errors.push((dir.to_path_buf(), e.to_string())); return false; }
    };
    let mut found = false;
    for entry in rd {
        match entry {
            Ok(e) => {
                let p = e.path();
                if p.is_dir() {
                    if depth < 2 && has_exe(&p, depth + 1, errors) { found = true; }
                } else if p.extension().is_some_and(|x| x.eq_ignore_ascii_case("exe")) && !is_bad_exe(&e.file_name().to_string_lossy()) {
                    found = true;
                }
            }
            Err(e) => errors.push((dir.to_path_buf(), e.to_string())),
        }
    }
    found
}

// Pastas técnicas comuns podem conter o executável sem serem a pasta do jogo.
fn has_exe_at_game_root(dir: &Path) -> bool {
    let Ok(entries) = std::fs::read_dir(dir) else { return false };
    entries.flatten().any(|entry| {
        let path = entry.path();
        if path.is_file() {
            path.extension().is_some_and(|x| x.eq_ignore_ascii_case("exe"))
                && !is_bad_exe(&entry.file_name().to_string_lossy())
        } else if path.is_dir() {
            let name = entry.file_name().to_string_lossy().to_lowercase();
            matches!(name.as_str(), "bin" | "binaries" | "binary" | "x64" | "x86" | "win64" | "win32")
                && has_exe(&path, 0, &mut Vec::new())
        } else {
            false
        }
    })
}

/// Varre as subpastas de `root` até `depth` níveis (1 = filhas diretas). Cada pasta com
/// executável vira um jogo; não desce dentro de uma pasta já tratada como jogo.
pub fn scan_folder(cat: &Catalog, root: &Path, depth: u32, env: &Env, windows: bool) -> Result<ScanResult> {
    let mut out = ScanResult::default();
    fn walk(cat: &Catalog, dir: &Path, left: u32, env: &Env, windows: bool, out: &mut ScanResult) -> Result<()> {
        let rd = match std::fs::read_dir(dir) {
            Ok(rd) => rd,
            Err(e) => { out.errors.push((dir.to_path_buf(), e.to_string())); return Ok(()); }
        };
        let mut dirs = Vec::new();
        for entry in rd {
            match entry {
                Ok(e) if e.path().is_dir() => dirs.push(e.path()),
                Ok(_) => {},
                Err(e) => out.errors.push((dir.to_path_buf(), e.to_string())),
            }
        }
        dirs.sort();
        for d in dirs {
            if has_exe(&d, 0, &mut out.errors) {
                let r = match recognize(cat, &d, env, windows) {
                    Ok(r) => r,
                    Err(e) => { out.errors.push((d, e.to_string())); continue; }
                };
                if r.confidence != Confidence::None || has_exe_at_game_root(&d) {
                    match r.confidence {
                        Confidence::High => out.recognized.push((d, r)),
                        Confidence::Low => out.needs_confirmation.push((d, r)),
                        Confidence::None => out.unknown.push((d, r)),
                    }
                } else if left > 1 {
                    walk(cat, &d, left - 1, env, windows, out)?;
                }
            } else if left > 1 {
                walk(cat, &d, left - 1, env, windows, out)?;
            }
        }
        Ok(())
    }
    walk(cat, root, depth.max(1), env, windows, &mut out)?;
    Ok(out)
}
