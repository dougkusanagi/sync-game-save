package core

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func enginePC(dir, name string, now uint64) Engine {
	return Engine{GameDir: filepath.Join(dir, "cloud"), Roots: []Root{{Path: filepath.Join(dir, name, "save")}}, StateDir: filepath.Join(dir, name, "state"), Device: name, Now: now}
}
func enginePut(t *testing.T, e Engine, value string) {
	t.Helper()
	if err := os.MkdirAll(e.Roots[0].Path, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(e.Roots[0].Path, "s.sav"), []byte(value), 0644); err != nil {
		t.Fatal(err)
	}
}
func engineGet(t *testing.T, e Engine) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(e.Roots[0].Path, "s.sav"))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
func noSleep(time.Duration) {}
func TestEngineDoisPCsEmSequencia(t *testing.T) {
	dir := t.TempDir()
	a := enginePC(dir, "A", 1000)
	enginePut(t, a, "v1")
	r, err := a.BeforeLaunch(false)
	if err != nil || r.Status != UploadedLocal {
		t.Fatal(r, err)
	}
	enginePut(t, a, "v2")
	created, err := a.afterExit(nil, noSleep)
	if err != nil || !created {
		t.Fatal(created, err)
	}
	b := enginePC(dir, "B", 2000)
	r, err = b.BeforeLaunch(false)
	if err != nil || r.Status != RestoredFromCloud || engineGet(t, b) != "v2" {
		t.Fatal(r, err)
	}
	enginePut(t, b, "v3")
	created, err = b.afterExit(nil, noSleep)
	if err != nil || !created {
		t.Fatal(created, err)
	}
	a.Now = 3000
	r, err = a.BeforeLaunch(false)
	if err != nil || r.Status != RestoredFromCloud || engineGet(t, a) != "v3" {
		t.Fatal(r, err)
	}
	created, err = a.afterExit(nil, noSleep)
	if err != nil || created {
		t.Fatal(created, err)
	}
}
func TestEngineConflitoPreservaBackup(t *testing.T) {
	dir := t.TempDir()
	a := enginePC(dir, "A", 1000)
	enginePut(t, a, "base")
	if _, err := a.BeforeLaunch(false); err != nil {
		t.Fatal(err)
	}
	if _, err := a.afterExit(nil, noSleep); err != nil {
		t.Fatal(err)
	}
	b := enginePC(dir, "B", 2000)
	if _, err := b.BeforeLaunch(false); err != nil {
		t.Fatal(err)
	}
	if _, err := b.afterExit(nil, noSleep); err != nil {
		t.Fatal(err)
	}
	enginePut(t, a, "de A")
	enginePut(t, b, "de B")
	if _, err := b.afterExit(nil, noSleep); err != nil {
		t.Fatal(err)
	}
	a.Now = 4000
	r, err := a.BeforeLaunch(false)
	if err != nil || r.Status != SyncConflict || engineGet(t, a) != "de A" {
		t.Fatal(r, err)
	}
	if err := a.Resolve(UseCloud); err != nil {
		t.Fatal(err)
	}
	if engineGet(t, a) != "de B" {
		t.Fatal("restauração incorreta")
	}
	backup, err := os.ReadFile(filepath.Join(a.StateDir, "backups", "4000_conflito", "s.sav"))
	if err != nil || string(backup) != "de A" {
		t.Fatal(string(backup), err)
	}
}
func TestEngineTravaEEstabilizacao(t *testing.T) {
	dir := t.TempDir()
	a := enginePC(dir, "A", 1000)
	enginePut(t, a, "inicial")
	if _, err := a.BeforeLaunch(false); err != nil {
		t.Fatal(err)
	}
	b := enginePC(dir, "B", 1100)
	r, err := b.BeforeLaunch(false)
	if err != nil || r.Status != LockedByOther || r.Device != "A" {
		t.Fatal(r, err)
	}
	ticks := 0
	created, err := a.afterExit(func() {
		ticks++
		if ticks == 4 {
			enginePut(t, a, "final")
		}
	}, noSleep)
	if err != nil || !created || ticks != 5 {
		t.Fatal(created, ticks, err)
	}
	head, err := ReadHead(a.GameDir)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(a.GameDir, "snapshots", head.Snapshot, "files", "s.sav"))
	if err != nil || string(data) != "final" {
		t.Fatal(string(data), err)
	}
}
