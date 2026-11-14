//go:build !windows

package core

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func systemDocuments(home string) string {
	if home == "" {
		return ""
	}
	if runtime.GOOS == "darwin" {
		return filepath.Join(home, "Documents")
	}
	config := os.Getenv("XDG_CONFIG_HOME")
	if !filepath.IsAbs(config) {
		config = filepath.Join(home, ".config")
	}
	data, err := os.ReadFile(filepath.Join(config, "user-dirs.dirs"))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		key, value, ok := strings.Cut(line, "=")
		if !ok || strings.TrimSpace(key) != "XDG_DOCUMENTS_DIR" {
			continue
		}
		value = strings.TrimSpace(value)
		if len(value) < 2 || value[0] != '"' || value[len(value)-1] != '"' {
			continue
		}
		value = value[1 : len(value)-1]
		relative := strings.HasPrefix(value, "$HOME/")
		if value == "$HOME/" || (!relative && !strings.HasPrefix(value, "/")) {
			continue
		}
		if relative {
			value = strings.TrimPrefix(value, "$HOME/")
		}
		var b strings.Builder
		for i := 0; i < len(value); i++ {
			if value[i] == '\\' && i+1 < len(value) && strings.ContainsRune("\\$`\"", rune(value[i+1])) {
				i++
			}
			b.WriteByte(value[i])
		}
		if relative {
			return filepath.Join(home, b.String())
		}
		return b.String()
	}
	return ""
}
