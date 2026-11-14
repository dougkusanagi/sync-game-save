package core

import (
	"path/filepath"
	"testing"
)

func TestTrackerDescendentesEProcessoRelancado(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "jogo")
	tr := NewProcessTracker(10, dir)
	items := []ProcessInfo{{PID: 10, Started: 1}, {PID: 11, Parent: 10, Started: 2}, {PID: 12, Parent: 11, Started: 3}}
	if !tr.Refresh(items) || !tr.Contains(12) {
		t.Fatal("descendentes não reconhecidos")
	}
	items = []ProcessInfo{{PID: 20, Started: 4, Executable: filepath.Join(dir, "jogo.exe")}}
	if !tr.Refresh(items) || !tr.Contains(20) {
		t.Fatal("jogo relançado não reconhecido")
	}
	if tr.Refresh(nil) {
		t.Fatal("processos encerrados continuam ativos")
	}
}
func TestTrackerIgnoraPIDReutilizado(t *testing.T) {
	tr := NewProcessTracker(10, "")
	if !tr.Refresh([]ProcessInfo{{PID: 10, Started: 1}}) {
		t.Fatal("processo ausente")
	}
	if tr.Refresh([]ProcessInfo{{PID: 10, Started: 2}}) || tr.Contains(10) {
		t.Fatal("PID reutilizado foi associado")
	}
}
