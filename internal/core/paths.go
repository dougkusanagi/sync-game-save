package core

import (
	"path/filepath"
	"strings"
)

// Env associa marcadores portáteis às pastas deste computador.
type Env map[string]string

func Resolve(logical string, env Env) (string, bool) {
	if !strings.HasPrefix(logical, "<") {
		return "", false
	}
	end := strings.IndexByte(logical, '>')
	if end < 0 {
		return "", false
	}
	root, ok := env[logical[1:end]]
	if !ok {
		return "", false
	}
	tail := strings.TrimLeft(logical[end+1:], `/\`)
	parts := strings.FieldsFunc(tail, func(r rune) bool { return r == '/' || r == '\\' })
	for _, p := range parts {
		if p == ".." {
			return "", false
		}
	}
	return filepath.Join(append([]string{root}, parts...)...), true
}

func ToLogical(path string, env Env) (string, bool) {
	var name, rel string
	longest := -1
	for key, root := range env {
		r, err := filepath.Rel(root, path)
		if err != nil || r == ".." || strings.HasPrefix(r, ".."+string(filepath.Separator)) || filepath.IsAbs(r) {
			continue
		}
		if len(filepath.Clean(root)) > longest {
			name, rel, longest = key, r, len(filepath.Clean(root))
		}
	}
	if longest < 0 {
		return "", false
	}
	if rel == "." {
		return "<" + name + ">", true
	}
	return "<" + name + ">/" + filepath.ToSlash(rel), true
}

// SafeName troca os caracteres que o Windows não aceita em nomes de pasta.
func SafeName(name string) string {
	s := strings.Map(func(c rune) rune {
		if strings.ContainsRune(`<>:"/\|?*`, c) || c < 0x20 || (c >= 0x7f && c < 0xa0) {
			return '_'
		}
		return c
	}, name)
	return strings.TrimRight(strings.TrimSpace(s), ".")
}
