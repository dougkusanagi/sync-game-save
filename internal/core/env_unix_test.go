//go:build !windows && !darwin

package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDocumentosXDGRedirecionado(t *testing.T) {
	home := t.TempDir()
	config := filepath.Join(home, "config")
	if err := os.Mkdir(config, 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CONFIG_HOME", config)
	if err := os.WriteFile(filepath.Join(config, "user-dirs.dirs"), []byte("XDG_DOCUMENTS_DIR=\"$HOME/Meus Documentos\"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if got := systemDocuments(home); got != filepath.Join(home, "Meus Documentos") {
		t.Fatal(got)
	}
}
