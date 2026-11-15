package main

import (
	"path/filepath"

	"sync-game-save/internal/core"
	"sync-game-save/internal/library"
)

// ResultadoPrimeiroBackup tem o mesmo formato JSON da versão Rust (campo kind).
type ResultadoPrimeiroBackup struct {
	Kind string  `json:"kind"`
	PC   *string `json:"pc,omitempty"`
}

// PrimeiroBackup faz o primeiro backup do save sem abrir o jogo. Recusa jogo aberto ou em sync,
// não restaura nada e não troca uma versão existente na pasta de sync.
func (App) PrimeiroBackup(id int64) (ResultadoPrimeiroBackup, error) {
	return primeiroBackup(pastaDados(), id)
}

func primeiroBackup(dados string, id int64) (ResultadoPrimeiroBackup, error) {
	fim, err := iniciarOperacao(id)
	if err != nil {
		return ResultadoPrimeiroBackup{Kind: "in_use"}, nil
	}
	defer fim()
	game, eng, err := engineDoId(dados, id)
	if err != nil {
		return ResultadoPrimeiroBackup{}, err
	}
	if core.ProcessRunning(game.Exe) {
		return ResultadoPrimeiroBackup{Kind: "in_use"}, nil
	}
	jdir := filepath.Join(eng.StateDir, "journal")
	if err := core.JournalBegin(jdir, game.Name, "primeiro-backup", eng.Now); err != nil {
		return ResultadoPrimeiroBackup{}, err
	}
	resultado, errBackup := eng.FirstBackup()
	if err := core.JournalEnd(jdir, game.Name); err != nil {
		return ResultadoPrimeiroBackup{}, err
	}
	if errBackup != nil {
		return ResultadoPrimeiroBackup{}, errBackup
	}
	var out ResultadoPrimeiroBackup
	switch resultado.Status {
	case core.FirstBackupCreated:
		out.Kind = "created"
	case core.FirstBackupAlreadySynced:
		out.Kind = "already_synced"
	case core.FirstBackupNothing:
		out.Kind = "nothing_to_backup"
	case core.FirstBackupNeedsDecision:
		out.Kind = "needs_decision"
	case core.FirstBackupLockedByOther:
		out.Kind, out.PC = "locked", &resultado.Device
	default:
		out.Kind = "in_use"
	}
	if resultado.Status == core.FirstBackupCreated || resultado.Status == core.FirstBackupAlreadySynced {
		digest := ""
		if idx, err := core.ScanRoots(eng.Roots); err == nil {
			digest = idx.Digest()
		}
		lib, err := library.Open(filepath.Join(dados, "library.db"))
		if err != nil {
			return ResultadoPrimeiroBackup{}, err
		}
		defer lib.Close()
		if err := lib.RecordSync(id, digest, int64(agora())); err != nil {
			return ResultadoPrimeiroBackup{}, err
		}
	}
	return out, nil
}
