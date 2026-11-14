//go:build linux

package core

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
	"time"
)

func TestForceStopLinuxIdentidadeDoProcesso(t *testing.T) {
	t.Setenv("SGS_PROCESS_TEST_HELPER", "1")
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	start := func() (*exec.Cmd, uint64) {
		t.Helper()
		cmd := exec.Command(exe, "-test.run=^TestProcessHelper$", "--", "15000")
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = cmd.Process.Kill(); _ = cmd.Wait() })
		data, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(cmd.Process.Pid), "stat"))
		if err != nil {
			t.Fatal(err)
		}
		_, started, err := parseProcessStat(data)
		if err != nil || started == 0 {
			t.Fatal(started, err)
		}
		return cmd, started
	}
	target, started := start()
	other, otherStarted := start()
	tracker := NewProcessTracker(uint32(target.Process.Pid), "")
	tracker.pids[uint32(other.Process.Pid)] = struct{}{}
	tracker.starts[uint32(target.Process.Pid)] = started
	tracker.starts[uint32(other.Process.Pid)] = otherStarted + 1
	processes := []ProcessInfo{{PID: uint32(target.Process.Pid), Started: started}, {PID: uint32(other.Process.Pid), Started: otherStarted + 1}}
	if err := (&processJob{}).forceStop(tracker, processes); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- target.Wait() }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("processo rastreado continuou aberto")
	}
	if err := other.Process.Signal(os.Signal(syscall.Signal(0))); err != nil {
		t.Fatal("processo com identidade divergente não permaneceu aberto:", err)
	}
}
