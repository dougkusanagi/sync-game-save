package library

import (
	"database/sql"
	"errors"
	"strings"

	"modernc.org/sqlite"
)

var (
	ErrNotFound      = errors.New("jogo não encontrado")
	ErrEmptyName     = errors.New("o nome do jogo não pode ficar vazio")
	ErrDuplicateName = errors.New("já existe um jogo com este nome")
)

type NewGame struct {
	Name       string   `json:"name"`
	Exe        string   `json:"exe"`
	InstallDir *string  `json:"install_dir"`
	CatalogID  *int64   `json:"catalog_id"`
	SavePaths  []string `json:"save_paths"`
}
type Game struct {
	ID           int64    `json:"id"`
	Name         string   `json:"name"`
	Exe          string   `json:"exe"`
	InstallDir   *string  `json:"install_dir"`
	CatalogID    *int64   `json:"catalog_id"`
	SavePaths    []string `json:"save_paths"`
	BaseDigest   *string  `json:"base_digest"`
	LastSyncAt   *int64   `json:"last_sync_at"`
	PlaytimeSecs int64    `json:"playtime_secs"`
	LastPlayedAt *int64   `json:"last_played_at"`
}

func gameWriteError(err error) error {
	var sqliteErr *sqlite.Error
	if errors.As(err, &sqliteErr) && sqliteErr.Code() == 2067 {
		return ErrDuplicateName
	}
	return err
}
func (l *Library) AddGame(g NewGame, now int64) (int64, error) {
	name := strings.TrimSpace(g.Name)
	if name == "" {
		return 0, ErrEmptyName
	}
	tx, err := l.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	result, err := tx.Exec("INSERT INTO games(name,exe,install_dir,catalog_id,created_at) VALUES(?,?,?,?,?)", name, g.Exe, g.InstallDir, g.CatalogID, now)
	if err != nil {
		return 0, gameWriteError(err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	if _, err := tx.Exec("INSERT INTO sync_state(game_id) VALUES(?)", id); err != nil {
		return 0, err
	}
	for _, p := range g.SavePaths {
		if _, err := tx.Exec("INSERT OR IGNORE INTO save_paths(game_id,path) VALUES(?,?)", id, p); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return id, nil
}
func (l *Library) Get(id int64) (Game, error) {
	var g Game
	err := l.db.QueryRow(`SELECT g.id,g.name,g.exe,g.install_dir,g.catalog_id,s.base_digest,s.last_sync_at,g.playtime_secs,g.last_played_at
 FROM games g LEFT JOIN sync_state s ON s.game_id=g.id WHERE g.id=?`, id).Scan(&g.ID, &g.Name, &g.Exe, &g.InstallDir, &g.CatalogID, &g.BaseDigest, &g.LastSyncAt, &g.PlaytimeSecs, &g.LastPlayedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Game{}, ErrNotFound
	}
	if err != nil {
		return Game{}, err
	}
	rows, err := l.db.Query("SELECT path FROM save_paths WHERE game_id=? AND enabled=1 ORDER BY id", id)
	if err != nil {
		return Game{}, err
	}
	defer rows.Close()
	g.SavePaths = []string{}
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return Game{}, err
		}
		g.SavePaths = append(g.SavePaths, p)
	}
	if err := rows.Err(); err != nil {
		return Game{}, err
	}
	return g, nil
}
func (l *Library) List() ([]Game, error) {
	rows, err := l.db.Query("SELECT id FROM games ORDER BY name COLLATE NOCASE")
	if err != nil {
		return nil, err
	}
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	games := make([]Game, 0, len(ids))
	for _, id := range ids {
		g, err := l.Get(id)
		if err != nil {
			return nil, err
		}
		games = append(games, g)
	}
	return games, nil
}
func (l *Library) UpdateGame(id int64, g NewGame) error {
	return l.UpdateGameReset(id, g, false)
}

// UpdateGameReset altera o jogo e invalida o estado de sync na mesma transação.
func (l *Library) UpdateGameReset(id int64, g NewGame, reset bool) error {
	name := strings.TrimSpace(g.Name)
	if name == "" {
		return ErrEmptyName
	}
	tx, err := l.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.Exec("UPDATE games SET name=?,exe=?,install_dir=?,catalog_id=? WHERE id=?", name, g.Exe, g.InstallDir, g.CatalogID, id)
	if err != nil {
		return gameWriteError(err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return ErrNotFound
	}
	if _, err := tx.Exec("DELETE FROM save_paths WHERE game_id=?", id); err != nil {
		return err
	}
	for _, p := range g.SavePaths {
		if _, err := tx.Exec("INSERT OR IGNORE INTO save_paths(game_id,path) VALUES(?,?)", id, p); err != nil {
			return err
		}
	}
	if reset {
		result, err := tx.Exec("UPDATE sync_state SET base_digest=NULL,last_sync_at=NULL WHERE game_id=?", id)
		if err != nil {
			return err
		}
		count, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if count == 0 {
			return ErrNotFound
		}
	}
	return tx.Commit()
}
func (l *Library) RemoveGame(id int64) error {
	result, err := l.db.Exec("DELETE FROM games WHERE id=?", id)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return ErrNotFound
	}
	return nil
}
