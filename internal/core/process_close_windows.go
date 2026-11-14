//go:build windows

package core

import (
	"fmt"
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
	job windows.Handle
	err error
}

func closeJobWindow(hwnd uintptr, param uintptr) uintptr {
	ctx := (*closeWindowContext)(unsafe.Pointer(param))
	var pid uint32
	getWindowThreadProcessID.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	if pid == 0 {
		return 1
	}
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return 1
	}
	defer windows.CloseHandle(h)
	var member int32
	ok, _, _ := isProcessInJob.Call(uintptr(h), uintptr(ctx.job), uintptr(unsafe.Pointer(&member)))
	if ok != 0 && member != 0 {
		result, _, callErr := postMessage.Call(hwnd, 0x0010, 0, 0)
		if result == 0 && ctx.err == nil {
			ctx.err = fmt.Errorf("não foi possível pedir o fechamento do jogo: %w", callErr)
		}
	}
	return 1
}

// requestClose envia WM_CLOSE somente às janelas dos processos associados ao Job.
func (j *processJob) requestClose() error {
	ctx := closeWindowContext{job: j.handle}
	if err := windows.EnumWindows(closeWindowCallback, unsafe.Pointer(&ctx)); err != nil {
		return fmt.Errorf("não foi possível procurar as janelas do jogo: %w", err)
	}
	return ctx.err
}

func forceStopAvailable() error { return nil }
func (j *processJob) forceStop() error {
	if err := j.terminate(); err != nil {
		return fmt.Errorf("não foi possível forçar o fechamento do jogo: %w", err)
	}
	return nil
}
