//go:build linux

package core

import (
	"fmt"
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
		parent, started, err := parseProcessStat(data)
		if err != nil {
			continue
		}
		exe, _ := os.Readlink(filepath.Join(base, "exe"))
		result = append(result, ProcessInfo{PID: uint32(pid), Parent: uint32(parent), Started: started, Executable: exe})
	}
	return result, nil
}

func parseProcessStat(data []byte) (uint64, uint64, error) {
	close := strings.LastIndexByte(string(data), ')')
	if close < 0 {
		return 0, 0, fmt.Errorf("stat sem nome de processo válido")
	}
	fields := strings.Fields(string(data[close+1:]))
	if len(fields) < 20 {
		return 0, 0, fmt.Errorf("stat incompleto")
	}
	parent, err := strconv.ParseUint(fields[1], 10, 32)
	if err != nil {
		return 0, 0, err
	}
	started, err := strconv.ParseUint(fields[19], 10, 64)
	return parent, started, err
}
