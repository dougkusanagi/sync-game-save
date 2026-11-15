package catalog

import (
	"os"
	"path/filepath"
	"testing"
)

// internal/testdata/rust/catalog.db foi gerado pela versão Rust (exemplo gerar_fixtures).
func TestLeCatalogoGeradoPeloRust(t *testing.T) {
	dados, err := os.ReadFile(filepath.Join("..", "testdata", "rust", "catalog.db"))
	if err != nil {
		t.Fatal(err)
	}
	caminho := filepath.Join(t.TempDir(), "catalog.db")
	os.WriteFile(caminho, dados, 0644)
	c, err := Open(caminho)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	hits, err := c.ByInstallDir("AGGELOS")
	if err != nil || len(hits) != 1 || hits[0].Name != "Aggelos" {
		t.Fatal(hits, err)
	}
	if h, _ := c.ByInstallDir("half-life 2"); len(h) != 1 || h[0].Name != "Half-Life 2: Épisode One" {
		t.Fatal(h)
	}
	if h, err := c.Search("agge", 5); err != nil || len(h) == 0 || h[0].Name != "Aggelos" {
		t.Fatal("busca por trigram", h, err)
	}
	if h, err := c.Search("Ag", 5); err != nil || len(h) == 0 {
		t.Fatal("prefixo curto", h, err)
	}
	id := hits[0].ID
	paths, err := c.SavePaths(id)
	if err != nil || len(paths) != 2 {
		t.Fatal(paths, err)
	}
	if l, _ := c.LaunchPaths(id); len(l) != 1 || l[0] != "<base>/LAUNCHER_x64.exe" {
		t.Fatal(l)
	}
	if s, _ := c.SteamID(id); s == nil || *s != 123 {
		t.Fatal(s)
	}
}
