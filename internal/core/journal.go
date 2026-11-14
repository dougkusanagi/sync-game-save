package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type JournalEntry struct {
	Op      string `json:"op"`
	Game    string `json:"game"`
	Started uint64 `json:"started"`
}

func JournalBegin(dir, game, op string, now uint64) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(JournalEntry{op, game, now}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, game+".json"), data, 0644)
}
func JournalEnd(dir, game string) error {
	err := os.Remove(filepath.Join(dir, game+".json"))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
func JournalPending(dir string) ([]JournalEntry, error) {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return []JournalEntry{}, nil
	}
	if err != nil {
		return nil, err
	}
	result := []JournalEntry{}
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		var item struct {
			Op      *string `json:"op"`
			Game    *string `json:"game"`
			Started *uint64 `json:"started"`
		}
		if err := json.Unmarshal(data, &item); err != nil {
			return nil, err
		}
		if item.Op == nil || item.Game == nil || item.Started == nil {
			return nil, fmt.Errorf("diário inválido: %s", entry.Name())
		}
		result = append(result, JournalEntry{*item.Op, *item.Game, *item.Started})
	}
	return result, nil
}
