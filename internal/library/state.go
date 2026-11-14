package library

import (
	"database/sql"
	"errors"
)

func (l *Library) ResetSyncState(id int64) error {
	result, err := l.db.Exec("UPDATE sync_state SET base_digest=NULL,last_sync_at=NULL WHERE game_id=?", id)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
func (l *Library) RecordSync(id int64, digest string, now int64) error {
	_, err := l.db.Exec("UPDATE sync_state SET base_digest=?,last_sync_at=? WHERE game_id=?", digest, now, id)
	return err
}
func (l *Library) AddPlaytime(id, secs, now int64) error {
	if secs <= 0 {
		return nil
	}
	result, err := l.db.Exec("UPDATE games SET playtime_secs=playtime_secs+?,last_played_at=? WHERE id=?", secs, now, id)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

type AchievementsCache struct {
	JSON      string
	FetchedAt int64
}

func (l *Library) SaveAchievements(id int64, data string, now int64) error {
	_, err := l.db.Exec(`INSERT INTO achievements_cache(game_id,fetched_at,json) VALUES(?,?,?) ON CONFLICT(game_id) DO UPDATE SET fetched_at=excluded.fetched_at,json=excluded.json`, id, now, data)
	return err
}
func (l *Library) AchievementsCache(id int64) (*AchievementsCache, error) {
	var cache AchievementsCache
	err := l.db.QueryRow("SELECT json,fetched_at FROM achievements_cache WHERE game_id=?", id).Scan(&cache.JSON, &cache.FetchedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &cache, nil
}
