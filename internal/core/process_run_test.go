package core

import (
	"os"
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
	start := time.Now()
	code, err := RunProcess(exe, []string{"-test.run=^TestProcessHelper$", "--", "500"}, "", 200*time.Millisecond, 100*time.Millisecond, func() { beats++ })
	if err != nil || code != 7 || beats == 0 || time.Since(start) < 700*time.Millisecond {
		t.Fatal(code, beats, err, time.Since(start))
	}
}
