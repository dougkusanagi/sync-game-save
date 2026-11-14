package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSnapshotRestauracaoEBackup(t *testing.T) {
	dir := t.TempDir()
	save, cloud, backup := filepath.Join(dir, "save"), filepath.Join(dir, "cloud"), filepath.Join(dir, "backup")
	if err := os.MkdirAll(save, 0755); err != nil {
		t.Fatal(err)
	}
	slot := filepath.Join(save, "ação.sav")
	if err := os.WriteFile(slot, []byte("v1"), 0644); err != nil {
		t.Fatal(err)
	}
	roots := []Root{{Path: save}}
	head, err := CreateSnapshot(cloud, roots, "s1", "PC-A", nil)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ReadHead(cloud)
	if err != nil || *got != *head {
		t.Fatal(got, err)
	}
	if err := os.WriteFile(slot, []byte("v2"), 0644); err != nil {
		t.Fatal(err)
	}
	extra := filepath.Join(save, "extra")
	if err := os.WriteFile(extra, []byte("novo"), 0644); err != nil {
		t.Fatal(err)
	}
	snap := filepath.Join(cloud, "snapshots", "s1")
	if err := RestoreSnapshot(snap, roots, backup); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(slot)
	if string(b) != "v1" {
		t.Fatal(string(b))
	}
	b, _ = os.ReadFile(filepath.Join(backup, "ação.sav"))
	if string(b) != "v2" {
		t.Fatal(string(b))
	}
	if _, err := os.Stat(extra); !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if err := os.WriteFile(slot, []byte("atual"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := RestoreSnapshot(snap, roots, backup); err == nil {
		t.Fatal("backup existente aceito")
	}
	b, _ = os.ReadFile(slot)
	if string(b) != "atual" {
		t.Fatal(string(b))
	}
	if _, err := CreateSnapshot(cloud, roots, "s1", "PC-A", nil); err == nil {
		t.Fatal("snapshot sobrescrito")
	}
	got, err = ReadHead(cloud)
	if err != nil || *got != *head {
		t.Fatal(got, err)
	}
	if err := os.WriteFile(filepath.Join(snap, "files", "ação.sav"), []byte("danificado"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := VerifySnapshot(snap); err == nil {
		t.Fatal("corrupção não detectada")
	}
}
func TestSnapshotRecusaCaminhoInseguro(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.json"), []byte(`{"files":{"../fora":{"size":0,"sha256":""}}}`), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := VerifySnapshot(dir); err == nil {
		t.Fatal("travessia aceita")
	}
}

func TestRetencaoPreservaHeadERecentes(t *testing.T) {
	dir := t.TempDir()
	save := filepath.Join(dir, "save")
	cloud := filepath.Join(dir, "cloud")
	if err := os.Mkdir(save, 0755); err != nil {
		t.Fatal(err)
	}
	roots := []Root{{Path: save}}
	for _, id := range []string{"100_PC", "200_PC", "300_PC", "400_PC"} {
		if err := os.WriteFile(filepath.Join(save, "a"), []byte(id), 0644); err != nil {
			t.Fatal(err)
		}
		if _, err := CreateSnapshot(cloud, roots, id, "PC", nil); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(cloud, "snapshots", "999_PC"), 0755); err != nil {
		t.Fatal(err)
	}
	ids, err := ListSnapshots(cloud)
	if err != nil || len(ids) != 4 || ids[0] != "400_PC" {
		t.Fatal(ids, err)
	}
	removed, err := PruneSnapshots(cloud, 2)
	if err != nil || len(removed) != 2 || removed[0] != "200_PC" || removed[1] != "100_PC" {
		t.Fatal(removed, err)
	}
}
func TestFalhaDeCopiaPreservaSaveEHead(t *testing.T) {
	dir := t.TempDir()
	save := filepath.Join(dir, "save")
	cloud := filepath.Join(dir, "cloud")
	if err := os.Mkdir(save, 0755); err != nil {
		t.Fatal(err)
	}
	roots := []Root{{Path: save}}
	for _, name := range []string{"a", "b"} {
		if err := os.WriteFile(filepath.Join(save, name), []byte("nuvem"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	head, err := CreateSnapshot(cloud, roots, "1_PC", "PC", nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"a", "b"} {
		if err := os.WriteFile(filepath.Join(save, name), []byte("local"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	calls := 0
	fail := func(src, dst string) error {
		calls++
		if calls == 2 {
			return os.ErrPermission
		}
		return copyFile(src, dst)
	}
	if err := restoreSnapshot(filepath.Join(cloud, "snapshots", "1_PC"), roots, filepath.Join(dir, "backup"), fail); err == nil {
		t.Fatal("falha de backup ignorada")
	}
	for _, name := range []string{"a", "b"} {
		b, err := os.ReadFile(filepath.Join(save, name))
		if err != nil || string(b) != "local" {
			t.Fatal(name, string(b), err)
		}
	}
	calls = 0
	if _, err := createSnapshot(cloud, roots, "2_PC", "PC", nil, fail); err == nil {
		t.Fatal("falha de snapshot ignorada")
	}
	got, err := ReadHead(cloud)
	if err != nil || *got != *head {
		t.Fatal(got, err)
	}
}
func TestFixtureGeradaPeloRust(t *testing.T) {
	cloud := filepath.Join("testdata", "rust", "cloud")
	head, err := ReadHead(cloud)
	if err != nil || head == nil {
		t.Fatal(head, err)
	}
	snap := filepath.Join(cloud, "snapshots", head.Snapshot)
	idx, err := VerifySnapshot(snap)
	if err != nil || idx.Digest() != head.IndexDigest {
		t.Fatal(idx, head, err)
	}
	dir := t.TempDir()
	save := filepath.Join(dir, "save")
	if err := RestoreSnapshot(snap, []Root{{Path: save}}, filepath.Join(dir, "backup")); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(save, "pasta", "ação.sav"))
	if err != nil || string(b) != "progresso do Rust\n" {
		t.Fatal(string(b), err)
	}
	// Atualiza uma cópia da nuvem legada e preserva o snapshot original.
	cloudCopy := filepath.Join(dir, "cloud")
	if err := filepath.Walk(cloud, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(cloud, path)
		if err != nil {
			return err
		}
		dst := filepath.Join(cloudCopy, rel)
		if info.IsDir() {
			return os.MkdirAll(dst, 0755)
		}
		return copyFile(path, dst)
	}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(save, "pasta", "ação.sav"), []byte("progresso Go"), 0644); err != nil {
		t.Fatal(err)
	}
	newHead, err := CreateSnapshot(cloudCopy, []Root{{Path: save}}, "200_PC", "PC", nil)
	if err != nil {
		t.Fatal(err)
	}
	if newHead.Snapshot != "200_PC" {
		t.Fatal(newHead)
	}
	if _, err := VerifySnapshot(filepath.Join(cloudCopy, "snapshots", head.Snapshot)); err != nil {
		t.Fatal(err)
	}
}
