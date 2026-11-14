package core

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func TestProcessHelper(t *testing.T) {
	if os.Getenv("SGS_PROCESS_TEST_HELPER") != "1" {
		return
	}
	for i, arg := range os.Args {
		if arg == "--" && i+1 < len(os.Args) {
			ms, _ := strconv.Atoi(os.Args[i+1])
			time.Sleep(time.Duration(ms) * time.Millisecond)
			os.Exit(7)
		}
	}
	os.Exit(2)
}
func TestRunProcessEsperaEEnviaHeartbeat(t *testing.T) {
	t.Setenv("SGS_PROCESS_TEST_HELPER", "1")
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	beats := 0
	observed := false
	start := time.Now()
	code, err := RunProcess(exe, []string{"-test.run=^TestProcessHelper$", "--", "500"}, "", 200*time.Millisecond, 100*time.Millisecond, func() { beats++; observed = observed || ProcessRunning(exe) })
	if err != nil || code != 7 || beats == 0 || !observed || ProcessRunning(exe) || time.Since(start) < 700*time.Millisecond {
		t.Fatal(code, beats, err, time.Since(start))
	}
}

func TestRunProcessRecebePedidoDeParada(t *testing.T) {
	t.Setenv("SGS_PROCESS_TEST_HELPER", "1")
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	// Uma cópia em pasta temporária evita incluir outros processos da pasta do teste.
	data, err := os.ReadFile(exe)
	if err != nil {
		t.Fatal(err)
	}
	copy := filepath.Join(t.TempDir(), "jogo-teste"+filepath.Ext(exe))
	if err := os.WriteFile(copy, data, 0755); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		_, err := RunProcess(copy, []string{"-test.run=^TestProcessHelper$", "--", "1200"}, "", 100*time.Millisecond, time.Hour, nil)
		done <- err
	}()
	deadline := time.Now().Add(5 * time.Second)
	for !ProcessRunning(copy) {
		if time.Now().After(deadline) {
			t.Fatal("jogo não iniciou")
		}
		time.Sleep(10 * time.Millisecond)
	}
	found, running, err := StopProcess(copy)
	if err != nil || !found || running {
		t.Fatal(found, running, err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}
