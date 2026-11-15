package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"sync-game-save/internal/core"
)

// O binário de teste serve de "jogo": com SGS_CLI_HELPER=<arquivo>, grava nele e sai com o código
// de SGS_CLI_CODE.
func TestMain(m *testing.M) {
	if f := os.Getenv("SGS_CLI_HELPER"); f != "" {
		code, _ := strconv.Atoi(os.Getenv("SGS_CLI_CODE"))
		if err := os.WriteFile(f, []byte("novo"), 0644); err != nil {
			os.Exit(99)
		}
		os.Exit(code)
	}
	os.Exit(m.Run())
}

func sgs(args []string, stdin string) (int, error, string) {
	var out bytes.Buffer
	code, err := Exec(args, &out, strings.NewReader(stdin))
	return code, err, out.String()
}

func base(root, pc, now string) []string {
	r := func(s string) string { return filepath.Join(root, s) }
	return []string{
		"--sync-dir", r("nuvem"), "--game", "Jogo: Teste", "--save-dir", r(pc + "/save"),
		"--state-dir", r(pc + "/state"), "--device", pc, "--now", now,
	}
}

func call(cmd string, extra, b []string, stdin string) (int, error, string) {
	return sgs(append(append([]string{cmd}, b...), extra...), stdin)
}

// jogoDeTeste copia o binário de teste para uma pasta própria, para o rastreador não incluir
// outros processos da pasta do teste.
func jogoDeTeste(t *testing.T) string {
	t.Helper()
	exe, _ := os.Executable()
	data, err := os.ReadFile(exe)
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(t.TempDir(), "jogo-teste"+filepath.Ext(exe))
	if err := os.WriteFile(p, data, 0755); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestRunSalvaMesmoComCodigoDeErroEOutroPCRecebe(t *testing.T) {
	root := t.TempDir()
	a := base(root, "A", "1000")
	os.MkdirAll(filepath.Join(root, "A", "save"), 0755)
	os.MkdirAll(filepath.Join(root, "nuvem"), 0755)
	t.Setenv("SGS_CLI_HELPER", filepath.Join(root, "A", "save", "slot.sav"))
	t.Setenv("SGS_CLI_CODE", "7")
	code, err, out := call("run", []string{"--quiet-secs", "1", "--", jogoDeTeste(t)}, a, "")
	if err != nil || code != 7 || !strings.Contains(out, "Save enviado para a nuvem") {
		t.Fatal(code, err, out)
	}
	if code, err, out = call("list", nil, a, ""); err != nil || code != 0 || !strings.Contains(out, "1000_A") {
		t.Fatal(code, err, out)
	}
	// Outro PC restaura pelo comando restore.
	b := base(root, "B", "2000")
	os.MkdirAll(filepath.Join(root, "B", "save"), 0755)
	if code, err, out = call("restore", nil, b, ""); err != nil || code != 0 {
		t.Fatal(code, err, out)
	}
	if got, _ := os.ReadFile(filepath.Join(root, "B", "save", "slot.sav")); !strings.Contains(string(got), "novo") {
		t.Fatal(string(got))
	}
	if _, err, out = call("status", []string{"--json"}, b, ""); err != nil || !strings.Contains(out, `"head":"1000_A"`) {
		t.Fatal(err, out)
	}
}

func TestRunRecusaQuandoOutroPCEstaJogando(t *testing.T) {
	root := t.TempDir()
	a := base(root, "A", "1000")
	os.MkdirAll(filepath.Join(root, "A", "save"), 0755)
	os.WriteFile(filepath.Join(root, "A", "save", "s.sav"), []byte("a"), 0644)
	os.MkdirAll(filepath.Join(root, "B", "save"), 0755)
	os.MkdirAll(filepath.Join(root, "nuvem"), 0755)
	t.Setenv("SGS_CLI_HELPER", filepath.Join(root, "A", "fora.txt"))
	t.Setenv("SGS_CLI_CODE", "0")
	jogo := jogoDeTeste(t)
	if code, err, out := call("run", []string{"--quiet-secs", "1", "--", jogo}, a, ""); err != nil || code != 0 {
		t.Fatal(code, err, out)
	}
	// Simula A ainda jogando: trava de A fresca.
	if err := core.AcquireLock(filepath.Join(root, "nuvem", "SyncGameSave", "games", "Jogo_ Teste"), "A", 1500); err != nil {
		t.Fatal(err)
	}
	code, err, out := call("run", []string{"--quiet-secs", "1", "--", jogo}, base(root, "B", "1600"), "")
	if err != nil || code != 3 || !strings.Contains(out, "--force") {
		t.Fatal(code, err, out)
	}
}

func TestConflitoCancelaOuEscolhe(t *testing.T) {
	root := t.TempDir()
	a, b := base(root, "A", "1000"), base(root, "B", "3000")
	for _, d := range []string{"A/save", "B/save", "nuvem"} {
		os.MkdirAll(filepath.Join(root, d), 0755)
	}
	os.WriteFile(filepath.Join(root, "A", "save", "s.sav"), []byte("a1"), 0644)
	t.Setenv("SGS_CLI_HELPER", filepath.Join(root, "A", "fora.txt"))
	jogo := jogoDeTeste(t)
	args := []string{"--quiet-secs", "1", "--", jogo}
	if code, err, out := call("run", args, a, ""); err != nil || code != 0 {
		t.Fatal(code, err, out)
	}
	// B ganha o save da nuvem, depois os dois lados mudam.
	if code, err, out := call("run", args, b, ""); err != nil || code != 0 {
		t.Fatal(code, err, out)
	}
	os.WriteFile(filepath.Join(root, "B", "save", "s.sav"), []byte("b2"), 0644)
	os.WriteFile(filepath.Join(root, "A", "save", "s.sav"), []byte("a2"), 0644)
	if code, err, out := call("run", args, base(root, "A", "4000"), ""); err != nil || code != 0 {
		t.Fatal(code, err, out)
	}
	// B entra em conflito; sem resposta (stdin vazio), cancela e nada muda.
	b2 := base(root, "B", "5000")
	code, err, out := call("run", args, b2, "")
	if err != nil || code != 2 || !strings.Contains(out, "Cancelado") {
		t.Fatal(code, err, out)
	}
	if got, _ := os.ReadFile(filepath.Join(root, "B", "save", "s.sav")); string(got) != "b2" {
		t.Fatal("cancelar alterou o save:", string(got))
	}
	// Escolher a nuvem pelo teclado restaura o save da nuvem.
	code, err, out = call("run", args, b2, "n\n")
	if err != nil || code != 0 {
		t.Fatal(code, err, out)
	}
	if got, _ := os.ReadFile(filepath.Join(root, "B", "save", "s.sav")); string(got) != "a2" {
		t.Fatal("save da nuvem não restaurado:", string(got))
	}
}

func TestBackupEStatus(t *testing.T) {
	root := t.TempDir()
	a := base(root, "A", "1000")
	os.MkdirAll(filepath.Join(root, "A", "save"), 0755)
	os.WriteFile(filepath.Join(root, "A", "save", "s.sav"), []byte("x"), 0644)
	os.MkdirAll(filepath.Join(root, "nuvem"), 0755)
	if _, err, out := call("status", nil, a, ""); err != nil || !strings.Contains(out, "Snapshot atual: nenhum") || !strings.Contains(out, "Trava: livre") {
		t.Fatal(err, out)
	}
	if _, err, out := call("backup", nil, a, ""); err != nil || !strings.Contains(out, "Snapshot criado.") {
		t.Fatal(err, out)
	}
	if _, err, out := call("backup", nil, base(root, "A", "1100"), ""); err != nil || !strings.Contains(out, "Nada mudou") {
		t.Fatal(err, out)
	}
	if _, err, out := call("list", []string{"--json"}, a, ""); err != nil || !strings.Contains(out, `["1000_A"]`) {
		t.Fatal(err, out)
	}
}

func TestErrosClaros(t *testing.T) {
	if _, err, _ := sgs([]string{"run"}, ""); err == nil || !strings.Contains(err.Error(), "--sync-dir") {
		t.Fatal(err)
	}
	if _, err, _ := sgs([]string{"voar", "--sync-dir", "x", "--game", "g", "--save-dir", "s", "--state-dir", t.TempDir()}, ""); err == nil || !strings.Contains(err.Error(), "desconhecido") {
		t.Fatal(err)
	}
	if code, err, out := sgs([]string{"help"}, ""); err != nil || code != 0 || !strings.Contains(out, "Uso: sgs") {
		t.Fatal(code, err)
	}
	if _, err, _ := sgs([]string{"run", "--sync-dir"}, ""); err == nil || !strings.Contains(err.Error(), "falta o valor") {
		t.Fatal(err)
	}
}
