//go:build linux

package core

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"golang.org/x/sys/unix"
)

// A versão anterior também não envia pedido de fechamento normal no Linux.
func (*processJob) requestClose(_ *ProcessTracker, _ []ProcessInfo) error { return nil }
func forceStopAvailable() error                                           { return nil }

// O pidfd mantém a identidade após a abertura, mesmo que o PID seja reutilizado.
func (*processJob) forceStop(tracker *ProcessTracker, processes []ProcessInfo) error {
	var failures []error
	for _, p := range processes {
		if p.PID == uint32(os.Getpid()) || !tracker.Contains(p.PID) {
			continue
		}
		expected := tracker.starts[p.PID]
		if p.Started == 0 || expected == 0 {
			failures = append(failures, fmt.Errorf("não foi possível identificar o processo %d", p.PID))
			continue
		}
		if p.Started != expected {
			continue
		}
		fd, err := unix.PidfdOpen(int(p.PID), 0)
		if errors.Is(err, unix.ESRCH) {
			continue
		}
		if err != nil {
			failures = append(failures, fmt.Errorf("não foi possível acessar o processo %d: %w", p.PID, err))
			continue
		}
		data, err := os.ReadFile(filepath.Join("/proc", strconv.FormatUint(uint64(p.PID), 10), "stat"))
		if errors.Is(err, os.ErrNotExist) {
			unix.Close(fd)
			continue
		}
		if err != nil {
			failures = append(failures, fmt.Errorf("não foi possível identificar o processo %d: %w", p.PID, err))
			unix.Close(fd)
			continue
		}
		_, started, err := parseProcessStat(data)
		if err != nil {
			failures = append(failures, fmt.Errorf("não foi possível identificar o processo %d: %w", p.PID, err))
		} else if started == expected {
			err = unix.PidfdSendSignal(fd, unix.SIGKILL, nil, 0)
			if err != nil && !errors.Is(err, unix.ESRCH) {
				failures = append(failures, fmt.Errorf("não foi possível encerrar o processo %d: %w", p.PID, err))
			}
		}
		unix.Close(fd)
	}
	return errors.Join(failures...)
}
