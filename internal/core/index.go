package core

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type IndexEntry struct {
	Size   uint64 `json:"size"`
	SHA256 string `json:"sha256"`
}
type Index struct {
	Files map[string]IndexEntry `json:"files"`
}
type Root struct{ Label, Path string }

func (i Index) Digest() string {
	keys := make([]string, 0, len(i.Files))
	for k := range i.Files {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	h := sha256.New()
	for _, k := range keys {
		e := i.Files[k]
		fmt.Fprintf(h, "%s\x00%d\x00%s\n", k, e.Size, e.SHA256)
	}
	return hex.EncodeToString(h.Sum(nil))
}
func HashFile(path string) (IndexEntry, error) {
	var b []byte
	err := retryFile(func() error { var err error; b, err = os.ReadFile(path); return err })
	if err != nil {
		return IndexEntry{}, err
	}
	sum := sha256.Sum256(b)
	return IndexEntry{uint64(len(b)), hex.EncodeToString(sum[:])}, nil
}
func Scan(root string) (Index, error) {
	idx := Index{Files: make(map[string]IndexEntry)}
	info, err := os.Stat(root)
	if os.IsNotExist(err) {
		return idx, nil
	}
	if err != nil {
		return idx, err
	}
	if !info.IsDir() {
		return idx, nil
	}
	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		// Walk does not follow symlinked directories; Rust follows them.
		if info.Mode()&os.ModeSymlink != 0 {
			target, err := os.Stat(path)
			if err != nil {
				return err
			}
			if target.IsDir() {
				return fmt.Errorf("pasta simbólica não suportada: %s", path)
			}
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		entry, err := HashFile(path)
		if err != nil {
			return err
		}
		idx.Files[filepath.ToSlash(rel)] = entry
		return nil
	})
	return idx, err
}
func Prefix(label string) string {
	return strings.Trim(strings.Map(func(r rune) rune {
		if strings.ContainsRune(`<>:"|?*\`, r) {
			return -1
		}
		return r
	}, label), "/")
}
func ScanRoots(roots []Root) (Index, error) {
	all := Index{Files: make(map[string]IndexEntry)}
	for _, root := range roots {
		idx, err := Scan(root.Path)
		if err != nil {
			return Index{}, err
		}
		pre := Prefix(root.Label)
		for rel, entry := range idx.Files {
			if pre != "" {
				rel = pre + "/" + rel
			}
			all.Files[rel] = entry
		}
	}
	return all, nil
}
func Locate(roots []Root, key string) (string, bool) {
	best := -1
	var result string
	for _, root := range roots {
		pre := Prefix(root.Label)
		rel := key
		if pre != "" {
			var ok bool
			rel, ok = strings.CutPrefix(key, pre+"/")
			if !ok {
				continue
			}
		}
		if len(pre) >= best {
			best = len(pre)
			result = filepath.Join(root.Path, filepath.FromSlash(rel))
		}
	}
	return result, best >= 0
}
