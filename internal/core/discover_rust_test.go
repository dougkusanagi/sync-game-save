package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// A captura foi gerada pela versão Rust (exemplo gerar_fixtures), para o exe C:/Jogos/aa.exe.
func TestLeCapturaDeDescobertaGeradaPeloRust(t *testing.T) {
	const chave = "0b063afc4f0cee22e299e92e823c7c9f9b4e2d05acc0ce489936b01158fa6004"
	if DiscoveryKey("C:/Jogos/aa.exe") != chave {
		t.Fatal("a chave da captura difere da versão Rust:", DiscoveryKey("C:/Jogos/aa.exe"))
	}
	bruto, err := os.ReadFile(filepath.Join("..", "testdata", "rust", "captura-"+chave+".json"))
	if err != nil {
		t.Fatal(err)
	}
	var antes Capture
	if err := json.Unmarshal(bruto, &antes); err != nil {
		t.Fatal(err)
	}
	f, ok := antes.Files["<base>/Saves/slot.sav"]
	if antes.TakenAt != 1_700_003_000 || !ok || f.Size != 5 || f.Mtime == 0 {
		t.Fatal(antes)
	}
	// O mesmo arquivo, visto agora pelo Go, tem a mesma unidade de tempo (nanossegundos).
	base := t.TempDir()
	os.MkdirAll(filepath.Join(base, "Saves"), 0755)
	os.WriteFile(filepath.Join(base, "Saves", "slot.sav"), []byte("12345"), 0644)
	agora := CaptureAt(Env{"base": base}, 1)
	g := agora.Files["<base>/Saves/slot.sav"]
	if g.Size != 5 || g.Mtime/f.Mtime > 2 || f.Mtime/g.Mtime > 2 {
		t.Fatal("unidade do mtime diferente:", g.Mtime, f.Mtime)
	}
	// Uma captura do Rust, comparada com o estado de hoje do Go, lista a pasta que mudou.
	c := CompareCaptures(antes, agora)
	if len(c) != 1 || c[0].Logical != "<base>/Saves" || c[0].Added != 1 {
		t.Fatal(c)
	}
}
