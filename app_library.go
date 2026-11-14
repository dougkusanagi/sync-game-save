package main

import (
	"path/filepath"
	"time"

	"sync-game-save/internal/library"
)

func (App) ListarJogos() ([]library.Game, error) {
	lib, err := library.Open(filepath.Join(pastaDados(), "library.db"))
	if err != nil {
		return nil, err
	}
	defer lib.Close()
	return lib.List()
}

func (App) AdicionarJogo(jogo library.NewGame) (int64, error) {
	lib, err := library.Open(filepath.Join(pastaDados(), "library.db"))
	if err != nil {
		return 0, err
	}
	defer lib.Close()
	return lib.AddGame(jogo, time.Now().Unix())
}
