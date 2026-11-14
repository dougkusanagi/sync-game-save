package core

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type ProcessInfo struct {
	PID, Parent uint32
	Started     uint64
	Executable  string
}
type ProcessTracker struct {
	pids   map[uint32]struct{}
	starts map[uint32]uint64
	dir    string
}

func processPath(path string) string {
	path = strings.TrimRight(filepath.ToSlash(path), "/")
	if runtime.GOOS == "windows" {
		path = strings.ToLower(path)
	}
	return path
}
func NewProcessTracker(root uint32, dir string) *ProcessTracker {
	normalized := processPath(dir)
	if strings.Count(normalized, "/") < 2 {
		normalized = ""
	}
	return &ProcessTracker{pids: map[uint32]struct{}{root: {}}, starts: map[uint32]uint64{}, dir: normalized}
}

// Refresh atualiza os descendentes e processos reiniciados a partir da pasta do jogo.
func (t *ProcessTracker) Refresh(processes []ProcessInfo) bool {
	byPID := make(map[uint32]ProcessInfo, len(processes))
	for _, p := range processes {
		byPID[p.PID] = p
	}
	for pid := range t.pids {
		if p, ok := byPID[pid]; ok {
			if start, known := t.starts[pid]; known && p.Started != 0 && start != p.Started {
				delete(t.pids, pid)
				delete(t.starts, pid)
			}
		}
	}
	if t.dir != "" {
		prefix := t.dir + "/"
		for _, p := range processes {
			if p.PID != uint32(os.Getpid()) && p.Executable != "" && strings.HasPrefix(processPath(p.Executable), prefix) {
				t.pids[p.PID] = struct{}{}
			}
		}
	}
	for {
		changed := false
		for _, p := range processes {
			if _, ok := t.pids[p.PID]; ok {
				continue
			}
			if _, ok := t.pids[p.Parent]; ok {
				t.pids[p.PID] = struct{}{}
				changed = true
			}
		}
		if !changed {
			break
		}
	}
	live := false
	for pid := range t.pids {
		if p, ok := byPID[pid]; ok {
			if p.Started != 0 {
				t.starts[pid] = p.Started
			}
			live = true
		}
	}
	return live
}
func (t *ProcessTracker) Contains(pid uint32) bool { _, ok := t.pids[pid]; return ok }
