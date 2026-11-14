package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDescobertaComparaRaizesERuido(t *testing.T) {
	dir := t.TempDir()
	home := filepath.Join(dir, "home")
	env := Env{"home": home, "winDocuments": filepath.Join(home, "Documents"), "winAppData": filepath.Join(home, "AppData", "Roaming"), "winLocalAppData": filepath.Join(home, "AppData", "Local"), "winPublic": filepath.Join(dir, "public"), "base": filepath.Join(dir, "jogo")}
	write := func(rel, body string) {
		t.Helper()
		p := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write("home/Documents/MeuJogo/antigo.dat", "antigo")
	before := CaptureAt(env, 1)
	write("home/Documents/MeuJogo/antigo.dat", "alterado")
	write("home/Documents/MeuJogo/novo.dat", "novo")
	write("home/Documents/My Games/Um Jogo/save.dat", "s")
	write("home/AppData/LocalLow/Dev/Jogo/save.dat", "s")
	write("home/AppData/Roaming/OutroJogo/logo.png", "x")
	write("jogo/cache/a.bin", "x")
	write("jogo/crash.dmp", "x")
	write("jogo/saves/slot.dat", "s")
	out := CompareCaptures(before, CaptureAt(env, 2))
	want := map[string]uint64{"<winDocuments>/MeuJogo": 2, "<winDocuments>/My Games/Um Jogo": 1, "<home>/AppData/LocalLow/Dev/Jogo": 1, "<winAppData>/OutroJogo": 1, "<base>/saves": 1}
	if len(out) != len(want) {
		t.Fatal(out)
	}
	if out[0].Logical != "<winDocuments>/MeuJogo" {
		t.Fatal(out)
	}
	for _, c := range out {
		if want[c.Logical] != c.Added {
			t.Fatal(c)
		}
	}
	if err := os.Remove(filepath.Join(home, "Documents", "MeuJogo", "antigo.dat")); err != nil {
		t.Fatal(err)
	}
	out = CompareCaptures(before, CaptureAt(env, 3))
	found := false
	for _, c := range out {
		if c.Logical == "<winDocuments>/MeuJogo" {
			found = true
			if c.Removed != 1 || c.Added != 1 {
				t.Fatal(c)
			}
		}
	}
	if !found {
		t.Fatal(out)
	}
}
func TestDescobertaArquivoNaRaiz(t *testing.T) {
	before := Capture{Files: map[string]FileStat{}}
	after := Capture{Files: map[string]FileStat{"<winDocuments>/sessao.sav": {Size: 1}}}
	out := CompareCaptures(before, after)
	if len(out) != 1 || out[0].Logical != "<winDocuments>" {
		t.Fatal(out)
	}
}
