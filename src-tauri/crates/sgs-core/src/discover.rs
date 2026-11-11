//! Descoberta de saves por mudança: o app registra o estado das pastas comuns
//! de save antes de abrir o jogo e, ao fechar, sugere as pastas que mudaram.

use crate::paths::{to_logical, Env};
use serde::{Deserialize, Serialize};
use std::collections::BTreeMap;
use std::path::Path;

/// Pastas nas quais se procura por mudanças (prefixos lógicos, resolvidos com
/// o `Env`). A pasta precisa existir na hora da procura.
const RAIZES: &[&str] = &[
    "<home>/Saved Games",
    "<winDocuments>",
    "<winPublic>/Documents",
    "<winAppData>",
    "<winLocalAppData>",
    "<home>/AppData/LocalLow",
    "<base>",
];

/// Níveis máximos de pastas abaixo da raiz que formam uma pasta candidata
/// (contando a partir da raiz). Arquivos direto na raiz fazem a raiz inteira
/// virar candidata.
const PROFUNDIDADES: &[(&str, usize)] = &[("<home>/AppData/LocalLow", 2), ("<winDocuments>", 1)];

/// Pastas de nome ruído (caches, logs, temporários e crash dumps), comparadas
/// por igualdade exata do nome de pasta.
const RUIDO_PASTA: &[&str] = &[
    "cache",
    "caches",
    "code cache",
    "gpucache",
    "d3dscache",
    "shadercache",
    "shaders",
    "shader",
    "log",
    "logs",
    "temp",
    "tmp",
    "crash",
    "crashes",
    "crashdumps",
    "dump",
    "dump",
    "reports",
];

/// Extensões de arquivos ruído (com o ponto).
const RUIDO_EXT: &[&str] = &[".log", ".dmp", ".tmp", ".temp"];

/// Limite de profundidade de varredura sob cada raiz, para evitar árvores
/// patológicas (as pastas comuns raramente têm mais que isso de save).
const MAX_NIVEIS: usize = 8;

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub struct Fstat {
    pub size: u64,
    /// Nanossegundos desde a era Unix; 0 quando o sistema não informa.
    pub mtime: u64,
}

/// Estado capturado de todos os arquivos das pastas de procura. A chave é o
/// caminho lógico do arquivo.
#[derive(Debug, Clone, Default, PartialEq, Eq, Serialize, Deserialize)]
pub struct Capture {
    pub taken_at: u64,
    pub files: BTreeMap<String, Fstat>,
}

/// Uma pasta que mudou entre a captura e o fim do jogo.
#[derive(Debug, Clone, PartialEq, Eq, Serialize)]
pub struct Candidate {
    /// Pasta com marcador, no mesmo formato dos caminhos de save.
    pub logical: String,
    /// Arquivos novos ou alterados.
    pub added: u64,
    /// Arquivos apagados desde a captura.
    pub removed: u64,
    /// Bytes somados dos arquivos novos e alterados.
    pub bytes: u64,
}

impl Candidate {
    pub fn changed(&self) -> u64 {
        self.added + self.removed
    }
}

fn agora() -> u64 {
    std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .map(|t| t.as_secs())
        .unwrap_or(0)
}

fn file_stat(path: &Path) -> Fstat {
    match std::fs::metadata(path) {
        Ok(m) => Fstat { size: m.len(), mtime: mtime_nanos(&m) },
        Err(_) => Fstat { size: 0, mtime: 0 },
    }
}

fn mtime_nanos(m: &std::fs::Metadata) -> u64 {
    m.modified().ok().and_then(|t| t.duration_since(std::time::UNIX_EPOCH).ok()).map(|d| d.as_nanos() as u64).unwrap_or(0)
}

fn eh_ruido(caminho: &str) -> bool {
    let baixo = caminho.to_lowercase().replace('\\', "/");
    // `nome` é a última pasta do caminho (o próprio caminho, se não houver).
    let nome = baixo.rsplit('/').next().unwrap_or(&baixo);
    if RUIDO_PASTA.contains(&nome) {
        return true;
    }
    match nome.rsplit_once('.') {
        Some((_, ext)) => RUIDO_EXT.contains(&format!(".{ext}").as_str()),
        None => false,
    }
}

fn prof_padrao(raiz: &str) -> usize {
    PROFUNDIDADES.iter().find(|(p, _)| *p == raiz).map(|(_, d)| *d).unwrap_or(1)
}

/// Encontra qual raiz contém o caminho lógico e devolve o restante.
fn encaixar(logical: &str) -> Option<(&'static str, &str)> {
    RAIZES.iter().filter_map(|r| logical.strip_prefix(&format!("{r}/")).map(|rel| (*r, rel))).next()
}

/// Em qual pasta candidata o arquivo (relativo à raiz) cai.
fn grupo(raiz: &str, rel: &str) -> String {
    let segs: Vec<&str> = rel.split('/').collect();
    let base = prof_padrao(raiz);
    // O último segmento é o arquivo: o grupo vai até a pasta antes dele.
    let mut d = base.min(segs.len().saturating_sub(1));
    // Saves no "My Games" ficam dentro da subpasta do jogo.
    if segs.len() >= 2 && segs[0].eq_ignore_ascii_case("my games") {
        d = 2.min(segs.len().saturating_sub(1));
    }
    if d == 0 {
        return raiz.to_string();
    }
    format!("{raiz}/{}", segs[..d].join("/"))
}

/// Chave estável (hex SHA-256 do caminho do executável) para nomear o arquivo
/// de captura de um jogo.
pub fn hex_key(exe: &str) -> String {
    use sha2::Digest;
    let mut h = sha2::Sha256::new();
    h.update(exe.as_bytes());
    crate::index::hex(&h.finalize())
}

/// Registra o estado atual das pastas de procura.
pub fn capture(env: &Env) -> Capture {
    capture_at(env, agora())
}

pub fn capture_at(env: &Env, taken_at: u64) -> Capture {
    let mut files = BTreeMap::new();
    let raizes: Vec<_> = RAIZES.iter().filter_map(|r| crate::paths::resolve(r, env)).collect();
    for dir in raizes {
        if !dir.is_dir() {
            continue;
        }
        // Filas de pastas com o nível restante (0 = sem limite).
        let mut fila = vec![(dir.clone(), 0usize)];
        while let Some((d, nivel)) = fila.pop() {
            // Erro de acesso (antivírus, permissão): pula a pasta sem falhar.
            let Ok(itens) = std::fs::read_dir(&d) else { continue };
            for item in itens.flatten() {
                let path = item.path();
                let Ok(kind) = item.file_type() else { continue };
                if kind.is_dir() {
                    if !eh_ruido(&path.to_string_lossy()) && nivel < MAX_NIVEIS {
                        fila.push((path, nivel + 1));
                    }
                } else {
                    let Some(logical) = to_logical(&path, env) else { continue };
                    if eh_ruido(&logical) {
                        continue;
                    }
                    files.insert(logical, file_stat(&path));
                }
            }
        }
    }
    Capture { taken_at, files }
}

/// Compara a captura com o estado atual e devolve as pastas que mudaram,
/// da mais provável (mais arquivos alterados) para a menos.
pub fn compare(before: &Capture, env: &Env) -> Vec<Candidate> {
    compare_at(before, &capture_at(env, agora()))
}

fn compare_at(before: &Capture, atual: &Capture) -> Vec<Candidate> {
    let mut grupos: BTreeMap<String, Candidate> = BTreeMap::new();
    let mut tocar = |grupo: String, f: Fstat| {
        let e = grupos
            .entry(grupo)
            .or_insert_with(|| Candidate { logical: String::new(), added: 0, removed: 0, bytes: 0 });
        e.added += 1;
        e.bytes += f.size;
    };
    // Arquivos novos ou alterados.
    for (logical, stat) in &atual.files {
        if before.files.get(logical) == Some(stat) {
            continue;
        }
        if let Some((raiz, rel)) = encaixar(logical) {
            tocar(grupo(raiz, rel), *stat);
        }
    }
    // Arquivos apagados desde a captura.
    for logical in before.files.keys() {
        if atual.files.contains_key(logical) {
            continue;
        }
        if let Some((raiz, rel)) = encaixar(logical) {
            let e = grupos
                .entry(grupo(raiz, rel))
                .or_insert_with(|| Candidate { logical: String::new(), added: 0, removed: 0, bytes: 0 });
            e.removed += 1;
        }
    }
    for (grupo, c) in &mut grupos {
        c.logical = grupo.clone();
    }
    let mut out: Vec<Candidate> = grupos.into_values().collect();
    out.sort_by(|a, b| b.changed().cmp(&a.changed()).then_with(|| a.logical.cmp(&b.logical)));
    out
}
