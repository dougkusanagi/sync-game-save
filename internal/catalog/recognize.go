package catalog

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"sync-game-save/internal/core"
)

type Confidence string

const (
	// ConfidenceHigh: o nome da pasta é igual a uma pasta de instalação do catálogo.
	ConfidenceHigh Confidence = "High"
	// ConfidenceLow: achado só pela busca; o usuário deve conferir.
	ConfidenceLow Confidence = "Low"
	// ConfidenceNone: nada no catálogo.
	ConfidenceNone Confidence = "None"
)

type SaveCandidate struct {
	Logical  string `json:"logical"`
	Resolved string `json:"resolved"`
	Exists   bool   `json:"exists"`
	Files    int    `json:"files"`
	Selected bool   `json:"selected"`
}

type Recognition struct {
	Name           string          `json:"name"`
	CatalogID      *int64          `json:"catalog_id"`
	Confidence     Confidence      `json:"confidence"`
	Exe            *string         `json:"exe"`
	SaveCandidates []SaveCandidate `json:"save_candidates"`
	Alternatives   []Hit           `json:"alternatives"`
}

// CleanFolderName tira o que não faz parte do nome: [GOG], (x64), v1.2.3, "_" e ".".
func CleanFolderName(name string) string {
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
		// "Game.v1.2.3": corta na versão colada ao nome.
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

var badExe = []string{"unins", "setup", "install", "crash", "redist", "vcredist", "dxsetup", "dotnet", "launcherhelper", "updater"}

func isBadExe(file string) bool {
	l := strings.ToLower(file)
	for _, b := range badExe {
		if strings.Contains(l, b) {
			return true
		}
	}
	return false
}

func isExe(path string) bool { return strings.EqualFold(filepath.Ext(path), ".exe") }

// GuessExe devolve o maior .exe da pasta (até 2 níveis), ignorando instaladores e ajudantes.
func GuessExe(dir string) *string {
	var best string
	var size int64 = -1
	var walk func(string, int)
	walk = func(path string, depth int) {
		entries, err := os.ReadDir(path)
		if err != nil {
			return
		}
		for _, e := range entries {
			file := filepath.Join(path, e.Name())
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
			if isExe(file) && !isBadExe(e.Name()) && info.Size() > size {
				best, size = file, info.Size()
			}
		}
	}
	walk(dir, 0)
	if best == "" {
		return nil
	}
	return &best
}

func countFiles(path string) int {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	if !info.IsDir() {
		return 1
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return 0
	}
	total := 0
	for _, e := range entries {
		total += countFiles(filepath.Join(path, e.Name()))
	}
	return total
}

func hasTag(tags, tag string) bool {
	for _, t := range strings.Split(tags, ",") {
		if t == tag {
			return true
		}
	}
	return false
}

func (c *Catalog) saveCandidates(id int64, env core.Env, windows bool) ([]SaveCandidate, error) {
	osName := "linux"
	if windows {
		osName = "windows"
	}
	paths, err := c.SavePaths(id)
	if err != nil {
		return nil, err
	}
	out := []SaveCandidate{}
	seen := map[string]bool{}
	for _, sp := range paths {
		if sp.OS != nil && *sp.OS != osName {
			continue
		}
		save := hasTag(sp.Tags, "save")
		if hasTag(sp.Tags, "config") && !save {
			continue
		}
		// Marcadores que variam por usuário da loja ou curingas não resolvem para um caminho único.
		if strings.Contains(sp.Path, "*") || strings.Contains(sp.Path, "<storeUserId>") {
			continue
		}
		resolved, ok := core.Resolve(sp.Path, env)
		if !ok || seen[sp.Path] {
			continue
		}
		seen[sp.Path] = true
		_, statErr := os.Stat(resolved)
		exists := statErr == nil
		// Pastas de emuladores ou cracks (regra sem tag) que não existem só poluem a lista.
		if !exists && sp.Tags == "" {
			continue
		}
		files := 0
		if exists {
			files = countFiles(resolved)
		}
		out = append(out, SaveCandidate{Logical: sp.Path, Resolved: resolved, Exists: exists, Files: files, Selected: save && exists && files > 0})
	}
	// Os que existem primeiro.
	rank := func(c SaveCandidate) int {
		n := 0
		if !c.Selected {
			n += 2
		}
		if !c.Exists {
			n++
		}
		return n
	}
	sort.SliceStable(out, func(i, j int) bool { return rank(out[i]) < rank(out[j]) })
	return out, nil
}

// Recognize reconhece a pasta de um jogo. windows escolhe as regras do Windows; senão, as do Linux.
func (c *Catalog) Recognize(dir string, env core.Env, windows bool) (Recognition, error) {
	folder := filepath.Base(filepath.Clean(dir))
	cleaned := CleanFolderName(folder)
	hits, err := c.ByInstallDir(folder)
	if err != nil {
		return Recognition{}, err
	}
	confidence := ConfidenceHigh
	if len(hits) == 0 && cleaned != "" {
		if hits, err = c.ByInstallDir(cleaned); err != nil {
			return Recognition{}, err
		}
	}
	if len(hits) == 0 {
		if hits, err = c.Search(cleaned, 5); err != nil {
			return Recognition{}, err
		}
		confidence = ConfidenceLow
		if len(hits) == 0 {
			confidence = ConfidenceNone
		}
	}
	// Entre vários jogos com a mesma pasta, prefere o que tem save existente no disco.
	type scored struct {
		score int
		hit   Hit
		cands []SaveCandidate
	}
	var all []scored
	for _, h := range hits {
		cands, err := c.saveCandidates(h.ID, env, windows)
		if err != nil {
			return Recognition{}, err
		}
		score := 0
		for _, cand := range cands {
			if cand.Selected {
				score++
			}
		}
		all = append(all, scored{score, h, cands})
	}
	sort.SliceStable(all, func(i, j int) bool { return all[i].score > all[j].score })

	rec := Recognition{Name: cleaned, Confidence: confidence, SaveCandidates: []SaveCandidate{}, Alternatives: []Hit{}}
	if rec.Name == "" {
		rec.Name = folder
	}
	if len(all) > 0 {
		id := all[0].hit.ID
		rec.Name, rec.CatalogID, rec.SaveCandidates = all[0].hit.Name, &id, all[0].cands
		for _, s := range all[1:] {
			rec.Alternatives = append(rec.Alternatives, s.hit)
		}
	}
	// O catálogo às vezes nomeia o jogo só pelo id da Steam; nesse caso a pasta dá o nome melhor.
	if cleaned != "" && strings.Trim(rec.Name, "0123456789") == "" {
		rec.Name = cleaned
	}
	// Executável: o do launch do catálogo, se existir; senão, o maior .exe.
	if rec.CatalogID != nil {
		baseEnv := core.Env{}
		for k, v := range env {
			baseEnv[k] = v
		}
		baseEnv["base"] = dir
		launches, err := c.LaunchPaths(*rec.CatalogID)
		if err != nil {
			return Recognition{}, err
		}
		for _, l := range launches {
			p, ok := core.Resolve(l, baseEnv)
			if !ok {
				continue
			}
			if info, err := os.Stat(p); err == nil && info.Mode().IsRegular() && (isExe(p) || !windows) {
				rec.Exe = &p
				break
			}
		}
	}
	if rec.Exe == nil {
		rec.Exe = GuessExe(dir)
	}
	return rec, nil
}

type ScanEntry struct {
	Path        string      `json:"path"`
	Recognition Recognition `json:"recognition"`
}

type ScanResult struct {
	Recognized        []ScanEntry `json:"recognized"`
	NeedsConfirmation []ScanEntry `json:"needs_confirmation"`
	Unknown           []ScanEntry `json:"unknown"`
}

func hasExe(dir string, depth int) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		p := filepath.Join(dir, e.Name())
		info, err := os.Stat(p)
		if err != nil {
			continue
		}
		if info.IsDir() {
			if depth < 2 && hasExe(p, depth+1) {
				return true
			}
		} else if isExe(p) && !isBadExe(e.Name()) {
			return true
		}
	}
	return false
}

// ScanFolder varre as subpastas de root até depth níveis (1 = filhas diretas). Cada pasta com
// executável vira um jogo; não desce dentro de uma pasta já tratada como jogo.
func (c *Catalog) ScanFolder(root string, depth int, env core.Env, windows bool) (ScanResult, error) {
	out := ScanResult{Recognized: []ScanEntry{}, NeedsConfirmation: []ScanEntry{}, Unknown: []ScanEntry{}}
	var walk func(string, int) error
	walk = func(dir string, left int) error {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil
		}
		var dirs []string
		for _, e := range entries {
			p := filepath.Join(dir, e.Name())
			if info, err := os.Stat(p); err == nil && info.IsDir() {
				dirs = append(dirs, p)
			}
		}
		sort.Strings(dirs)
		for _, d := range dirs {
			if hasExe(d, 0) {
				r, err := c.Recognize(d, env, windows)
				if err != nil {
					return err
				}
				entry := ScanEntry{Path: d, Recognition: r}
				switch r.Confidence {
				case ConfidenceHigh:
					out.Recognized = append(out.Recognized, entry)
				case ConfidenceLow:
					out.NeedsConfirmation = append(out.NeedsConfirmation, entry)
				default:
					out.Unknown = append(out.Unknown, entry)
				}
			} else if left > 1 {
				if err := walk(d, left-1); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if depth < 1 {
		depth = 1
	}
	return out, walk(root, depth)
}
