package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDescobertaListaPastaQueMudou(t *testing.T) {
	dados := t.TempDir()
	t.Setenv("SGS_DATA_DIR", dados)
	pasta := t.TempDir()
	// Pastas do usuário vazias: a captura não percorre o AppData real.
	for _, v := range []string{"USERPROFILE", "HOME", "APPDATA", "LOCALAPPDATA", "PUBLIC", "PROGRAMDATA", "XDG_DATA_HOME", "XDG_CONFIG_HOME"} {
		t.Setenv(v, t.TempDir())
	}
	// O "jogo" grava o save dentro da própria pasta (marcador <base>).
	t.Setenv("SGS_PLAY_HELPER", filepath.Join(pasta, "Saves", "slot.sav"))
	os.MkdirAll(filepath.Join(pasta, "Saves"), 0755)
	exe, _ := os.Executable()
	copia := filepath.Join(pasta, "jogo-teste"+filepath.Ext(exe))
	data, _ := os.ReadFile(exe)
	if err := os.WriteFile(copia, data, 0755); err != nil {
		t.Fatal(err)
	}
	app := App{}
	if _, err := app.CandidatosSave(copia); err == nil {
		t.Fatal("candidatos sem captura")
	}
	if _, err := app.DescobrirSave(filepath.Join(pasta, "nao-existe.exe")); err == nil {
		t.Fatal("executável ausente aceito")
	}
	code, err := descobrirSave(dados, copia, 100*time.Millisecond)
	if err != nil || code != 5 {
		t.Fatal(code, err)
	}
	cands, err := app.CandidatosSave(copia)
	if err != nil || len(cands) == 0 {
		t.Fatal(cands, err)
	}
	for _, c := range cands {
		if c.Logical == "<base>/Saves" {
			if c.Added != 1 || c.Resolved != filepath.Join(pasta, "Saves") {
				t.Fatal(c)
			}
			return
		}
	}
	t.Fatal("pasta de save não listada", cands)
}
