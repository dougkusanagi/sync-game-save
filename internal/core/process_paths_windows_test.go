//go:build windows

package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNaoRastreiaDiretorioDoWindows(t *testing.T) {
	root := os.Getenv("SystemRoot")
	if root == "" {
		t.Skip("SystemRoot não definido")
	}
	tr := NewProcessTracker(101, filepath.Join(root, "System32"))
	tr.Refresh([]ProcessInfo{{PID: 101, Started: 1}, {PID: 102, Started: 2, Executable: filepath.Join(root, "System32", "outro.exe")}})
	if tr.Contains(102) {
		t.Fatal("processo de sistema associado ao jogo")
	}
}
