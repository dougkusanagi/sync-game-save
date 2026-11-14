//go:build !windows

package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSnapshotAceitaNomesValidosNoUnix(t *testing.T) {
	dir := t.TempDir()
	save := filepath.Join(dir, "save")
	if err := os.Mkdir(save, 0755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{`slot:1`, `slot\2`} {
		if err := os.WriteFile(filepath.Join(save, name), []byte(name), 0644); err != nil {
			t.Fatal(err)
		}
	}
	cloud := filepath.Join(dir, "cloud")
	if _, err := CreateSnapshot(cloud, []Root{{Path: save}}, "1_PC", "PC", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := VerifySnapshot(filepath.Join(cloud, "snapshots", "1_PC")); err != nil {
		t.Fatal(err)
	}
}
