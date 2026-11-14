package core

import (
	"fmt"
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

func TestEngineNuvemIncompletaPreservaSaveLocal(t *testing.T) {
	dir := t.TempDir()
	a := enginePC(dir, "A", 1000)
	enginePut(t, a, "versão completa")
	if _, err := a.BeforeLaunch(false); err != nil {
		t.Fatal(err)
	}
	if _, err := a.afterExit(nil, noSleep); err != nil {
		t.Fatal(err)
	}
	head, err := ReadHead(a.GameDir)
	if err != nil || head == nil {
		t.Fatal(head, err)
	}
	if err := os.WriteFile(filepath.Join(a.GameDir, "snapshots", head.Snapshot, "files", "s.sav"), []byte("incompleta"), 0644); err != nil {
		t.Fatal(err)
	}
	b := enginePC(dir, "B", 2000)
	enginePut(t, b, "save do B")
	result, err := b.BeforeLaunch(false)
	if err == nil && result.Status != SyncConflict {
		t.Fatal(result)
	}
	if got := engineGet(t, b); got != "save do B" {
		t.Fatalf("save local alterado: %q", got)
	}
	c := enginePC(dir, "C", 3000)
	if _, err := c.BeforeLaunch(false); err == nil {
		t.Fatal("restauração de snapshot incompleto aceita")
	}
	if _, err := os.Stat(filepath.Join(c.Roots[0].Path, "s.sav")); !os.IsNotExist(err) {
		t.Fatal("save foi criado", err)
	}
	lock, err := ReadLock(a.GameDir)
	if err != nil || (lock != nil && lock.Device != "A") {
		t.Fatal(lock, err)
	}
}

func TestEngineSaveAusenteNaoApagaNuvem(t *testing.T) {
	dir := t.TempDir()
	a := enginePC(dir, "A", 1000)
	enginePut(t, a, "original")
	if _, err := a.BeforeLaunch(false); err != nil {
		t.Fatal(err)
	}
	if _, err := a.afterExit(nil, noSleep); err != nil {
		t.Fatal(err)
	}
	before, err := ReadHead(a.GameDir)
	if err != nil || before == nil {
		t.Fatal(before, err)
	}
	if err := os.RemoveAll(a.Roots[0].Path); err != nil {
		t.Fatal(err)
	}
	a.Now = 2000
	if _, err := a.BeforeLaunch(false); err != nil {
		t.Fatal(err)
	}
	if created, err := a.afterExit(nil, noSleep); err != nil || created {
		t.Fatal(created, err)
	}
	after, err := ReadHead(a.GameDir)
	if err != nil || after == nil || *after != *before {
		t.Fatal(before, after, err)
	}
	data, err := os.ReadFile(filepath.Join(a.GameDir, "snapshots", before.Snapshot, "files", "s.sav"))
	if err != nil || string(data) != "original" {
		t.Fatal(string(data), err)
	}
}

func TestEngineArquivosInstaveisNaoAlteramNuvemNemLiberamTrava(t *testing.T) {
	a := enginePC(t.TempDir(), "A", 1000)
	enginePut(t, a, "antes")
	if _, err := a.BeforeLaunch(false); err != nil {
		t.Fatal(err)
	}
	before, err := ReadHead(a.GameDir)
	if err != nil || before == nil {
		t.Fatal(before, err)
	}
	ticks := 0
	created, err := a.afterExit(func() { ticks++; enginePut(t, a, fmt.Sprintf("mudança %d", ticks)) }, noSleep)
	if err == nil || created || ticks != 63 {
		t.Fatal(created, ticks, err)
	}
	after, err := ReadHead(a.GameDir)
	if err != nil || after == nil || *after != *before {
		t.Fatal(before, after, err)
	}
	lock, err := ReadLock(a.GameDir)
	if err != nil || lock == nil || lock.Device != "A" {
		t.Fatal(lock, err)
	}
}

func TestEngineConfirmaAberturaComTravaDeOutroPC(t *testing.T) {
	dir := t.TempDir()
	a := enginePC(dir, "A", 1000)
	enginePut(t, a, "save")
	if _, err := a.BeforeLaunch(false); err != nil {
		t.Fatal(err)
	}
	b := enginePC(dir, "B", 1100)
	result, err := b.BeforeLaunch(false)
	if err != nil || result.Status != LockedByOther || result.Device != "A" {
		t.Fatal(result, err)
	}
	result, err = b.BeforeLaunch(true)
	if err != nil || result.Status != RestoredFromCloud || engineGet(t, b) != "save" {
		t.Fatal(result, err)
	}
	lock, err := ReadLock(a.GameDir)
	if err != nil || lock == nil || lock.Device != "B" {
		t.Fatal(lock, err)
	}
}

func TestEngineConflitoUsaLocalSemPerderVersaoDaNuvem(t *testing.T) {
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
	enginePut(t, a, "local")
	enginePut(t, b, "nuvem")
	if _, err := b.afterExit(nil, noSleep); err != nil {
		t.Fatal(err)
	}
	cloud, err := ReadHead(a.GameDir)
	if err != nil || cloud == nil {
		t.Fatal(cloud, err)
	}
	a.Now = 4000
	result, err := a.BeforeLaunch(false)
	if err != nil || result.Status != SyncConflict {
		t.Fatal(result, err)
	}
	if err := a.Resolve(UseLocal); err != nil {
		t.Fatal(err)
	}
	newHead, err := ReadHead(a.GameDir)
	if err != nil || newHead == nil || newHead.Snapshot == cloud.Snapshot {
		t.Fatal(newHead, err)
	}
	for _, item := range []struct{ id, want string }{{cloud.Snapshot, "nuvem"}, {newHead.Snapshot, "local"}} {
		data, err := os.ReadFile(filepath.Join(a.GameDir, "snapshots", item.id, "files", "s.sav"))
		if err != nil || string(data) != item.want {
			t.Fatal(item.id, string(data), err)
		}
	}
	lock, err := ReadLock(a.GameDir)
	if err != nil || lock == nil || lock.Device != "A" {
		t.Fatal(lock, err)
	}
}

func TestEngineJogoHelper(t *testing.T) {
	if os.Getenv("SGS_ENGINE_GAME_HELPER") != "1" {
		return
	}
	for i, arg := range os.Args {
		if arg == "--" && i+1 < len(os.Args) {
			if err := os.WriteFile(os.Args[i+1], []byte("save após o jogo"), 0644); err != nil {
				os.Exit(2)
			}
			time.Sleep(500 * time.Millisecond)
			os.Exit(7)
		}
	}
	os.Exit(3)
}

func TestEngineProcessoComErroSincronizaDepoisDeEncerrar(t *testing.T) {
	t.Setenv("SGS_ENGINE_GAME_HELPER", "1")
	a := enginePC(t.TempDir(), "A", 1000)
	enginePut(t, a, "save anterior")
	if _, err := a.BeforeLaunch(false); err != nil {
		t.Fatal(err)
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	// A cópia isola o rastreamento por pasta dos demais processos de teste.
	binary, err := os.ReadFile(exe)
	if err != nil {
		t.Fatal(err)
	}
	game := filepath.Join(t.TempDir(), "jogo"+filepath.Ext(exe))
	if err := os.WriteFile(game, binary, 0755); err != nil {
		t.Fatal(err)
	}
	ticks := 0
	var heartbeatErr error
	code, err := RunProcess(game, []string{"-test.run=^TestEngineJogoHelper$", "--", filepath.Join(a.Roots[0].Path, "s.sav")}, "", 100*time.Millisecond, 100*time.Millisecond, func() {
		ticks++
		if heartbeatErr == nil {
			heartbeatErr = HeartbeatLock(a.GameDir, a.Device, 1100)
		}
	})
	if err != nil || code != 7 || ticks == 0 || heartbeatErr != nil {
		t.Fatal(code, ticks, heartbeatErr, err)
	}
	lock, err := ReadLock(a.GameDir)
	if err != nil || lock == nil || lock.Heartbeat != 1100 {
		t.Fatal(lock, err)
	}
	created, err := a.afterExit(nil, noSleep)
	if err != nil || !created {
		t.Fatal(created, err)
	}
	head, err := ReadHead(a.GameDir)
	if err != nil || head == nil {
		t.Fatal(head, err)
	}
	data, err := os.ReadFile(filepath.Join(a.GameDir, "snapshots", head.Snapshot, "files", "s.sav"))
	if err != nil || string(data) != "save após o jogo" {
		t.Fatal(string(data), err)
	}
	lock, err = ReadLock(a.GameDir)
	if err != nil || lock != nil {
		t.Fatal(lock, err)
	}
}
