//! `library.db`: biblioteca de jogos e configurações deste PC.
//! Fica fora da pasta da nuvem (ver ADR 0003). Remover um jogo daqui nunca apaga saves.

use serde::{Deserialize, Serialize};
use rusqlite::{params, Connection, OptionalExtension};
use std::path::Path;

#[derive(Debug, thiserror::Error)]
pub enum Error {
    #[error("erro no banco da biblioteca: {0}")]
    Db(#[from] rusqlite::Error),
    #[error("o banco foi criado por uma versão mais nova do aplicativo (esquema {0})")]
    TooNew(i64),
    #[error("já existe um jogo com este nome")]
    DuplicateName,
    #[error("jogo não encontrado")]
    NotFound,
    #[error("o nome do jogo não pode ficar vazio")]
    EmptyName,
}

pub type Result<T> = std::result::Result<T, Error>;

/// Cada item é uma migração; a posição + 1 é o número do esquema (`PRAGMA user_version`).
const MIGRATIONS: &[&str] = &[
    "CREATE TABLE games(
        id INTEGER PRIMARY KEY,
        name TEXT NOT NULL UNIQUE COLLATE NOCASE,
        exe TEXT NOT NULL,
        install_dir TEXT,
        catalog_id INTEGER,
        created_at INTEGER NOT NULL
     );
     CREATE TABLE save_paths(
        id INTEGER PRIMARY KEY,
        game_id INTEGER NOT NULL REFERENCES games(id) ON DELETE CASCADE,
        path TEXT NOT NULL,
        enabled INTEGER NOT NULL DEFAULT 1,
        UNIQUE(game_id, path)
     );
     CREATE TABLE sync_state(
        game_id INTEGER PRIMARY KEY REFERENCES games(id) ON DELETE CASCADE,
        base_digest TEXT,
        last_sync_at INTEGER
     );
     CREATE TABLE settings(key TEXT PRIMARY KEY, value TEXT NOT NULL);",
    "ALTER TABLE games ADD COLUMN playtime_secs INTEGER NOT NULL DEFAULT 0;
     ALTER TABLE games ADD COLUMN last_played_at INTEGER;",
    "CREATE TABLE achievements_cache(
        game_id INTEGER PRIMARY KEY REFERENCES games(id) ON DELETE CASCADE,
        fetched_at INTEGER NOT NULL,
        json TEXT NOT NULL
     );",
];

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct NewGame {
    pub name: String,
    pub exe: String,
    pub install_dir: Option<String>,
    pub catalog_id: Option<i64>,
    /// Caminhos lógicos com marcadores, ex.: `<winAppData>/Jogo/saves`.
    pub save_paths: Vec<String>,
}

#[derive(Debug, Clone, PartialEq, Eq, Serialize)]
pub struct Game {
    pub id: i64,
    pub name: String,
    pub exe: String,
    pub install_dir: Option<String>,
    pub catalog_id: Option<i64>,
    pub save_paths: Vec<String>,
    pub base_digest: Option<String>,
    pub last_sync_at: Option<i64>,
    pub playtime_secs: i64,
    pub last_played_at: Option<i64>,
}

pub struct Library(Connection);

impl Library {
    pub fn open(path: &Path) -> Result<Self> {
        if let Some(dir) = path.parent() {
            let _ = std::fs::create_dir_all(dir);
        }
        Self::init(Connection::open(path)?)
    }

    pub fn open_in_memory() -> Result<Self> {
        Self::init(Connection::open_in_memory()?)
    }

    fn init(conn: Connection) -> Result<Self> {
        conn.pragma_update(None, "foreign_keys", true)?;
        let current: i64 = conn.pragma_query_value(None, "user_version", |r| r.get(0))?;
        if current > MIGRATIONS.len() as i64 {
            return Err(Error::TooNew(current));
        }
        for (i, sql) in MIGRATIONS.iter().enumerate().skip(current as usize) {
            // Cada migração é atômica: ou aplica inteira e sobe a versão, ou nada muda.
            let tx = conn.unchecked_transaction()?;
            tx.execute_batch(sql)?;
            tx.pragma_update(None, "user_version", i as i64 + 1)?;
            tx.commit()?;
        }
        Ok(Library(conn))
    }

    pub fn schema_version(&self) -> Result<i64> {
        Ok(self.0.pragma_query_value(None, "user_version", |r| r.get(0))?)
    }

    pub fn add_game(&mut self, g: &NewGame, now: i64) -> Result<i64> {
        let name = g.name.trim();
        if name.is_empty() {
            return Err(Error::EmptyName);
        }
        let tx = self.0.transaction()?;
        let inserted = tx.execute(
            "INSERT INTO games(name,exe,install_dir,catalog_id,created_at) VALUES(?,?,?,?,?)",
            params![name, g.exe, g.install_dir, g.catalog_id, now],
        );
        match inserted {
            Err(rusqlite::Error::SqliteFailure(e, _)) if e.code == rusqlite::ErrorCode::ConstraintViolation => {
                return Err(Error::DuplicateName);
            }
            r => r?,
        };
        let id = tx.last_insert_rowid();
        tx.execute("INSERT INTO sync_state(game_id) VALUES(?)", [id])?;
        for p in &g.save_paths {
            tx.execute("INSERT OR IGNORE INTO save_paths(game_id,path) VALUES(?,?)", params![id, p])?;
        }
        tx.commit()?;
        Ok(id)
    }

    pub fn get(&self, id: i64) -> Result<Game> {
        self.0
            .query_row(
                "SELECT g.id,g.name,g.exe,g.install_dir,g.catalog_id,s.base_digest,s.last_sync_at,g.playtime_secs,g.last_played_at
                 FROM games g LEFT JOIN sync_state s ON s.game_id=g.id WHERE g.id=?",
                [id],
                |r| {
                    Ok(Game {
                        id: r.get(0)?,
                        name: r.get(1)?,
                        exe: r.get(2)?,
                        install_dir: r.get(3)?,
                        catalog_id: r.get(4)?,
                        save_paths: vec![],
                        base_digest: r.get(5)?,
                        last_sync_at: r.get(6)?,
                        playtime_secs: r.get(7)?,
                        last_played_at: r.get(8)?,
                    })
                },
            )
            .optional()?
            .map(|mut g| {
                g.save_paths = self.paths(g.id)?;
                Ok(g)
            })
            .unwrap_or(Err(Error::NotFound))
    }

    fn paths(&self, id: i64) -> Result<Vec<String>> {
        let mut q = self.0.prepare("SELECT path FROM save_paths WHERE game_id=? AND enabled=1 ORDER BY id")?;
        let rows = q.query_map([id], |r| r.get(0))?;
        Ok(rows.collect::<std::result::Result<_, _>>()?)
    }

    pub fn list(&self) -> Result<Vec<Game>> {
        let ids: Vec<i64> = {
            let mut q = self.0.prepare("SELECT id FROM games ORDER BY name COLLATE NOCASE")?;
            let rows = q.query_map([], |r| r.get(0))?;
            rows.collect::<std::result::Result<_, _>>()?
        };
        ids.into_iter().map(|id| self.get(id)).collect()
    }

    /// Edita nome, executável e caminhos de save. Os caminhos são substituídos pela lista nova.
    pub fn update_game(&mut self, id: i64, g: &NewGame) -> Result<()> {
        let name = g.name.trim();
        if name.is_empty() {
            return Err(Error::EmptyName);
        }
        let tx = self.0.transaction()?;
        let n = match tx.execute(
            "UPDATE games SET name=?,exe=?,install_dir=?,catalog_id=? WHERE id=?",
            params![name, g.exe, g.install_dir, g.catalog_id, id],
        ) {
            Err(rusqlite::Error::SqliteFailure(e, _)) if e.code == rusqlite::ErrorCode::ConstraintViolation => {
                return Err(Error::DuplicateName);
            }
            r => r?,
        };
        if n == 0 {
            return Err(Error::NotFound);
        }
        tx.execute("DELETE FROM save_paths WHERE game_id=?", [id])?;
        for p in &g.save_paths {
            tx.execute("INSERT OR IGNORE INTO save_paths(game_id,path) VALUES(?,?)", params![id, p])?;
        }
        tx.commit()?;
        Ok(())
    }

    /// Invalida a base local quando os caminhos monitorados mudam.
    pub fn reset_sync_state(&mut self, id: i64) -> Result<()> {
        if self.0.execute("UPDATE sync_state SET base_digest=NULL,last_sync_at=NULL WHERE game_id=?", [id])? == 0 {
            return Err(Error::NotFound);
        }
        Ok(())
    }

    /// Tira o jogo da biblioteca. Não toca em arquivos de save, locais ou da nuvem.
    pub fn remove_game(&mut self, id: i64) -> Result<()> {
        match self.0.execute("DELETE FROM games WHERE id=?", [id])? {
            0 => Err(Error::NotFound),
            _ => Ok(()),
        }
    }

    pub fn record_sync(&mut self, id: i64, digest: &str, now: i64) -> Result<()> {
        self.0.execute(
            "UPDATE sync_state SET base_digest=?, last_sync_at=? WHERE game_id=?",
            params![digest, now, id],
        )?;
        Ok(())
    }

    /// Soma uma sessão ao tempo de jogo. Sessões de 0 segundo não contam.
    pub fn add_playtime(&mut self, id: i64, secs: i64, now: i64) -> Result<()> {
        if secs <= 0 {
            return Ok(());
        }
        match self.0.execute(
            "UPDATE games SET playtime_secs = playtime_secs + ?, last_played_at = ? WHERE id=?",
            params![secs, now, id],
        )? {
            0 => Err(Error::NotFound),
            _ => Ok(()),
        }
    }

    /// Guarda a última resposta de conquistas, para funcionar sem internet.
    pub fn save_achievements(&mut self, id: i64, json: &str, now: i64) -> Result<()> {
        self.0.execute(
            "INSERT INTO achievements_cache(game_id,fetched_at,json) VALUES(?,?,?)
             ON CONFLICT(game_id) DO UPDATE SET fetched_at=excluded.fetched_at, json=excluded.json",
            params![id, now, json],
        )?;
        Ok(())
    }

    pub fn achievements_cache(&self, id: i64) -> Result<Option<(String, i64)>> {
        Ok(self
            .0
            .query_row("SELECT json, fetched_at FROM achievements_cache WHERE game_id=?", [id], |r| Ok((r.get(0)?, r.get(1)?)))
            .optional()?)
    }

    pub fn setting(&self, key: &str) -> Result<Option<String>> {
        Ok(self
            .0
            .query_row("SELECT value FROM settings WHERE key=?", [key], |r| r.get(0))
            .optional()?)
    }

    pub fn set_setting(&mut self, key: &str, value: &str) -> Result<()> {
        self.0.execute(
            "INSERT INTO settings(key,value) VALUES(?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value",
            params![key, value],
        )?;
        Ok(())
    }
}
