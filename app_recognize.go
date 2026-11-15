package main

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"sync-game-save/internal/catalog"
	"sync-game-save/internal/core"
)

type Reconhecimento = catalog.Recognition

func (App) ReconhecerPasta(dir string) (Reconhecimento, error) {
	cat, err := catalog.Open(filepath.Join(pastaDados(), "catalog.db"))
	if errors.Is(err, os.ErrNotExist) {
		// Sem o catálogo baixado, só dá para saber o nome e o executável pela pasta.
		folder := filepath.Base(filepath.Clean(dir))
		name := catalog.CleanFolderName(folder)
		if name == "" {
			name = folder
		}
		return Reconhecimento{Name: name, Confidence: catalog.ConfidenceNone, Exe: catalog.GuessExe(dir), SaveCandidates: []catalog.SaveCandidate{}, Alternatives: []catalog.Hit{}}, nil
	}
	if err != nil {
		return Reconhecimento{}, err
	}
	defer cat.Close()
	return cat.Recognize(dir, core.SystemEnv(dir), runtime.GOOS == "windows")
}

func (App) ConverterCaminho(path string) string {
	if logical, ok := core.ToLogical(path, core.SystemEnv("")); ok {
		return logical
	}
	return path
}

var atualizandoCatalogo sync.Mutex

// AtualizarCatalogo baixa o manifesto, se mudou, e remonta o catálogo. Devolve o número de jogos
// (0 quando nada mudou).
func (App) AtualizarCatalogo() (int, error) {
	atualizandoCatalogo.Lock()
	defer atualizandoCatalogo.Unlock()
	dados := pastaDados()
	if err := os.MkdirAll(dados, 0755); err != nil {
		return 0, err
	}
	db := filepath.Join(dados, "catalog.db")
	changed, err := catalog.Fetch(filepath.Join(dados, "manifest.yaml"), catalog.ManifestURL)
	if err != nil {
		return 0, err
	}
	if _, statErr := os.Stat(db); !changed && statErr == nil {
		return 0, nil
	}
	stats, err := catalog.Build(filepath.Join(dados, "manifest.yaml"), db)
	return stats.Games, err
}
