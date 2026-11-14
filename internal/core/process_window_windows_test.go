//go:build windows

package core

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	createWindowEx   = user32.NewProc("CreateWindowExW")
	peekMessage      = user32.NewProc("PeekMessageW")
	translateMessage = user32.NewProc("TranslateMessage")
	dispatchMessage  = user32.NewProc("DispatchMessageW")
	isWindow         = user32.NewProc("IsWindow")
)

type testMessage struct {
	hwnd           uintptr
	message        uint32
	wParam, lParam uintptr
	time           uint32
	x, y           int32
	private        uint32
}

// TestWindowHelper cria uma janela real, oculta, em processo separado.
func TestWindowHelper(t *testing.T) {
	if os.Getenv("SGS_WINDOW_HELPER") != "1" {
		return
	}
	var marker string
	for i, arg := range os.Args {
		if arg == "--" && i+1 < len(os.Args) {
			marker = os.Args[i+1]
			break
		}
	}
	if marker == "" {
		os.Exit(2)
	}
	runtime.LockOSThread()
	class, _ := windows.UTF16PtrFromString("STATIC")
	name, _ := windows.UTF16PtrFromString("SGS teste")
	hwnd, _, _ := createWindowEx.Call(0, uintptr(unsafe.Pointer(class)), uintptr(unsafe.Pointer(name)), 0, 0, 0, 100, 100, 0, 0, 0, 0)
	if hwnd == 0 {
		os.Exit(3)
	}
	if os.WriteFile(marker, []byte("aberta"), 0600) != nil {
		os.Exit(4)
	}
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		var msg testMessage
		for {
			found, _, _ := peekMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0, 1)
			if found == 0 {
				break
			}
			translateMessage.Call(uintptr(unsafe.Pointer(&msg)))
			dispatchMessage.Call(uintptr(unsafe.Pointer(&msg)))
		}
		exists, _, _ := isWindow.Call(hwnd)
		if exists == 0 {
			os.Exit(0)
		}
		time.Sleep(10 * time.Millisecond)
	}
	os.Exit(5)
}

func TestStopProcessFechaJanelaSemAtingirOutra(t *testing.T) {
	t.Setenv("SGS_WINDOW_HELPER", "1")
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(exe)
	if err != nil {
		t.Fatal(err)
	}
	prepare := func() (string, string) {
		t.Helper()
		dir := t.TempDir()
		copy := filepath.Join(dir, "jogo.exe")
		if err := os.WriteFile(copy, data, 0755); err != nil {
			t.Fatal(err)
		}
		return copy, filepath.Join(dir, "pronto")
	}
	unrelatedExe, unrelatedMarker := prepare()
	unrelated := exec.Command(unrelatedExe, "-test.run=^TestWindowHelper$", "--", unrelatedMarker)
	if err := unrelated.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = unrelated.Process.Kill(); _ = unrelated.Wait() })
	game, marker := prepare()
	done := make(chan error, 1)
	go func() {
		code, err := RunProcess(game, []string{"-test.run=^TestWindowHelper$", "--", marker}, "", 100*time.Millisecond, time.Hour, nil)
		if err == nil && code != 0 {
			err = fmt.Errorf("jogo terminou com código %d", code)
		}
		done <- err
	}()
	deadline := time.Now().Add(10 * time.Second)
	for {
		_, e1 := os.Stat(marker)
		_, e2 := os.Stat(unrelatedMarker)
		if e1 == nil && e2 == nil && ProcessRunning(game) {
			break
		}
		select {
		case err := <-done:
			t.Fatal("jogo encerrou antes do teste:", err)
		default:
		}
		if time.Now().After(deadline) {
			t.Fatal("janelas não ficaram prontas")
		}
		time.Sleep(20 * time.Millisecond)
	}
	found, running, err := StopProcess(game)
	if err != nil || !found || running {
		t.Fatal(found, running, err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("jogo não terminou")
	}
	h, err := windows.OpenProcess(windows.SYNCHRONIZE, false, uint32(unrelated.Process.Pid))
	if err != nil {
		t.Fatal(err)
	}
	state, err := windows.WaitForSingleObject(h, 0)
	windows.CloseHandle(h)
	if err != nil || state != uint32(windows.WAIT_TIMEOUT) {
		t.Fatal("outra janela foi fechada", state, err)
	}
}
