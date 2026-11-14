//go:build windows

package core

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

func TestForceStopProcessRastreadoForaDoJob(t *testing.T) {
	t.Setenv("SGS_PROCESS_TEST_HELPER", "1")
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(exe)
	if err != nil {
		t.Fatal(err)
	}
	start := func() (*exec.Cmd, uint64) {
		t.Helper()
		name := filepath.Join(t.TempDir(), "jogo.exe")
		if err := os.WriteFile(name, data, 0755); err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command(name, "-test.run=^TestProcessHelper$", "--", "12000")
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = cmd.Process.Kill(); _ = cmd.Wait() })
		h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(cmd.Process.Pid))
		if err != nil {
			t.Fatal(err)
		}
		started, err := processStarted(h)
		windows.CloseHandle(h)
		if err != nil || started == 0 {
			t.Fatal(started, err)
		}
		return cmd, started
	}
	target, started := start()
	other, otherStarted := start()
	job, err := newProcessJob()
	if err != nil {
		t.Fatal(err)
	}
	defer job.close()
	tracker := NewProcessTracker(uint32(target.Process.Pid), "")
	// O outro PID simula uma identidade antiga; deve continuar aberto.
	tracker.pids[uint32(other.Process.Pid)] = struct{}{}
	tracker.starts[uint32(target.Process.Pid)] = started
	tracker.starts[uint32(other.Process.Pid)] = otherStarted - 1
	processes := []ProcessInfo{{PID: uint32(target.Process.Pid), Started: started}, {PID: uint32(other.Process.Pid), Started: otherStarted - 1}}
	if err := job.forceStop(tracker, processes); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- target.Wait() }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("processo rastreado continuou aberto")
	}
	h, err := windows.OpenProcess(windows.SYNCHRONIZE, false, uint32(other.Process.Pid))
	if err != nil {
		t.Fatal("processo com identidade divergente não está aberto:", err)
	}
	state, err := windows.WaitForSingleObject(h, 0)
	windows.CloseHandle(h)
	if err != nil || state != uint32(windows.WAIT_TIMEOUT) {
		t.Fatal("processo com identidade divergente foi encerrado", state, err)
	}
}
