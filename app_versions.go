package main

import (
	"errors"
	"fmt"
	"path/filepath"
	"strconv"

	"sync-game-save/internal/core"
	"sync-game-save/internal/library"
)

// engineDoId carrega o jogo e monta o contexto de sincronização dele.
func engineDoId(dados string, id int64) (library.Game, core.Engine, error) {
	lib, err := library.Open(filepath.Join(dados, "library.db"))
	if err != nil {
		return library.Game{}, core.Engine{}, err
	}
	defer lib.Close()
	game, err := lib.Get(id)
	if err != nil {
		return game, core.Engine{}, err
	}
	syncDir, err := lib.Setting("sync_dir")
	if err != nil {
		return game, core.Engine{}, err
	}
	device, err := lib.Setting("device")
	if err != nil {
		return game, core.Engine{}, err
	}
	nome := nomePCPadrao()
	if device != nil {
		nome = *device
	}
	eng, err := montarEngine(dados, game, syncDir, nome)
	return game, eng, err
}

func (App) Versoes(id int64) ([]string, error) {
	_, eng, err := engineDoId(pastaDados(), id)
	if err != nil {
		return nil, err
	}
	return core.ListSnapshots(eng.GameDir)
}

// RestaurarVersao restaura uma versão. O save atual vai antes para a pasta de backups,
// cujo caminho é devolvido.
func (App) RestaurarVersao(id int64, versao string) (string, error) {
	fim, err := iniciarOperacao(id)
	if err != nil {
		return "", err
	}
	defer fim()
	dados := pastaDados()
	game, eng, err := engineDoId(dados, id)
	if err != nil {
		return "", err
	}
	if core.ProcessRunning(game.Exe) {
		return "", errors.New("Feche o jogo e aguarde a sincronização antes de restaurar uma versão.")
	}
	status, lock, err := core.LockState(eng.GameDir, eng.Device, eng.Now)
	if err != nil {
		return "", err
	}
	if status == core.LockHeldByOther {
		return "", fmt.Errorf("O jogo parece estar aberto em %s. Feche-o antes de restaurar.", lock.Device)
	}
	lista, err := core.ListSnapshots(eng.GameDir)
	if err != nil {
		return "", err
	}
	achou := false
	for _, s := range lista {
		achou = achou || s == versao
	}
	if !achou {
		return "", errors.New("Esta versão não foi encontrada. Atualize a lista e tente novamente.")
	}
	snap := filepath.Join(eng.GameDir, "snapshots", versao)
	backup := filepath.Join(eng.StateDir, "backups", strconv.FormatUint(eng.Now, 10)+"_restaurar")
	jdir := filepath.Join(eng.StateDir, "journal")
	if err := core.JournalBegin(jdir, game.Name, "restaurar", eng.Now); err != nil {
		return "", err
	}
	if err := core.RestoreSnapshot(snap, eng.Roots, backup); err != nil {
		return "", fmt.Errorf("Não consegui restaurar a versão: %v. Confira as cópias em %s antes de tentar novamente.", err, backup)
	}
	if err := core.JournalEnd(jdir, game.Name); err != nil {
		return "", err
	}
	return backup, nil
}
