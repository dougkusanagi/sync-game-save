package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"sync-game-save/internal/core"
)

type Reconhecimento struct {
	Name           string  `json:"name"`
	CatalogID      *int64  `json:"catalog_id"`
	Confidence     string  `json:"confidence"`
	Exe            *string `json:"exe"`
	SaveCandidates []any   `json:"save_candidates"`
	Alternatives   []any   `json:"alternatives"`
}

func nomePasta(name string) string {
	var out strings.Builder
	depth := 0
	for _, c := range name {
		switch c {
		case '[', '(':
			depth++
		case ']', ')':
			if depth > 0 {
				depth--
			}
		default:
			if depth == 0 {
				if c == '_' {
					c = ' '
				}
				out.WriteRune(c)
			}
		}
	}
	var words []string
	for _, word := range strings.Fields(out.String()) {
		lower := strings.ToLower(word)
		version := len(lower) > 1 && lower[0] == 'v' && strings.Trim(lower[1:], "0123456789.") == ""
		if version || lower == "gog" || lower == "repack" || lower == "portable" {
			continue
		}
		for i := 0; i+2 < len(word); i++ {
			if word[i] == '.' && word[i+1] == 'v' && word[i+2] >= '0' && word[i+2] <= '9' {
				word = word[:i]
				break
			}
		}
		words = append(words, strings.ReplaceAll(word, ".", " "))
	}
	return strings.Join(strings.Fields(strings.Join(words, " ")), " ")
}

func executavelPasta(dir string) *string {
	bad := []string{"unins", "setup", "install", "crash", "redist", "vcredist", "dxsetup", "dotnet", "launcherhelper", "updater"}
	var best string
	var size int64 = -1
	var walk func(string, int)
	walk = func(path string, depth int) {
		entries, err := os.ReadDir(path)
		if err != nil {
			return
		}
		for _, entry := range entries {
			file := filepath.Join(path, entry.Name())
			info, err := os.Stat(file)
			if err != nil {
				continue
			}
			if info.IsDir() {
				if depth < 2 {
					walk(file, depth+1)
				}
				continue
			}
			if !strings.EqualFold(filepath.Ext(file), ".exe") {
				continue
			}
			name := strings.ToLower(entry.Name())
			skip := false
			for _, part := range bad {
				if strings.Contains(name, part) {
					skip = true
					break
				}
			}
			if !skip && info.Size() > size {
				best = file
				size = info.Size()
			}
		}
	}
	walk(dir, 0)
	if best == "" {
		return nil
	}
	return &best
}

func (App) ReconhecerPasta(dir string) (Reconhecimento, error) {
	if _, err := os.Stat(filepath.Join(pastaDados(), "catalog.db")); err == nil {
		return Reconhecimento{}, errors.New("Ainda não consigo consultar o catálogo nesta versão.")
	} else if !errors.Is(err, os.ErrNotExist) {
		return Reconhecimento{}, err
	}
	folder := filepath.Base(filepath.Clean(dir))
	name := nomePasta(folder)
	if name == "" {
		name = folder
	}
	return Reconhecimento{Name: name, Confidence: "None", Exe: executavelPasta(dir), SaveCandidates: []any{}, Alternatives: []any{}}, nil
}
func (App) ConverterCaminho(path string) string {
	if logical, ok := core.ToLogical(path, core.SystemEnv("")); ok {
		return logical
	}
	return path
}
