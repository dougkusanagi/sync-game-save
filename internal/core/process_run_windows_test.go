//go:build windows

package core

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestProcessLauncherHelper(t *testing.T) {
	if os.Getenv("SGS_PROCESS_LAUNCHER") != "1" {
		return
	}
	for i, arg := range os.Args {
		if arg == "--" && i+2 < len(os.Args) {
			cmd := exec.Command(os.Args[i+1], "-test.run=^TestProcessChildHelper$", "--", os.Args[i+2])
			if err := cmd.Start(); err != nil {
				os.Exit(2)
			}
			os.Exit(0)
		}
	}
	os.Exit(3)
}
func TestProcessChildHelper(t *testing.T) {
	if os.Getenv("SGS_PROCESS_LAUNCHER") != "1" {
		return
	}
	for i, arg := range os.Args {
		if arg == "--" && i+1 < len(os.Args) {
			time.Sleep(800 * time.Millisecond)
			if os.WriteFile(os.Args[i+1], []byte("ok"), 0644) != nil {
				os.Exit(2)
			}
			os.Exit(0)
		}
	}
	os.Exit(3)
}
func TestRunProcessAguardaDescendente(t *testing.T) {
	t.Setenv("SGS_PROCESS_LAUNCHER", "1")
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	copy := filepath.Join(dir, "launcher.exe")
	data, err := os.ReadFile(exe)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(copy, data, 0755); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(dir, "terminou")
	code, err := RunProcess(copy, []string{"-test.run=^TestProcessLauncherHelper$", "--", exe, marker}, "", 200*time.Millisecond, time.Hour, nil)
	if err != nil || code != 0 {
		t.Fatal(code, err)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatal("descendente ainda não terminou", err)
	}
}
