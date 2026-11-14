//go:build linux

package core

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func ListProcesses() ([]ProcessInfo, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, err
	}
	result := []ProcessInfo{}
	for _, entry := range entries {
		pid, err := strconv.ParseUint(entry.Name(), 10, 32)
		if err != nil || !entry.IsDir() {
			continue
		}
		base := filepath.Join("/proc", entry.Name())
		data, err := os.ReadFile(filepath.Join(base, "stat"))
		if err != nil {
			continue
		}
		close := strings.LastIndexByte(string(data), ')')
		if close < 0 {
			continue
		}
		fields := strings.Fields(string(data[close+1:]))
		if len(fields) < 20 {
			continue
		}
		parent, err := strconv.ParseUint(fields[1], 10, 32)
		if err != nil {
			continue
		}
		started, err := strconv.ParseUint(fields[19], 10, 64)
		if err != nil {
			continue
		}
		exe, _ := os.Readlink(filepath.Join(base, "exe"))
		result = append(result, ProcessInfo{PID: uint32(pid), Parent: uint32(parent), Started: started, Executable: exe})
	}
	return result, nil
}
