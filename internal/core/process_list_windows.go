//go:build windows

package core

import (
	"errors"
	"unsafe"

	"golang.org/x/sys/windows"
)

// ListProcesses retorna o instantâneo do sistema e os dados disponíveis de cada processo.
func ListProcesses() ([]ProcessInfo, error) {
	snap, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, err
	}
	defer windows.CloseHandle(snap)
	entry := windows.ProcessEntry32{Size: uint32(unsafe.Sizeof(windows.ProcessEntry32{}))}
	err = windows.Process32First(snap, &entry)
	result := []ProcessInfo{}
	for err == nil {
		p := ProcessInfo{PID: entry.ProcessID, Parent: entry.ParentProcessID}
		h, e := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, p.PID)
		if e == nil {
			var created, exit, kernel, user windows.Filetime
			if windows.GetProcessTimes(h, &created, &exit, &kernel, &user) == nil {
				p.Started = uint64(created.HighDateTime)<<32 | uint64(created.LowDateTime)
			}
			buf := make([]uint16, 32768)
			size := uint32(len(buf))
			if windows.QueryFullProcessImageName(h, 0, &buf[0], &size) == nil {
				p.Executable = windows.UTF16ToString(buf[:size])
			}
			windows.CloseHandle(h)
		}
		result = append(result, p)
		err = windows.Process32Next(snap, &entry)
	}
	if !errors.Is(err, windows.ERROR_NO_MORE_FILES) {
		return nil, err
	}
	return result, nil
}
