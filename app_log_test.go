package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRegistrarGravaUmaLinhaPorEventoNoArquivoDoDia(t *testing.T) {
	dados := t.TempDir()
	registrar(dados, "ERRO", "jogo %d: falhou\nem duas linhas", 7)
	registrar(dados, "INFO", "jogo %d: ok", 7)
	arqs, _ := filepath.Glob(filepath.Join(pastaDeLogs(dados), "sync-*.log"))
	if len(arqs) != 1 {
		t.Fatalf("arquivos = %v", arqs)
	}
	b, _ := os.ReadFile(arqs[0])
	linhas := strings.Split(strings.TrimSpace(string(b)), "\n")
	if len(linhas) != 2 || !strings.Contains(linhas[0], "ERRO") || !strings.Contains(linhas[0], "jogo 7: falhou em duas linhas") {
		t.Fatalf("linhas = %q", linhas)
	}
}

func TestLimparLogsAntigosApagaSoLogsVelhos(t *testing.T) {
	dados := t.TempDir()
	pasta := pastaDeLogs(dados)
	_ = os.MkdirAll(pasta, 0o755)
	velho := filepath.Join(pasta, "sync-2020-01-01.log")
	novo := filepath.Join(pasta, "sync-2026-11-16.log")
	outro := filepath.Join(pasta, "notas.txt")
	for _, f := range []string{velho, novo, outro} {
		_ = os.WriteFile(f, []byte("x"), 0o644)
	}
	antigo := time.Now().AddDate(0, 0, -diasDeLog-1)
	_ = os.Chtimes(velho, antigo, antigo)
	_ = os.Chtimes(outro, antigo, antigo)
	limparLogsAntigos(dados, time.Now())
	if _, err := os.Stat(velho); err == nil {
		t.Error("log velho deveria ter sido apagado")
	}
	for _, f := range []string{novo, outro} {
		if _, err := os.Stat(f); err != nil {
			t.Errorf("%s não deveria ser apagado", f)
		}
	}
}
