package core

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// RunProcess abre o jogo e espera o processo, seus descendentes e o intervalo quieto.
// tick renova a trava enquanto o jogo estiver aberto ou no intervalo de espera.
func RunProcess(prog string, args []string, cwd string, quiet, every time.Duration, tick func()) (int, error) {
	cmd := exec.Command(prog, args...)
	dir := filepath.Dir(prog)
	if dir == "." {
		dir = ""
	}
	if dir != "" {
		if info, err := os.Stat(dir); err != nil || !info.IsDir() {
			dir = ""
		}
	}
	if cwd != "" {
		cmd.Dir = cwd
	} else {
		cmd.Dir = dir
	}
	group, err := prepareProcess(cmd)
	if err != nil {
		return 0, err
	}
	defer group.close()
	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("não consegui abrir %s: %w", prog, err)
	}
	if err := group.attach(cmd); err != nil {
		return 0, fmt.Errorf("não foi possível acompanhar o jogo: %w", err)
	}
	control := &stopControl{}
	defer registerProcess(prog, control)()
	tracker := NewProcessTracker(uint32(cmd.Process.Pid), dir)
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	var finished bool
	var exitErr error
	var gone time.Time
	lastBeat := time.Now()
	for {
		now := time.Now()
		if tick != nil && now.Sub(lastBeat) >= every {
			tick()
			lastBeat = now
		}
		if !finished {
			select {
			case exitErr = <-done:
				finished = true
			default:
			}
		}
		processes, err := ListProcesses()
		if err != nil {
			return 0, fmt.Errorf("não foi possível verificar os processos do jogo: %w", err)
		}
		tracked := tracker.Refresh(processes)
		active, err := group.active()
		if err != nil {
			return 0, fmt.Errorf("não foi possível verificar o grupo do jogo: %w", err)
		}
		if !finished || tracked || active {
			gone = time.Time{}
		} else if gone.IsZero() {
			gone = now
		} else if now.Sub(gone) >= quiet {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if exitErr != nil {
		if _, ok := exitErr.(*exec.ExitError); !ok {
			return 0, exitErr
		}
	}
	if cmd.ProcessState == nil {
		return 0, fmt.Errorf("não foi possível obter o resultado do jogo")
	}
	code := cmd.ProcessState.ExitCode()
	if code < 0 {
		code = 1
	}
	return code, nil
}
