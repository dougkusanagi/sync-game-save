//go:build windows

package core

import (
	"os/exec"
	"syscall"
	"testing"
	"time"
)

func suspendedPing(t *testing.T) *exec.Cmd {
	t.Helper()
	cmd := exec.Command(`C:\Windows\System32\PING.EXE`, "-n", "30", "127.0.0.1")
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 4, HideWindow: true}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	return cmd
}
func TestJobAssociaRetomaETermina(t *testing.T) {
	job, err := newProcessJob()
	if err != nil {
		t.Fatal(err)
	}
	defer job.close()
	cmd := suspendedPing(t)
	if err := job.attachAndResume(cmd); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cmd.Process.Kill(); _ = cmd.Wait() }()
	active, err := job.active()
	if err != nil || !active {
		t.Fatal(active, err)
	}
	if err := job.terminate(); err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() { _ = cmd.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("processo não terminou")
	}
	active, err = job.active()
	if err != nil || active {
		t.Fatal(active, err)
	}
}
func TestJobFalhaNaoDeixaProcessoSuspenso(t *testing.T) {
	cmd := suspendedPing(t)
	job := processJob{}
	if err := job.attachAndResume(cmd); err == nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		t.Fatal("associação inválida aceita")
	}
	if cmd.ProcessState == nil || !cmd.ProcessState.Exited() {
		t.Fatal("processo suspenso não foi encerrado")
	}
}
