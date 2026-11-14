package core

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

var discoveryRoots = []string{"<home>/Saved Games", "<winDocuments>", "<winPublic>/Documents", "<winAppData>", "<winLocalAppData>", "<home>/AppData/LocalLow", "<base>"}

type FileStat struct {
	Size  uint64 `json:"size"`
	Mtime uint64 `json:"mtime"`
}
type Capture struct {
	TakenAt uint64              `json:"taken_at"`
	Files   map[string]FileStat `json:"files"`
}
type Candidate struct {
	Logical string `json:"logical"`
	Added   uint64 `json:"added"`
	Removed uint64 `json:"removed"`
	Bytes   uint64 `json:"bytes"`
}

func DiscoveryKey(exe string) string {
	h := sha256.Sum256([]byte(exe))
	return hex.EncodeToString(h[:])
}
func discoveryNoise(path string) bool {
	name := strings.ToLower(strings.ReplaceAll(path, `\`, `/`))
	if i := strings.LastIndexByte(name, '/'); i >= 0 {
		name = name[i+1:]
	}
	switch name {
	case "cache", "caches", "code cache", "gpucache", "d3dscache", "shadercache", "shaders", "shader", "log", "logs", "temp", "tmp", "crash", "crashes", "crashdumps", "dump", "reports":
		return true
	}
	switch filepath.Ext(name) {
	case ".log", ".dmp", ".tmp", ".temp":
		return true
	}
	return false
}
func CaptureNow(env Env) Capture {
	now := time.Now().Unix()
	if now < 0 {
		now = 0
	}
	return CaptureAt(env, uint64(now))
}
func CaptureAt(env Env, takenAt uint64) Capture {
	result := Capture{TakenAt: takenAt, Files: map[string]FileStat{}}
	type node struct {
		path  string
		depth int
	}
	for _, root := range discoveryRoots {
		dir, ok := Resolve(root, env)
		if !ok {
			continue
		}
		items := []node{{dir, 0}}
		for len(items) > 0 {
			n := items[len(items)-1]
			items = items[:len(items)-1]
			entries, err := os.ReadDir(n.path)
			if err != nil {
				continue
			}
			for _, entry := range entries {
				path := filepath.Join(n.path, entry.Name())
				if entry.IsDir() {
					if n.depth < 8 && !discoveryNoise(path) {
						items = append(items, node{path, n.depth + 1})
					}
					continue
				}
				logical, ok := ToLogical(path, env)
				if !ok || discoveryNoise(logical) {
					continue
				}
				stat := FileStat{}
				info, err := os.Stat(path)
				if err == nil {
					stat.Size = uint64(info.Size())
					ns := info.ModTime().UnixNano()
					if ns > 0 {
						stat.Mtime = uint64(ns)
					}
				}
				result.Files[logical] = stat
			}
		}
	}
	return result
}
func discoveryGroup(logical string) (string, bool) {
	for _, root := range discoveryRoots {
		rel, ok := strings.CutPrefix(logical, root+"/")
		if !ok {
			continue
		}
		parts := strings.Split(rel, "/")
		depth := 1
		if root == "<home>/AppData/LocalLow" {
			depth = 2
		}
		if strings.EqualFold(parts[0], "my games") {
			depth = 2
		}
		if depth > len(parts)-1 {
			depth = len(parts) - 1
		}
		if depth == 0 {
			return root, true
		}
		return root + "/" + strings.Join(parts[:depth], "/"), true
	}
	return "", false
}
func CompareCaptures(before, after Capture) []Candidate {
	groups := map[string]*Candidate{}
	get := func(key string) *Candidate {
		c := groups[key]
		if c == nil {
			c = &Candidate{Logical: key}
			groups[key] = c
		}
		return c
	}
	for path, stat := range after.Files {
		old, ok := before.Files[path]
		if ok && old == stat {
			continue
		}
		if group, ok := discoveryGroup(path); ok {
			c := get(group)
			c.Added++
			c.Bytes += stat.Size
		}
	}
	for path := range before.Files {
		if _, ok := after.Files[path]; ok {
			continue
		}
		if group, ok := discoveryGroup(path); ok {
			get(group).Removed++
		}
	}
	result := make([]Candidate, 0, len(groups))
	for _, c := range groups {
		result = append(result, *c)
	}
	sort.Slice(result, func(i, j int) bool {
		a, b := result[i], result[j]
		if a.Added+a.Removed != b.Added+b.Removed {
			return a.Added+a.Removed > b.Added+b.Removed
		}
		return a.Logical < b.Logical
	})
	return result
}
func CompareCapture(before Capture, env Env) []Candidate {
	return CompareCaptures(before, CaptureNow(env))
}
