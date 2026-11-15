package main

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"

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
