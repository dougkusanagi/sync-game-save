//go:build windows

package core

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	isWindowVisible     = user32.NewProc("IsWindowVisible")
	getWindowTextLength = user32.NewProc("GetWindowTextLengthW")
	getWindow           = user32.NewProc("GetWindow")
	isIconic            = user32.NewProc("IsIconic")
	showWindow          = user32.NewProc("ShowWindow")
	setForegroundWindow = user32.NewProc("SetForegroundWindow")
	focusCallback       = windows.NewCallback(findGameWindow)
)

type focusContext struct {
	pids  map[uint32]bool
	found uintptr
}

// findGameWindow escolhe a primeira janela principal (visível, sem dona e com título) do jogo.
func findGameWindow(hwnd uintptr, param uintptr) uintptr {
	ctx := (*focusContext)(unsafe.Pointer(param))
	var pid uint32
	getWindowThreadProcessID.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	if !ctx.pids[pid] {
		return 1
	}
	if v, _, _ := isWindowVisible.Call(hwnd); v == 0 {
		return 1
	}
	if owner, _, _ := getWindow.Call(hwnd, 4); owner != 0 { // GW_OWNER
		return 1
	}
	if n, _, _ := getWindowTextLength.Call(hwnd); n == 0 {
		return 1
	}
	ctx.found = hwnd
	return 0
}

func focusPIDs(pids map[uint32]bool) (bool, error) {
	if len(pids) == 0 {
		return false, nil
	}
	ctx := focusContext{pids: pids}
	// EnumWindows devolve erro quando o callback para a busca; o que importa é ctx.found.
	_ = windows.EnumWindows(focusCallback, unsafe.Pointer(&ctx))
	if ctx.found == 0 {
		return false, nil
	}
	if min, _, _ := isIconic.Call(ctx.found); min != 0 {
		showWindow.Call(ctx.found, 9) // SW_RESTORE
	}
	ok, _, _ := setForegroundWindow.Call(ctx.found)
	return ok != 0, nil
}
