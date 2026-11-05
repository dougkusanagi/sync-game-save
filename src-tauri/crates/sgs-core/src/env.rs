//! Preenche os marcadores de caminho (`<home>`, `<winAppData>`...) com as pastas reais deste PC.

use crate::paths::Env;
use std::path::PathBuf;

fn var(name: &str) -> Option<PathBuf> {
    std::env::var_os(name).filter(|v| !v.is_empty()).map(PathBuf::from)
}

/// `base` é a pasta de instalação do jogo (marcador `<base>`).
pub fn system_env(base: Option<PathBuf>) -> Env {
    let mut vars: Vec<(String, PathBuf)> = Vec::new();
    let mut add = |k: &str, v: Option<PathBuf>| {
        if let Some(v) = v {
            vars.push((k.to_string(), v));
        }
    };
    add("base", base);
    let home = var("USERPROFILE").or_else(|| var("HOME"));
    add("home", home.clone());
    // Windows
    add("winAppData", var("APPDATA"));
    add("winLocalAppData", var("LOCALAPPDATA"));
    add("winPublic", var("PUBLIC"));
    add("winProgramData", var("PROGRAMDATA"));
    // Documentos pode estar redirecionado (OneDrive, outro nome de pasta): pergunta ao sistema.
    add("winDocuments", dirs::document_dir().or_else(|| home.as_ref().map(|h| h.join("Documents"))));
    // Linux
    add(
        "xdgData",
        var("XDG_DATA_HOME").or_else(|| home.as_ref().map(|h| h.join(".local/share"))),
    );
    add(
        "xdgConfig",
        var("XDG_CONFIG_HOME").or_else(|| home.as_ref().map(|h| h.join(".config"))),
    );
    Env { vars }
}
