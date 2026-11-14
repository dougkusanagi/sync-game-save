package library

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestAbreBancoNovoEReabreConfiguracao(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dados", "library.db")
	lib, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if version, err := lib.SchemaVersion(); err != nil || version != 3 {
		t.Fatal(version, err)
	}
	if err := lib.SetSetting("sync_dir", "D:/Nuvem"); err != nil {
		t.Fatal(err)
	}
	if err := lib.Close(); err != nil {
		t.Fatal(err)
	}
	lib, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer lib.Close()
	value, err := lib.Setting("sync_dir")
	if err != nil || value == nil || *value != "D:/Nuvem" {
		t.Fatal(value, err)
	}
	missing, err := lib.Setting("ausente")
	if err != nil || missing != nil {
		t.Fatal(missing, err)
	}
}

func TestMigraBancoAntigoSemPerderDados(t *testing.T) {
	path := filepath.Join(t.TempDir(), "library.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`CREATE TABLE games(id INTEGER PRIMARY KEY, name TEXT NOT NULL UNIQUE COLLATE NOCASE, exe TEXT NOT NULL, install_dir TEXT, catalog_id INTEGER, created_at INTEGER NOT NULL);
 CREATE TABLE save_paths(id INTEGER PRIMARY KEY, game_id INTEGER NOT NULL REFERENCES games(id) ON DELETE CASCADE, path TEXT NOT NULL, enabled INTEGER NOT NULL DEFAULT 1, UNIQUE(game_id,path));
 CREATE TABLE sync_state(game_id INTEGER PRIMARY KEY REFERENCES games(id) ON DELETE CASCADE, base_digest TEXT, last_sync_at INTEGER);
 CREATE TABLE settings(key TEXT PRIMARY KEY, value TEXT NOT NULL);
 INSERT INTO games(name,exe,created_at) VALUES('Antigo','a.exe',1);
 INSERT INTO sync_state(game_id) VALUES(1);
 PRAGMA user_version=1;`)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	lib, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer lib.Close()
	if version, err := lib.SchemaVersion(); err != nil || version != 3 {
		t.Fatal(version, err)
	}
	var name string
	var playtime int64
	if err := lib.db.QueryRow("SELECT name,playtime_secs FROM games WHERE id=1").Scan(&name, &playtime); err != nil || name != "Antigo" || playtime != 0 {
		t.Fatal(name, playtime, err)
	}
}

func TestRecusaBancoMaisNovoSemModificar(t *testing.T) {
	path := filepath.Join(t.TempDir(), "library.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("PRAGMA user_version=99"); err != nil {
		t.Fatal(err)
	}
	db.Close()
	lib, err := Open(path)
	if err == nil {
		lib.Close()
		t.Fatal("esquema futuro aceito")
	}
	db, err = sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var version int
	if err := db.QueryRow("PRAGMA user_version").Scan(&version); err != nil || version != 99 {
		t.Fatal(version, err)
	}
}

func TestSQLiteTemFTS5(t *testing.T) {
	lib, err := Open(filepath.Join(t.TempDir(), "library.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer lib.Close()
	if _, err := lib.db.Exec(`CREATE VIRTUAL TABLE teste_fts USING fts5(nome); INSERT INTO teste_fts(nome) VALUES('Rogue Legacy');`); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := lib.db.QueryRow(`SELECT count(*) FROM teste_fts WHERE teste_fts MATCH 'Rogue'`).Scan(&count); err != nil || count != 1 {
		t.Fatal(count, err)
	}
}
