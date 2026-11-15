// Package catalog converte o manifesto do Ludusavi (YAML, MIT) em SQLite com FTS5.
// O formato do catalog.db é o mesmo da versão Rust.
package catalog

import (
	"database/sql"
	"fmt"
	"os"
	"sort"
	"strings"
	"unicode"

	"gopkg.in/yaml.v3"
	_ "modernc.org/sqlite"
)

type manifestGame struct {
	Files      map[string]fileRule `yaml:"files"`
	InstallDir map[string]any      `yaml:"installDir"`
	Launch     map[string]any      `yaml:"launch"`
	Steam      struct {
		ID *uint64 `yaml:"id"`
	} `yaml:"steam"`
}

type fileRule struct {
	Tags []string `yaml:"tags"`
	When []struct {
		OS    *string `yaml:"os"`
		Store *string `yaml:"store"`
	} `yaml:"when"`
}

type Stats struct{ Games, Files int }

// Normalize remove acentos comuns, passa para minúsculas e tira o que não é letra ou dígito.
func Normalize(s string) string {
	var out strings.Builder
	for _, c := range s {
		for _, l := range []rune(strings.ToLower(string(c))) {
			switch l {
			case 'á', 'à', 'â', 'ã', 'ä':
				l = 'a'
			case 'é', 'è', 'ê', 'ë':
				l = 'e'
			case 'í', 'ì', 'î', 'ï':
				l = 'i'
			case 'ó', 'ò', 'ô', 'õ', 'ö':
				l = 'o'
			case 'ú', 'ù', 'û', 'ü':
				l = 'u'
			case 'ç':
				l = 'c'
			}
			if unicode.IsLetter(l) || unicode.IsDigit(l) || unicode.IsNumber(l) {
				out.WriteRune(l)
			}
		}
	}
	return out.String()
}

const schema = `
CREATE TABLE games(id INTEGER PRIMARY KEY, name TEXT NOT NULL, name_norm TEXT NOT NULL, steam_id INTEGER);
CREATE TABLE install_dirs(game_id INTEGER NOT NULL, dir TEXT NOT NULL, dir_norm TEXT NOT NULL);
CREATE TABLE save_paths(game_id INTEGER NOT NULL, path TEXT NOT NULL, tags TEXT NOT NULL, os TEXT, store TEXT);
CREATE TABLE launch(game_id INTEGER NOT NULL, path TEXT NOT NULL);
CREATE VIRTUAL TABLE search USING fts5(name, dirs, tokenize='trigram');
`

const indexes = `
CREATE INDEX games_name_norm ON games(name_norm);
CREATE INDEX install_dirs_norm ON install_dirs(dir_norm);
CREATE INDEX save_paths_game ON save_paths(game_id);
CREATE INDEX launch_game ON launch(game_id);
`

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// Build monta o catálogo num arquivo temporário e o troca pelo definitivo no fim.
func Build(manifest, dbPath string) (Stats, error) {
	raw, err := os.ReadFile(manifest)
	if err != nil {
		return Stats{}, err
	}
	// O manifesto real tem caracteres de controle (ex.: DEL) que o YAML recusa.
	text := strings.Map(func(c rune) rune {
		if unicode.IsControl(c) && c != '\n' && c != '\r' && c != '\t' {
			return ' '
		}
		return c
	}, string(raw))
	var games map[string]manifestGame
	if err := yaml.Unmarshal([]byte(text), &games); err != nil {
		return Stats{}, fmt.Errorf("manifesto inválido: %w", err)
	}
	tmp := dbPath + ".tmp"
	_ = os.Remove(tmp)
	db, err := sql.Open("sqlite", tmp)
	if err != nil {
		return Stats{}, err
	}
	db.SetMaxOpenConns(1)
	closed := false
	defer func() {
		if !closed {
			db.Close()
			_ = os.Remove(tmp)
		}
	}()
	if _, err := db.Exec(schema); err != nil {
		return Stats{}, err
	}
	tx, err := db.Begin()
	if err != nil {
		return Stats{}, err
	}
	defer tx.Rollback()
	var stats Stats
	for _, name := range sortedKeys(games) {
		game := games[name]
		var steam any
		if game.Steam.ID != nil {
			steam = int64(*game.Steam.ID)
		}
		res, err := tx.Exec("INSERT INTO games(name,name_norm,steam_id) VALUES(?,?,?)", name, Normalize(name), steam)
		if err != nil {
			return Stats{}, err
		}
		id, _ := res.LastInsertId()
		dirs := sortedKeys(game.InstallDir)
		for _, dir := range dirs {
			if _, err := tx.Exec("INSERT INTO install_dirs VALUES(?,?,?)", id, dir, Normalize(dir)); err != nil {
				return Stats{}, err
			}
		}
		for _, path := range sortedKeys(game.Files) {
			rule := game.Files[path]
			tags := strings.Join(rule.Tags, ",")
			if len(rule.When) == 0 {
				if _, err := tx.Exec("INSERT INTO save_paths VALUES(?,?,?,?,?)", id, path, tags, nil, nil); err != nil {
					return Stats{}, err
				}
			}
			for _, w := range rule.When {
				if _, err := tx.Exec("INSERT INTO save_paths VALUES(?,?,?,?,?)", id, path, tags, w.OS, w.Store); err != nil {
					return Stats{}, err
				}
			}
			stats.Files++
		}
		for _, path := range sortedKeys(game.Launch) {
			if _, err := tx.Exec("INSERT INTO launch VALUES(?,?)", id, path); err != nil {
				return Stats{}, err
			}
		}
		if _, err := tx.Exec("INSERT INTO search(rowid,name,dirs) VALUES(?,?,?)", id, name, strings.Join(dirs, " ")); err != nil {
			return Stats{}, err
		}
		stats.Games++
	}
	if err := tx.Commit(); err != nil {
		return Stats{}, err
	}
	if _, err := db.Exec(indexes); err != nil {
		return Stats{}, err
	}
	closed = true
	if err := db.Close(); err != nil {
		_ = os.Remove(tmp)
		return Stats{}, err
	}
	if err := os.Rename(tmp, dbPath); err != nil {
		_ = os.Remove(tmp)
		return Stats{}, err
	}
	return stats, nil
}

type Hit struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type SavePath struct {
	Path, Tags string
	OS, Store  *string
}

type Catalog struct{ db *sql.DB }

// Open abre o catálogo só para leitura.
func Open(dbPath string) (*Catalog, error) {
	if _, err := os.Stat(dbPath); err != nil {
		return nil, err
	}
	uri := "file:" + strings.ReplaceAll(strings.ReplaceAll(dbPath, "\\", "/"), "?", "%3f") + "?mode=ro"
	db, err := sql.Open("sqlite", uri)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	return &Catalog{db}, nil
}

func (c *Catalog) Close() error { return c.db.Close() }

func (c *Catalog) hits(query string, args ...any) ([]Hit, error) {
	rows, err := c.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Hit{}
	for rows.Next() {
		var h Hit
		if err := rows.Scan(&h.ID, &h.Name); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

// ByInstallDir lista os jogos cuja pasta de instalação tem este nome, sem acentos nem maiúsculas.
func (c *Catalog) ByInstallDir(dir string) ([]Hit, error) {
	return c.hits(`SELECT DISTINCT g.id, g.name FROM games g JOIN install_dirs d ON d.game_id=g.id
		WHERE d.dir_norm=? ORDER BY g.name`, Normalize(dir))
}

// Search busca por nome ou pasta. Com menos de 3 letras usa prefixo, porque o trigram exige 3.
func (c *Catalog) Search(text string, limit int) ([]Hit, error) {
	norm := Normalize(text)
	if norm == "" {
		return []Hit{}, nil
	}
	if len([]rune(norm)) < 3 {
		return c.hits("SELECT id, name FROM games WHERE name_norm LIKE ?||'%' ORDER BY length(name), name LIMIT ?", norm, limit)
	}
	// Aspas duplas fazem o FTS5 tratar o texto como frase literal.
	phrase := `"` + strings.ReplaceAll(strings.TrimSpace(text), `"`, `""`) + `"`
	return c.hits(`SELECT g.id, g.name FROM search s JOIN games g ON g.id=s.rowid
		WHERE search MATCH ? ORDER BY rank LIMIT ?`, phrase, limit)
}

func (c *Catalog) SavePaths(gameID int64) ([]SavePath, error) {
	rows, err := c.db.Query("SELECT path, tags, os, store FROM save_paths WHERE game_id=? ORDER BY path", gameID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SavePath
	for rows.Next() {
		var p SavePath
		if err := rows.Scan(&p.Path, &p.Tags, &p.OS, &p.Store); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// SteamID devolve o id do jogo na Steam, se o catálogo tiver.
func (c *Catalog) SteamID(gameID int64) (*uint64, error) {
	var id sql.NullInt64
	err := c.db.QueryRow("SELECT steam_id FROM games WHERE id=?", gameID).Scan(&id)
	if err == sql.ErrNoRows || !id.Valid {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	v := uint64(id.Int64)
	return &v, nil
}

func (c *Catalog) LaunchPaths(gameID int64) ([]string, error) {
	rows, err := c.db.Query("SELECT path FROM launch WHERE game_id=? ORDER BY path", gameID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
