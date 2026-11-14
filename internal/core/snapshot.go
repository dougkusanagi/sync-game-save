package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Head struct {
	Snapshot    string `json:"snapshot"`
	IndexDigest string `json:"index_digest"`
	Device      string `json:"device"`
}

func ReadHead(dir string) (*Head, error) {
	b, err := os.ReadFile(filepath.Join(dir, "head.json"))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var v struct {
		Snapshot    *string `json:"snapshot"`
		IndexDigest *string `json:"index_digest"`
		Device      *string `json:"device"`
	}
	if err := json.Unmarshal(b, &v); err != nil {
		return nil, err
	}
	if v.Snapshot == nil || v.IndexDigest == nil || v.Device == nil {
		return nil, errors.New("head inválido")
	}
	return &Head{*v.Snapshot, *v.IndexDigest, *v.Device}, nil
}
func safeKey(key string) bool {
	if key == "" || strings.Contains(key, `\`) || strings.HasPrefix(key, "/") {
		return false
	}
	for _, part := range strings.Split(key, "/") {
		if part == "" || part == "." || part == ".." || strings.Contains(part, ":") {
			return false
		}
	}
	return true
}
func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	_, err = io.Copy(out, in)
	closeErr := out.Close()
	if err != nil {
		return err
	}
	return closeErr
}
func CreateSnapshot(dir string, roots []Root, id, device string, expected *Index) (*Head, error) {
	if !safeKey(id) || strings.Contains(id, "/") {
		return nil, errors.New("identificador de snapshot inválido")
	}
	idx, err := ScanRoots(roots)
	if err != nil {
		return nil, err
	}
	if expected != nil && idx.Digest() != expected.Digest() {
		return nil, errors.New("os arquivos de save mudaram durante a sincronização")
	}
	snapshots := filepath.Join(dir, "snapshots")
	if err := os.MkdirAll(snapshots, 0755); err != nil {
		return nil, err
	}
	snap := filepath.Join(snapshots, id)
	if err := os.Mkdir(snap, 0755); err != nil {
		return nil, err
	}
	files := filepath.Join(snap, "files")
	if err := os.Mkdir(files, 0755); err != nil {
		return nil, err
	}
	for key := range idx.Files {
		if !safeKey(key) {
			return nil, fmt.Errorf("caminho inválido: %s", key)
		}
		src, ok := Locate(roots, key)
		if !ok {
			return nil, fmt.Errorf("origem ausente: %s", key)
		}
		if err := copyFile(src, filepath.Join(files, filepath.FromSlash(key))); err != nil {
			return nil, err
		}
	}
	b, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(snap, "index.json"), b, 0644); err != nil {
		return nil, err
	}
	if _, err := VerifySnapshot(snap); err != nil {
		return nil, err
	}
	head := &Head{id, idx.Digest(), device}
	b, err = json.MarshalIndent(head, "", "  ")
	if err != nil {
		return nil, err
	}
	tmp := filepath.Join(dir, "head.json.tmp")
	if err := os.WriteFile(tmp, b, 0644); err != nil {
		return nil, err
	}
	if err := os.Rename(tmp, filepath.Join(dir, "head.json")); err != nil {
		return nil, err
	}
	return head, nil
}
func VerifySnapshot(snap string) (Index, error) {
	b, err := os.ReadFile(filepath.Join(snap, "index.json"))
	if err != nil {
		return Index{}, err
	}
	var raw struct {
		Files *map[string]struct {
			Size   *uint64 `json:"size"`
			SHA256 *string `json:"sha256"`
		} `json:"files"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return Index{}, err
	}
	if raw.Files == nil {
		return Index{}, errors.New("índice inválido")
	}
	idx := Index{Files: map[string]IndexEntry{}}
	for key, entry := range *raw.Files {
		if !safeKey(key) || entry.Size == nil || entry.SHA256 == nil {
			return Index{}, fmt.Errorf("entrada inválida: %s", key)
		}
		want := IndexEntry{*entry.Size, *entry.SHA256}
		got, err := HashFile(filepath.Join(snap, "files", filepath.FromSlash(key)))
		if err != nil {
			return Index{}, err
		}
		if got != want {
			return Index{}, fmt.Errorf("o arquivo %s não confere com o hash do índice", key)
		}
		idx.Files[key] = want
	}
	return idx, nil
}
func RestoreSnapshot(snap string, roots []Root, backup string) error {
	idx, err := VerifySnapshot(snap)
	if err != nil {
		return err
	}
	current, err := ScanRoots(roots)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(backup), 0755); err != nil {
		return err
	}
	if err := os.Mkdir(backup, 0755); err != nil {
		return err
	}
	keys := func(i Index) []string {
		k := make([]string, 0, len(i.Files))
		for s := range i.Files {
			k = append(k, s)
		}
		sort.Strings(k)
		return k
	}
	for _, key := range keys(current) {
		if !safeKey(key) {
			return fmt.Errorf("caminho inválido: %s", key)
		}
		src, ok := Locate(roots, key)
		if !ok {
			return errors.New("origem ausente")
		}
		if err := copyFile(src, filepath.Join(backup, filepath.FromSlash(key))); err != nil {
			return err
		}
	}
	for _, key := range keys(current) {
		if _, ok := idx.Files[key]; !ok {
			dst, _ := Locate(roots, key)
			if err := os.Remove(dst); err != nil {
				return err
			}
		}
	}
	for _, key := range keys(idx) {
		dst, ok := Locate(roots, key)
		if !ok {
			continue
		}
		if err := copyFile(filepath.Join(snap, "files", filepath.FromSlash(key)), dst); err != nil {
			return err
		}
	}
	return nil
}
