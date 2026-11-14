package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanSeguePastaSimbolicaESinalizaCiclo(t *testing.T) {
	dir := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "slot"), []byte("save"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(dir, "link")); err != nil {
		t.Skipf("não foi possível criar link simbólico: %v", err)
	}
	idx, err := Scan(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := idx.Files["link/slot"]; !ok {
		t.Fatal(idx)
	}
	if err := os.Symlink(dir, filepath.Join(outside, "ciclo")); err != nil {
		t.Fatal(err)
	}
	if _, err := Scan(dir); err == nil {
		t.Fatal("ciclo aceito")
	}
}
