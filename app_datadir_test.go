package main

import (
	"path/filepath"
	"testing"
)

func TestPastaDeDadosIgualAVersaoTauri(t *testing.T) {
	casos := []struct{ sistema, xdg, home, config, esperado string }{
		{"windows", "", `C:\Users\ana`, filepath.Join("C:", "Users", "ana", "AppData", "Roaming"), filepath.Join("C:", "Users", "ana", "AppData", "Roaming", "com.syncgamesave.app")},
		{"linux", "", "/home/ana", "/home/ana/.config", filepath.Join("/home/ana", ".local", "share", "com.syncgamesave.app")},
		{"linux", "/dados/xdg", "/home/ana", "/home/ana/.config", filepath.Join("/dados/xdg", "com.syncgamesave.app")},
		{"darwin", "", "/Users/ana", "/Users/ana/Library/Application Support", filepath.Join("/Users/ana/Library/Application Support", "com.syncgamesave.app")},
	}
	for _, c := range casos {
		if got := pastaDadosDe(c.sistema, c.xdg, c.home, c.config); got != c.esperado {
			t.Fatal(c.sistema, got, c.esperado)
		}
	}
}

func TestSGSDataDirTemPrioridade(t *testing.T) {
	t.Setenv("SGS_DATA_DIR", "/tmp/teste-dados")
	if pastaDados() != "/tmp/teste-dados" {
		t.Fatal(pastaDados())
	}
}
