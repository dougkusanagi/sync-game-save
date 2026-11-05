//! Catálogo de jogos: converte o manifesto do Ludusavi (YAML, MIT) em SQLite com FTS5.

use rusqlite::{params, Connection};
use serde::Deserialize;
use std::collections::BTreeMap;
use std::path::Path;

pub const MANIFEST_URL: &str =
    "https://raw.githubusercontent.com/mtkennerly/ludusavi-manifest/master/data/manifest.yaml";

#[derive(Debug, thiserror::Error)]
pub enum Error {
    #[error("erro de arquivo: {0}")]
    Io(#[from] std::io::Error),
    #[error("manifesto inválido: {0}")]
    Yaml(#[from] serde_yaml::Error),
    #[error("erro no banco: {0}")]
    Db(#[from] rusqlite::Error),
    #[error("falha ao baixar o manifesto: {0}")]
    Net(String),
}

pub type Result<T> = std::result::Result<T, Error>;

#[derive(Deserialize, Default)]
struct Game {
    #[serde(default)]
    files: BTreeMap<String, FileRule>,
    #[serde(default, rename = "installDir")]
    install_dir: BTreeMap<String, serde::de::IgnoredAny>,
    #[serde(default)]
    launch: BTreeMap<String, Vec<serde::de::IgnoredAny>>,
    #[serde(default)]
    steam: Steam,
}

#[derive(Deserialize, Default)]
struct Steam {
    id: Option<u64>,
}

#[derive(Deserialize, Default)]
struct FileRule {
    #[serde(default)]
    tags: Vec<String>,
    #[serde(default)]
    when: Vec<When>,
}

#[derive(Deserialize, Default)]
struct When {
    os: Option<String>,
    store: Option<String>,
}

#[derive(Debug, Default, PartialEq, Eq)]
pub struct Stats {
    pub games: usize,
    pub files: usize,
}

/// Remove acentos comuns, passa para minúsculas e tira tudo que não é letra/dígito.
pub fn normalize(s: &str) -> String {
    s.chars()
        .flat_map(|c| c.to_lowercase())
        .map(|c| match c {
            'á' | 'à' | 'â' | 'ã' | 'ä' => 'a',
            'é' | 'è' | 'ê' | 'ë' => 'e',
            'í' | 'ì' | 'î' | 'ï' => 'i',
            'ó' | 'ò' | 'ô' | 'õ' | 'ö' => 'o',
            'ú' | 'ù' | 'û' | 'ü' => 'u',
            'ç' => 'c',
            c => c,
        })
        .filter(|c| c.is_alphanumeric())
        .collect()
}

const SCHEMA: &str = "
CREATE TABLE games(id INTEGER PRIMARY KEY, name TEXT NOT NULL, name_norm TEXT NOT NULL, steam_id INTEGER);
CREATE TABLE install_dirs(game_id INTEGER NOT NULL, dir TEXT NOT NULL, dir_norm TEXT NOT NULL);
CREATE TABLE save_paths(game_id INTEGER NOT NULL, path TEXT NOT NULL, tags TEXT NOT NULL, os TEXT, store TEXT);
CREATE TABLE launch(game_id INTEGER NOT NULL, path TEXT NOT NULL);
CREATE VIRTUAL TABLE search USING fts5(name, dirs, tokenize='trigram');
";

const INDEXES: &str = "
CREATE INDEX games_name_norm ON games(name_norm);
CREATE INDEX install_dirs_norm ON install_dirs(dir_norm);
CREATE INDEX save_paths_game ON save_paths(game_id);
CREATE INDEX launch_game ON launch(game_id);
";

/// Monta o `catalog.db` num arquivo temporário e troca pelo definitivo no fim.
pub fn build(manifest: &Path, db: &Path) -> Result<Stats> {
    // O manifesto real tem caracteres de controle (ex.: DEL) que o parser YAML recusa.
    let text: String = std::fs::read_to_string(manifest)?
        .chars()
        .map(|c| if c.is_control() && !matches!(c, '\n' | '\r' | '\t') { ' ' } else { c })
        .collect();
    let games: BTreeMap<String, Game> = serde_yaml::from_str(&text)?;
    let tmp = db.with_extension("db.tmp");
    let _ = std::fs::remove_file(&tmp);
    let mut conn = Connection::open(&tmp)?;
    conn.execute_batch(SCHEMA)?;
    let mut stats = Stats::default();
    let tx = conn.transaction()?;
    {
        let mut g = tx.prepare("INSERT INTO games(name,name_norm,steam_id) VALUES(?,?,?)")?;
        let mut d = tx.prepare("INSERT INTO install_dirs VALUES(?,?,?)")?;
        let mut f = tx.prepare("INSERT INTO save_paths VALUES(?,?,?,?,?)")?;
        let mut l = tx.prepare("INSERT INTO launch VALUES(?,?)")?;
        let mut s = tx.prepare("INSERT INTO search(rowid,name,dirs) VALUES(?,?,?)")?;
        for (name, game) in &games {
            g.execute(params![name, normalize(name), game.steam.id.map(|i| i as i64)])?;
            let id = tx.last_insert_rowid();
            for dir in game.install_dir.keys() {
                d.execute(params![id, dir, normalize(dir)])?;
            }
            for (path, rule) in &game.files {
                if rule.when.is_empty() {
                    f.execute(params![id, path, rule.tags.join(","), None::<String>, None::<String>])?;
                }
                for w in &rule.when {
                    f.execute(params![id, path, rule.tags.join(","), w.os, w.store])?;
                }
                stats.files += 1;
            }
            for path in game.launch.keys() {
                l.execute(params![id, path])?;
            }
            let dirs = game.install_dir.keys().cloned().collect::<Vec<_>>().join(" ");
            s.execute(params![id, name, dirs])?;
            stats.games += 1;
        }
    }
    tx.commit()?;
    conn.execute_batch(INDEXES)?;
    drop(conn);
    std::fs::rename(&tmp, db)?;
    Ok(stats)
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Hit {
    pub id: i64,
    pub name: String,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct SavePath {
    pub path: String,
    pub tags: String,
    pub os: Option<String>,
    pub store: Option<String>,
}

pub struct Catalog(Connection);

impl Catalog {
    pub fn open(db: &Path) -> Result<Self> {
        Ok(Catalog(Connection::open_with_flags(
            db,
            rusqlite::OpenFlags::SQLITE_OPEN_READ_ONLY,
        )?))
    }

    /// Jogos cuja pasta de instalação tem este nome (sem acentos e sem diferenciar maiúsculas).
    pub fn by_install_dir(&self, dir: &str) -> Result<Vec<Hit>> {
        let mut q = self.0.prepare(
            "SELECT DISTINCT g.id, g.name FROM games g JOIN install_dirs d ON d.game_id=g.id
             WHERE d.dir_norm=? ORDER BY g.name",
        )?;
        let rows = q.query_map([normalize(dir)], |r| Ok(Hit { id: r.get(0)?, name: r.get(1)? }))?;
        Ok(rows.collect::<std::result::Result<_, _>>()?)
    }

    /// Busca por nome ou pasta. Com menos de 3 letras usa prefixo, porque o trigram exige 3.
    pub fn search(&self, text: &str, limit: usize) -> Result<Vec<Hit>> {
        let norm = normalize(text);
        if norm.is_empty() {
            return Ok(vec![]);
        }
        let map = |r: &rusqlite::Row| Ok(Hit { id: r.get(0)?, name: r.get(1)? });
        let hits = if norm.chars().count() < 3 {
            let mut q = self.0.prepare(
                "SELECT id, name FROM games WHERE name_norm LIKE ?||'%' ORDER BY length(name), name LIMIT ?",
            )?;
            let v: Vec<Hit> = q.query_map(params![norm, limit as i64], map)?.collect::<std::result::Result<_, _>>()?;
            v
        } else {
            // Aspas duplas fazem o FTS5 tratar o texto como frase literal.
            let phrase = format!("\"{}\"", text.trim().replace('"', "\"\""));
            let mut q = self.0.prepare(
                "SELECT g.id, g.name FROM search s JOIN games g ON g.id=s.rowid
                 WHERE search MATCH ? ORDER BY rank LIMIT ?",
            )?;
            let v: Vec<Hit> = q.query_map(params![phrase, limit as i64], map)?.collect::<std::result::Result<_, _>>()?;
            v
        };
        Ok(hits)
    }

    pub fn save_paths(&self, game_id: i64) -> Result<Vec<SavePath>> {
        let mut q = self
            .0
            .prepare("SELECT path, tags, os, store FROM save_paths WHERE game_id=? ORDER BY path")?;
        let rows = q.query_map([game_id], |r| {
            Ok(SavePath { path: r.get(0)?, tags: r.get(1)?, os: r.get(2)?, store: r.get(3)? })
        })?;
        Ok(rows.collect::<std::result::Result<_, _>>()?)
    }

    pub fn launch_paths(&self, game_id: i64) -> Result<Vec<String>> {
        let mut q = self.0.prepare("SELECT path FROM launch WHERE game_id=? ORDER BY path")?;
        let rows = q.query_map([game_id], |r| r.get(0))?;
        Ok(rows.collect::<std::result::Result<_, _>>()?)
    }
}

/// Baixa o manifesto só se mudou (ETag). Devolve `true` se baixou algo novo.
/// O arquivo novo entra por troca atômica; em erro, o antigo continua valendo.
pub fn fetch(manifest: &Path) -> Result<bool> {
    let etag_file = manifest.with_extension("etag");
    let mut req = ureq::get(MANIFEST_URL);
    if manifest.exists() {
        if let Ok(tag) = std::fs::read_to_string(&etag_file) {
            req = req.header("If-None-Match", tag.trim());
        }
    }
    let mut resp = req.config().http_status_as_error(false).build().call().map_err(|e| Error::Net(e.to_string()))?;
    match resp.status().as_u16() {
        304 => Ok(false),
        200 => {
            let tag = resp.headers().get("etag").and_then(|v| v.to_str().ok()).map(str::to_owned);
            let body = resp.body_mut().with_config().limit(200 * 1024 * 1024).read_to_vec().map_err(|e| Error::Net(e.to_string()))?;
            let tmp = manifest.with_extension("yaml.tmp");
            std::fs::write(&tmp, body)?;
            std::fs::rename(&tmp, manifest)?;
            if let Some(t) = tag {
                std::fs::write(etag_file, t)?;
            }
            Ok(true)
        }
        s => Err(Error::Net(format!("resposta HTTP {s}"))),
    }
}
