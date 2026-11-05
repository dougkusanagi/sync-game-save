//! Caminhos lógicos: o save é guardado com marcadores (`<winAppData>/Jogo`),
//! para restaurar em outro PC, com outro usuário ou outra pasta de instalação.

use std::path::{Path, PathBuf};

/// Valores dos marcadores neste PC.
#[derive(Debug, Clone, Default)]
pub struct Env {
    pub vars: Vec<(String, PathBuf)>,
}

impl Env {
    pub fn new<I: IntoIterator<Item = (&'static str, PathBuf)>>(vars: I) -> Self {
        Env {
            vars: vars.into_iter().map(|(k, v)| (k.to_string(), v)).collect(),
        }
    }
}

/// Troca o marcador inicial (`<base>/x`) pelo caminho real. `None` se o marcador
/// for desconhecido ou o caminho tiver `..` (nunca sai da pasta do marcador).
pub fn resolve(logical: &str, env: &Env) -> Option<PathBuf> {
    let rest = logical.strip_prefix('<')?;
    let (name, tail) = rest.split_once('>')?;
    let root = &env.vars.iter().find(|(k, _)| k == name)?.1;
    let tail = tail.trim_start_matches(['/', '\\']);
    if tail.split(['/', '\\']).any(|p| p == "..") {
        return None;
    }
    Some(if tail.is_empty() { root.clone() } else { root.join(tail) })
}

/// Converte um caminho real em lógico, usando o marcador de prefixo mais longo.
/// Sem marcador que sirva, devolve `None` (o caminho fica absoluto e não é portátil).
pub fn to_logical(path: &Path, env: &Env) -> Option<String> {
    let (name, root) = env
        .vars
        .iter()
        .filter(|(_, root)| path.starts_with(root))
        .max_by_key(|(_, root)| root.components().count())?;
    let rel = path
        .strip_prefix(root)
        .ok()?
        .components()
        .map(|c| c.as_os_str().to_string_lossy().into_owned())
        .collect::<Vec<_>>()
        .join("/");
    Some(if rel.is_empty() { format!("<{name}>") } else { format!("<{name}>/{rel}") })
}
