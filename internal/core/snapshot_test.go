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
