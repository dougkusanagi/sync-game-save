//! Temas: pacotes locais que trocam só os tokens visuais (`--sgs-*`).
//! O pacote é JSON; nenhum CSS ou JavaScript do tema é executado. Os valores passam por
//! uma lista restrita para não abrir URL externa nem injetar regras.

use serde::{Deserialize, Serialize};
use std::collections::BTreeMap;
use std::path::Path;

pub const PADRAO: &str = "escuro";
const LIMITE_TOKENS: usize = 200;
const LIMITE_VALOR: usize = 160;

#[derive(Debug, Clone, Serialize, PartialEq, Eq)]
pub struct Theme {
    pub id: String,
    pub name: String,
    pub author: String,
    pub version: String,
    pub builtin: bool,
    pub tokens: BTreeMap<String, String>,
}

#[derive(Debug, Clone, Serialize)]
pub struct ThemeList {
    pub atual: String,
    pub temas: Vec<Theme>,
}

#[derive(Deserialize)]
struct ThemeFile {
    name: String,
    #[serde(default)]
    author: String,
    #[serde(default)]
    version: String,
    #[serde(default)]
    min_app_version: Option<String>,
    #[serde(default)]
    tokens: BTreeMap<String, String>,
}

fn theme(id: &str, name: &str, author: &str, tokens: &[(&str, &str)]) -> Theme {
    Theme {
        id: id.into(),
        name: name.into(),
        author: author.into(),
        version: env!("CARGO_PKG_VERSION").into(),
        builtin: true,
        tokens: tokens.iter().map(|(k, v)| ((*k).into(), (*v).into())).collect(),
    }
}

/// Tema escuro (o padrão, sem trocas) e o claro oficial, que também serve de exemplo.
pub fn builtin_themes() -> Vec<Theme> {
    vec![
        theme("escuro", "Escuro (padrão)", "Sync Game Save", &[]),
        theme(
            "claro",
            "Claro",
            "Sync Game Save",
            &[
                ("--sgs-bg", "#f2f5fb"),
                ("--sgs-bg-rgb", "242 245 251"),
                ("--sgs-surface", "rgb(255 255 255 / 0.72)"),
                ("--sgs-surface-strong", "rgb(255 255 255 / 0.92)"),
                ("--sgs-surface-solid", "#ffffff"),
                ("--sgs-surface-hover", "rgb(16 22 37 / 0.06)"),
                ("--sgs-border", "rgb(16 22 37 / 0.14)"),
                ("--sgs-border-strong", "rgb(16 22 37 / 0.32)"),
                ("--sgs-text", "#101625"),
                ("--sgs-text-dim", "rgb(16 22 37 / 0.72)"),
                ("--sgs-text-faint", "rgb(16 22 37 / 0.55)"),
                ("--sgs-accent-default", "#1d6fe0"),
                ("--sgs-ok", "#0f8a60"),
                ("--sgs-info", "#1d6fe0"),
                ("--sgs-warn", "#9a6500"),
                ("--sgs-danger", "#c62d45"),
                ("--sgs-tier-common", "#5b6b82"),
                ("--sgs-tier-uncommon", "#0f8a60"),
                ("--sgs-tier-rare", "#1d6fe0"),
                ("--sgs-tier-ultra", "#9a6500"),
                ("--sgs-focus-ring", "0 0 0 0.1875rem #1d6fe0, 0 0.75rem 2.5rem rgb(16 22 37 / 0.2)"),
                ("--sgs-capa-bg", "linear-gradient(135deg, #dfe7f5, #eef2f9)"),
                ("--sgs-color-scheme", "light"),
            ],
        ),
    ]
}

/// Lê `themes/<id>/theme.json` na pasta de dados. Pacotes inválidos são ignorados.
pub fn installed_themes(data_dir: &Path) -> Vec<Theme> {
    let Ok(dir) = std::fs::read_dir(data_dir.join("themes")) else {
        return vec![];
    };
    let mut lista: Vec<Theme> = dir
        .flatten()
        .filter(|e| e.path().is_dir())
        .filter_map(|e| {
            let id = e.file_name().to_str()?.to_owned();
            if !valid_id(&id) || id == PADRAO || id == "claro" {
                return None;
            }
            let bruto = std::fs::read_to_string(e.path().join("theme.json")).ok()?;
            let arquivo: ThemeFile = serde_json::from_str(&bruto).ok()?;
            from_file(&id, arquivo)
        })
        .collect();
    lista.sort_by_key(|t| t.name.to_lowercase());
    lista
}

fn from_file(id: &str, arquivo: ThemeFile) -> Option<Theme> {
    let name = arquivo.name.trim();
    if name.is_empty() || name.chars().count() > 60 {
        return None;
    }
    if let Some(min) = arquivo.min_app_version.as_deref().map(str::trim).filter(|m| !m.is_empty()) {
        if !version_at_least(min, env!("CARGO_PKG_VERSION")) {
            return None;
        }
    }
    let autor = arquivo.author.trim();
    let versao = arquivo.version.trim();
    let tokens = arquivo
        .tokens
        .into_iter()
        .filter(|(k, v)| valid_token(k, v))
        .take(LIMITE_TOKENS)
        .collect();
    Some(Theme {
        id: id.into(),
        name: name.into(),
        author: if autor.is_empty() { "Autor não informado".into() } else { autor.chars().take(60).collect() },
        version: if versao.is_empty() { "1.0.0".into() } else { versao.chars().take(20).collect() },
        builtin: false,
        tokens,
    })
}

fn valid_id(id: &str) -> bool {
    !id.is_empty() && id.len() <= 40 && id.chars().all(|c| c.is_ascii_alphanumeric() || c == '-' || c == '_')
}

fn valid_token(chave: &str, valor: &str) -> bool {
    let Some(nome) = chave.strip_prefix("--sgs-") else {
        return false;
    };
    if nome.is_empty() || chave.len() > 48 || !nome.chars().all(|c| c.is_ascii_lowercase() || c.is_ascii_digit() || c == '-') {
        return false;
    }
    let valor = valor.trim();
    if valor.is_empty() || valor.len() > LIMITE_VALOR {
        return false;
    }
    let baixo = valor.to_ascii_lowercase();
    if baixo.contains('@') || baixo.contains("url(") || baixo.contains("expression(") {
        return false;
    }
    !valor.chars().any(|c| c.is_control() || matches!(c, ';' | '{' | '}' | '<' | '>' | '\\'))
}

fn version_at_least(minima: &str, atual: &str) -> bool {
    match (parse_version(minima), parse_version(atual)) {
        (Some(m), Some(a)) => a >= m,
        _ => false,
    }
}

fn parse_version(v: &str) -> Option<(u64, u64, u64)> {
    let nucleo = v.trim().split(['-', '+']).next()?;
    let mut partes = nucleo.split('.');
    let maior = partes.next()?.parse().ok()?;
    let menor = partes.next().unwrap_or("0").parse().ok()?;
    let correcao = partes.next().unwrap_or("0").parse().ok()?;
    if partes.next().is_some() {
        return None;
    }
    Some((maior, menor, correcao))
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::fs;

    fn tmp() -> tempfile::TempDir {
        tempfile::tempdir().unwrap()
    }

    #[test]
    fn embutidos_incluem_escuro_e_claro() {
        let t = builtin_themes();
        assert_eq!(t[0].id, PADRAO);
        assert!(t.iter().all(|x| x.builtin));
        let claro = t.iter().find(|x| x.id == "claro").unwrap();
        assert_eq!(claro.tokens.get("--sgs-color-scheme").map(String::as_str), Some("light"));
        assert!(claro.tokens.contains_key("--sgs-bg"));
    }

    #[test]
    fn aceita_pacote_valido_e_ignora_ruim() {
        let d = tmp();
        let bom = d.path().join("themes/meu-tema");
        fs::create_dir_all(&bom).unwrap();
        fs::write(
            bom.join("theme.json"),
            r##"{"name":"Meu tema","author":"Ana","version":"2.0.0","tokens":{
                "--sgs-bg":"#123456",
                "--sgs-accent":"var(--sgs-accent-default)",
                "cor":"#fff",
                "--sgs-text":"url(https://exemplo.test/x)"
            }}"##,
        )
        .unwrap();

        let futuro = d.path().join("themes/futuro");
        fs::create_dir_all(&futuro).unwrap();
        fs::write(
            futuro.join("theme.json"),
            r##"{"name":"Futuro","min_app_version":"99.0.0","tokens":{"--sgs-bg":"#000"}}"##,
        )
        .unwrap();

        let lista = installed_themes(d.path());
        assert_eq!(lista.len(), 1, "{lista:?}");
        let tema = &lista[0];
        assert_eq!(tema.id, "meu-tema");
        assert_eq!(tema.name, "Meu tema");
        assert_eq!(tema.author, "Ana");
        assert!(!tema.builtin);
        assert_eq!(tema.tokens.len(), 2);
        assert!(!tema.tokens.contains_key("cor"));
        assert!(!tema.tokens.contains_key("--sgs-text"));
    }

    #[test]
    fn tokens_recusam_url_e_caracteres_de_regra() {
        assert!(valid_token("--sgs-bg", "#fff"));
        assert!(valid_token("--sgs-font", "\"Outfit\", system-ui"));
        assert!(!valid_token("--sgs-bg", "url(https://x)"));
        assert!(!valid_token("--sgs-bg", "red; color: blue"));
        assert!(!valid_token("--sgs-bg", "red}body{color:red"));
        assert!(!valid_token("--outra", "#fff"));
        assert!(!valid_token("--sgs-", "#fff"));
    }

    #[test]
    fn compara_versoes() {
        assert!(version_at_least("0.1.0", "0.1.0"));
        assert!(version_at_least("0.1", "0.1.2"));
        assert!(!version_at_least("0.2.0", "0.1.9"));
        assert!(!version_at_least("x", "0.1.0"));
    }
}
