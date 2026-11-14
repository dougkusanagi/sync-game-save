// Package library preserva o formato SQLite da biblioteca da versão Rust.
package library

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

var migrations = []string{
	`CREATE TABLE games(id INTEGER PRIMARY KEY, name TEXT NOT NULL UNIQUE COLLATE NOCASE, exe TEXT NOT NULL, install_dir TEXT, catalog_id INTEGER, created_at INTEGER NOT NULL);
 CREATE TABLE save_paths(id INTEGER PRIMARY KEY, game_id INTEGER NOT NULL REFERENCES games(id) ON DELETE CASCADE, path TEXT NOT NULL, enabled INTEGER NOT NULL DEFAULT 1, UNIQUE(game_id,path));
 CREATE TABLE sync_state(game_id INTEGER PRIMARY KEY REFERENCES games(id) ON DELETE CASCADE, base_digest TEXT, last_sync_at INTEGER);
 CREATE TABLE settings(key TEXT PRIMARY KEY, value TEXT NOT NULL);`,
	`ALTER TABLE games ADD COLUMN playtime_secs INTEGER NOT NULL DEFAULT 0;
 ALTER TABLE games ADD COLUMN last_played_at INTEGER;`,
	`CREATE TABLE achievements_cache(game_id INTEGER PRIMARY KEY REFERENCES games(id) ON DELETE CASCADE, fetched_at INTEGER NOT NULL, json TEXT NOT NULL);`,
}

type Library struct{ db *sql.DB }

// Open abre a biblioteca sem alterar dados existentes e aplica migrações atômicas.
func Open(path string) (*Library, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	// PRAGMA foreign_keys pertence à conexão; uma conexão mantém a mesma regra em todas as operações.
	db.SetMaxOpenConns(1)
	lib := &Library{db: db}
	if err := lib.init(); err != nil {
		db.Close()
		return nil, err
	}
	return lib, nil
}
func (l *Library) Close() error { return l.db.Close() }
func (l *Library) SchemaVersion() (int, error) {
	var version int
	err := l.db.QueryRow("PRAGMA user_version").Scan(&version)
	return version, err
}
func (l *Library) init() error {
	if _, err := l.db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		return err
	}
	version, err := l.SchemaVersion()
	if err != nil {
		return err
	}
	if version > len(migrations) {
		return fmt.Errorf("o banco foi criado por uma versão mais nova do aplicativo (esquema %d)", version)
	}
	for i := version; i < len(migrations); i++ {
		tx, err := l.db.Begin()
		if err != nil {
			return err
		}
		if _, err = tx.Exec(migrations[i]); err == nil {
			_, err = tx.Exec(fmt.Sprintf("PRAGMA user_version=%d", i+1))
		}
		if err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}
func (l *Library) Setting(key string) (*string, error) {
	var value string
	err := l.db.QueryRow("SELECT value FROM settings WHERE key=?", key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &value, nil
}
func (l *Library) SetSetting(key, value string) error {
	_, err := l.db.Exec("INSERT INTO settings(key,value) VALUES(?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value", key, value)
	return err
}
