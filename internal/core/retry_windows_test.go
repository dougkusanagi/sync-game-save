//go:build windows

package core

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

func TestArquivoOcupadoPodeSerLidoDepois(t *testing.T) {
	dir := t.TempDir()
	save := filepath.Join(dir, "save")
	if err := os.Mkdir(save, 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(save, "slot")
	if err := os.WriteFile(path, []byte("save"), 0644); err != nil {
		t.Fatal(err)
	}
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		t.Fatal(err)
	}
	handle, err := windows.CreateFile(name, windows.GENERIC_READ, 0, nil, windows.OPEN_EXISTING, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() { time.Sleep(600 * time.Millisecond); windows.CloseHandle(handle); close(done) }()
	_, err = CreateSnapshot(filepath.Join(dir, "cloud"), []Root{{Path: save}}, "1_PC", "PC", nil)
	<-done
	if err != nil {
		t.Fatal(err)
	}
}
