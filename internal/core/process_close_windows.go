//go:build windows

package core

import (
	"errors"
	"fmt"
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	user32                   = windows.NewLazySystemDLL("user32.dll")
	getWindowThreadProcessID = user32.NewProc("GetWindowThreadProcessId")
	postMessage              = user32.NewProc("PostMessageW")
	kernel32Job              = windows.NewLazySystemDLL("kernel32.dll")
	isProcessInJob           = kernel32Job.NewProc("IsProcessInJob")
	closeWindowCallback      = windows.NewCallback(closeJobWindow)
)

type closeWindowContext struct {
	job     windows.Handle
	tracked map[uint32]uint64
	err     error
}

func processStarted(h windows.Handle) (uint64, error) {
	var created, exit, kernel, user windows.Filetime
	if err := windows.GetProcessTimes(h, &created, &exit, &kernel, &user); err != nil {
		return 0, err
	}
	return uint64(created.HighDateTime)<<32 | uint64(created.LowDateTime), nil
}

func inProcessJob(h, job windows.Handle) bool {
	var member int32
	ok, _, _ := isProcessInJob.Call(uintptr(h), uintptr(job), uintptr(unsafe.Pointer(&member)))
	return ok != 0 && member != 0
}

func closeJobWindow(hwnd uintptr, param uintptr) uintptr {
	ctx := (*closeWindowContext)(unsafe.Pointer(param))
	var pid uint32
	getWindowThreadProcessID.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	if pid == 0 || pid == uint32(os.Getpid()) {
		return 1
	}
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return 1
	}
	defer windows.CloseHandle(h)
	started, err := processStarted(h)
	tracked := err == nil && started != 0 && ctx.tracked[pid] == started
	if inProcessJob(h, ctx.job) || tracked {
		result, _, callErr := postMessage.Call(hwnd, 0x0010, 0, 0)
		if result == 0 && ctx.err == nil {
			ctx.err = fmt.Errorf("não foi possível pedir o fechamento do jogo: %w", callErr)
		}
	}
	return 1
}

// trackedProcesses aceita apenas processos do instantâneo atual com identidade conhecida.
func trackedProcesses(tracker *ProcessTracker, processes []ProcessInfo) map[uint32]uint64 {
	tracked := make(map[uint32]uint64)
	for _, p := range processes {
		if p.PID != uint32(os.Getpid()) && p.Started != 0 && tracker.Contains(p.PID) && tracker.starts[p.PID] == p.Started {
			tracked[p.PID] = p.Started
		}
	}
	return tracked
}

func (j *processJob) requestClose(tracker *ProcessTracker, processes []ProcessInfo) error {
	ctx := closeWindowContext{job: j.handle, tracked: trackedProcesses(tracker, processes)}
	if err := windows.EnumWindows(closeWindowCallback, unsafe.Pointer(&ctx)); err != nil {
		return fmt.Errorf("não foi possível procurar as janelas do jogo: %w", err)
	}
	return ctx.err
}

func forceStopAvailable() error { return nil }
func (j *processJob) forceStop(tracker *ProcessTracker, processes []ProcessInfo) error {
	var failures []error
	for _, p := range processes {
		if p.PID != uint32(os.Getpid()) && tracker.Contains(p.PID) && (p.Started == 0 || tracker.starts[p.PID] == 0) {
			failures = append(failures, fmt.Errorf("não foi possível identificar o processo %d", p.PID))
		}
	}
	// O handle mantém a identidade do processo mesmo se o PID for reutilizado.
	for pid, expected := range trackedProcesses(tracker, processes) {
		h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION|windows.PROCESS_TERMINATE, false, pid)
		if errors.Is(err, windows.ERROR_INVALID_PARAMETER) {
			continue
		}
		if err != nil {
			failures = append(failures, fmt.Errorf("não foi possível acessar o processo %d: %w", pid, err))
			continue
		}
		started, err := processStarted(h)
		if err != nil {
			failures = append(failures, fmt.Errorf("não foi possível identificar o processo %d: %w", pid, err))
		} else if started == expected && !inProcessJob(h, j.handle) {
			if err := windows.TerminateProcess(h, 1); err != nil {
				failures = append(failures, fmt.Errorf("não foi possível encerrar o processo %d: %w", pid, err))
			}
		}
		windows.CloseHandle(h)
	}
	if err := j.terminate(); err != nil {
		failures = append(failures, fmt.Errorf("não foi possível encerrar o grupo do jogo: %w", err))
	}
	return errors.Join(failures...)
}
