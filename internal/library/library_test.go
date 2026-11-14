package library

import (
	"database/sql"
	"errors"
	"path/filepath"
	"sync"
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

func TestJogosAdicionaEditaListaERemove(t *testing.T) {
	lib, err := Open(filepath.Join(t.TempDir(), "library.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer lib.Close()
	game := NewGame{Name: "  Árvore & Ação  ", Exe: "jogo.exe", SavePaths: []string{"<base>/save", "<home>/save"}}
	id, err := lib.AddGame(game, 10)
	if err != nil {
		t.Fatal(err)
	}
	got, err := lib.Get(id)
	if err != nil || got.Name != "Árvore & Ação" || len(got.SavePaths) != 2 || got.BaseDigest != nil {
		t.Fatal(got, err)
	}
	other := game
	other.Name = "Outro"
	other.SavePaths = []string{"<base>/novo", "<base>/novo"}
	if err := lib.UpdateGame(id, other); err != nil {
		t.Fatal(err)
	}
	got, err = lib.Get(id)
	if err != nil || got.Name != "Outro" || len(got.SavePaths) != 1 || got.SavePaths[0] != "<base>/novo" {
		t.Fatal(got, err)
	}
	games, err := lib.List()
	if err != nil || len(games) != 1 || games[0].ID != id {
		t.Fatal(games, err)
	}
	if err := lib.RemoveGame(id); err != nil {
		t.Fatal(err)
	}
	if _, err := lib.Get(id); !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}
}

func TestJogosNomesInvalidosEDuplicadosNaoAlteramDados(t *testing.T) {
	lib, err := Open(filepath.Join(t.TempDir(), "library.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer lib.Close()
	first, err := lib.AddGame(NewGame{Name: "Jogo", Exe: "a.exe"}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := lib.AddGame(NewGame{Name: "jogo", Exe: "b.exe"}, 2); !errors.Is(err, ErrDuplicateName) {
		t.Fatal(err)
	}
	if _, err := lib.AddGame(NewGame{Name: "   ", Exe: "b.exe"}, 2); !errors.Is(err, ErrEmptyName) {
		t.Fatal(err)
	}
	second, err := lib.AddGame(NewGame{Name: "Outro", Exe: "b.exe"}, 3)
	if err != nil {
		t.Fatal(err)
	}
	if err := lib.UpdateGame(second, NewGame{Name: "JOGO", Exe: "x.exe"}); !errors.Is(err, ErrDuplicateName) {
		t.Fatal(err)
	}
	game, err := lib.Get(second)
	if err != nil || game.Name != "Outro" || game.Exe != "b.exe" {
		t.Fatal(game, err)
	}
	if err := lib.RemoveGame(first); err != nil {
		t.Fatal(err)
	}
	if err := lib.RemoveGame(first); !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}
}

func TestTempoSyncECacheDaBiblioteca(t *testing.T) {
	lib, err := Open(filepath.Join(t.TempDir(), "library.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer lib.Close()
	id, err := lib.AddGame(NewGame{Name: "Tempo", Exe: "jogo.exe"}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := lib.RecordSync(id, "abc", 20); err != nil {
		t.Fatal(err)
	}
	if err := lib.AddPlaytime(id, 600, 100); err != nil {
		t.Fatal(err)
	}
	if err := lib.AddPlaytime(id, 0, 200); err != nil {
		t.Fatal(err)
	}
	if err := lib.AddPlaytime(id, 30, 300); err != nil {
		t.Fatal(err)
	}
	game, err := lib.Get(id)
	if err != nil || game.BaseDigest == nil || *game.BaseDigest != "abc" || game.PlaytimeSecs != 630 || game.LastPlayedAt == nil || *game.LastPlayedAt != 300 {
		t.Fatal(game, err)
	}
	if err := lib.ResetSyncState(id); err != nil {
		t.Fatal(err)
	}
	game, err = lib.Get(id)
	if err != nil || game.BaseDigest != nil || game.LastSyncAt != nil {
		t.Fatal(game, err)
	}
	if err := lib.SaveAchievements(id, `{"a":1}`, 100); err != nil {
		t.Fatal(err)
	}
	if err := lib.SaveAchievements(id, `{"a":2}`, 200); err != nil {
		t.Fatal(err)
	}
	cache, err := lib.AchievementsCache(id)
	if err != nil || cache == nil || cache.JSON != `{"a":2}` || cache.FetchedAt != 200 {
		t.Fatal(cache, err)
	}
	if err := lib.RemoveGame(id); err != nil {
		t.Fatal(err)
	}
	cache, err = lib.AchievementsCache(id)
	if err != nil || cache != nil {
		t.Fatal(cache, err)
	}
	if err := lib.AddPlaytime(id, 5, 1); !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}
}

func TestAberturasConcorrentesDeBancoNovo(t *testing.T) {
	path := filepath.Join(t.TempDir(), "library.db")
	var wg sync.WaitGroup
	errorsOut := make(chan error, 12)
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			lib, err := Open(path)
			if err == nil {
				err = lib.Close()
			}
			errorsOut <- err
		}()
	}
	wg.Wait()
	close(errorsOut)
	for err := range errorsOut {
		if err != nil {
			t.Fatal(err)
		}
	}
}
