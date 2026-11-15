package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"sync"
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
	operacoesBiblioteca.Lock()
	defer operacoesBiblioteca.Unlock()
	lib, err := library.Open(filepath.Join(pastaDados(), "library.db"))
	if err != nil {
		return 0, err
	}
	defer lib.Close()
	return lib.AddGame(jogo, time.Now().Unix())
}

var operacoesBiblioteca sync.Mutex

func (App) EditarJogo(id int64, jogo library.NewGame) error {
	operacoesBiblioteca.Lock()
	defer operacoesBiblioteca.Unlock()
	lib, err := abrirBiblioteca()
	if err != nil {
		return err
	}
	defer lib.Close()
	anterior, err := lib.Get(id)
	if err != nil {
		return err
	}
	changed := !slices.Equal(anterior.SavePaths, jogo.SavePaths)
	if !changed {
		return lib.UpdateGame(id, jogo)
	}
	// Renomear a base antes de alterar o banco evita usá-la após uma interrupção.
	base := filepath.Join(pastaDados(), "state", strconv.FormatInt(id, 10), "base.json")
	var backup string
	if _, err := os.Stat(base); err == nil {
		temp, err := os.CreateTemp(filepath.Dir(base), ".base-anterior-*")
		if err != nil {
			return fmt.Errorf("Não consegui atualizar o estado de sincronização: %w", err)
		}
		backup = temp.Name()
		if err := temp.Close(); err != nil {
			os.Remove(backup)
			return err
		}
		if err := os.Remove(backup); err != nil {
			return err
		}
		if err := os.Rename(base, backup); err != nil {
			return fmt.Errorf("Não consegui atualizar o estado de sincronização: %w", err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("Não consegui verificar o estado de sincronização: %w", err)
	}
	if err := lib.UpdateGameReset(id, jogo, true); err != nil {
		if backup != "" {
			if restoreErr := os.Rename(backup, base); restoreErr != nil {
				return errors.Join(err, fmt.Errorf("Não consegui recuperar o estado anterior em %s: %w", backup, restoreErr))
			}
		}
		return err
	}
	if backup != "" {
		_ = os.Remove(backup)
	}
	return nil
}

func (App) RemoverJogo(id int64) error {
	operacoesBiblioteca.Lock()
	defer operacoesBiblioteca.Unlock()
	lib, err := abrirBiblioteca()
	if err != nil {
		return err
	}
	defer lib.Close()
	if err := lib.RemoveGame(id); err != nil {
		return err
	}
	_ = lib.SetSetting(fmt.Sprintf("steam_appid:%d", id), "")
	for _, suffix := range []string{".jpg", "-hero.jpg", "-logo.png", "-cover.jpg"} {
		_ = os.Remove(filepath.Join(pastaDados(), "art", strconv.FormatInt(id, 10)+suffix))
	}
	return nil
}
