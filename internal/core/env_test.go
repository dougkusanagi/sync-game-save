package core

import (
	"path/filepath"
	"testing"
)

func TestAmbienteDoSistema(t *testing.T) {
	base := filepath.Join(t.TempDir(), "Jogo")
	env := SystemEnv(base)
	if got, ok := Resolve("<base>/save", env); !ok || got != filepath.Join(base, "save") {
		t.Fatal(got, ok)
	}
	if env["home"] == "" {
		t.Fatal("home ausente")
	}
	if env["winDocuments"] == "" {
		t.Fatal("Documentos ausente")
	}
}
