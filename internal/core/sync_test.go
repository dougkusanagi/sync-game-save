package core

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestIndiceETresVias(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "a"), 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "a", "slot.sav")
	if err := os.WriteFile(path, []byte("um"), 0644); err != nil {
		t.Fatal(err)
	}
	a, err := Scan(dir)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte("um"))
	entry := a.Files["a/slot.sav"]
	if entry.Size != 2 || entry.SHA256 != hex.EncodeToString(sum[:]) {
		t.Fatal(entry)
	}
	// Digest segue a representação usada pelo BTreeMap da versão Rust.
	expected := sha256.Sum256([]byte("a/slot.sav\x002\x00" + entry.SHA256 + "\n"))
	if a.Digest() != hex.EncodeToString(expected[:]) {
		t.Fatal(a.Digest())
	}
	if err := os.WriteFile(path, []byte("dois"), 0644); err != nil {
		t.Fatal(err)
	}
	b, err := Scan(dir)
	if err != nil {
		t.Fatal(err)
	}
	da, db := a.Digest(), b.Digest()
	x := "x"
	cases := []struct {
		base  *string
		local Index
		cloud *string
		want  Action
	}{{&da, a, &da, Nothing}, {&da, a, &db, RestoreFromCloud}, {&da, b, &da, UploadLocal}, {&x, b, &da, Conflict}, {&x, b, &db, Nothing}, {nil, b, nil, UploadLocal}, {nil, Index{Files: map[string]IndexEntry{}}, &da, RestoreFromCloud}}
	for _, c := range cases {
		if got := Decide(c.base, c.local, c.cloud); got != c.want {
			t.Errorf("decisão %v, esperada %v", got, c.want)
		}
	}
}
func TestTravaDeOutroPC(t *testing.T) {
	dir := t.TempDir()
	if err := AcquireLock(dir, "A", 100); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		device string
		now    uint64
		want   LockStatus
	}{{"A", 5000, LockOwn}, {"B", 699, LockHeldByOther}, {"B", 700, LockStale}, {"B", 1, LockHeldByOther}} {
		got, _, err := LockState(dir, c.device, c.now)
		if err != nil || got != c.want {
			t.Fatalf("%v %v", got, err)
		}
	}
	if err := HeartbeatLock(dir, "A", 1000); err != nil {
		t.Fatal(err)
	}
	got, l, err := LockState(dir, "B", 1060)
	if err != nil || got != LockHeldByOther || l.Started != 100 {
		t.Fatalf("%v %v %v", got, l, err)
	}
	if err := ReleaseLock(dir, "B"); err != nil {
		t.Fatal(err)
	}
	got, _, err = LockState(dir, "B", 1060)
	if err != nil || got != LockHeldByOther {
		t.Fatal(got, err)
	}
	if err := ReleaseLock(dir, "A"); err != nil {
		t.Fatal(err)
	}
	got, _, err = LockState(dir, "B", 1060)
	if err != nil || got != LockFree {
		t.Fatal(got, err)
	}
}
func TestIndiceMultiplasPastas(t *testing.T) {
	a, b := t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(b, "save"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	roots := []Root{{"<base>", a}, {"<base>/nested", b}}
	idx, err := ScanRoots(roots)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := idx.Files["base/nested/save"]; !ok {
		t.Fatal(idx)
	}
	got, ok := Locate(roots, "base/nested/save")
	if !ok || got != filepath.Join(b, "save") {
		t.Fatal(got, ok)
	}
}
