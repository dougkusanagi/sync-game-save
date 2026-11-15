//go:build windows

package main

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	user32           = windows.NewLazySystemDLL("user32.dll")
	getForegroundWnd = user32.NewProc("GetForegroundWindow")
	getAncestor      = user32.NewProc("GetAncestor")
)

// janelaEmPrimeiroPlano diz se a janela de topo que recebe a entrada é a do app. O WebView é filho
// dela, então o foco dele conta; o evento de foco do Wails não volta nesse caso.
func janelaEmPrimeiroPlano(hwnd unsafe.Pointer, focada func() bool) bool {
	if hwnd == nil {
		return false
	}
	fg, _, _ := getForegroundWnd.Call()
	if fg == 0 {
		return false
	}
	raiz, _, _ := getAncestor.Call(fg, 2) // GA_ROOT
	return fg == uintptr(hwnd) || raiz == uintptr(hwnd)
}
