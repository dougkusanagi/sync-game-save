//go:build windows

package core

import (
	"os"
	"testing"
)

func TestListaProcessoAtual(t *testing.T) {
	processes, err := ListProcesses()
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range processes {
		if p.PID == uint32(os.Getpid()) {
			if p.Started == 0 || p.Executable == "" {
				t.Fatal(p)
			}
			return
		}
	}
	t.Fatal("processo atual ausente")
}
