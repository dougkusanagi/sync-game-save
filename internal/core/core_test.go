package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCaminhoLogicoEntrePCs(t *testing.T) {
	pc1 := Env{"home": filepath.Join("Users", "ana"), "winAppData": filepath.Join("Users", "ana", "AppData", "Roaming")}
	pc2 := Env{"winAppData": filepath.Join("Users", "beto", "AppData", "Roaming")}
	logical, ok := ToLogical(filepath.Join(pc1["winAppData"], "Jogo Ç", "saves"), pc1)
	if !ok || logical != "<winAppData>/Jogo Ç/saves" {
		t.Fatalf("caminho: %q", logical)
	}
	real, ok := Resolve(logical, pc2)
	if !ok || real != filepath.Join(pc2["winAppData"], "Jogo Ç", "saves") {
		t.Fatalf("resolução: %q", real)
	}
	for _, invalid := range []string{"<outro>/a", "<home>/../../Windows", `<home>\..\Windows`} {
		if _, ok := Resolve(invalid, pc1); ok {
			t.Errorf("aceitou %q", invalid)
		}
	}
}
func TestJournalDetectaOperacaoInterrompida(t *testing.T) {
	dir := t.TempDir()
	if err := JournalBegin(dir, "jogo", "restore", 5); err != nil {
		t.Fatal(err)
	}
	entries, err := JournalPending(dir)
	if err != nil || len(entries) != 1 || entries[0] != (JournalEntry{"restore", "jogo", 5}) {
		t.Fatalf("%v, %v", entries, err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "jogo.json"))
	if err != nil || string(data) != "{\n  \"op\": \"restore\",\n  \"game\": \"jogo\",\n  \"started\": 5\n}" {
		t.Fatalf("formato incompatível: %s (%v)", data, err)
	}
	if err := JournalEnd(dir, "jogo"); err != nil {
		t.Fatal(err)
	}
	entries, err = JournalPending(dir)
	if err != nil || len(entries) != 0 {
		t.Fatalf("%v, %v", entries, err)
	}
}
