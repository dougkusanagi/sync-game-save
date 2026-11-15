package library

import (
	"os"
	"path/filepath"
	"testing"
)

// internal/testdata/rust/library.db foi gerado pela versão Rust (exemplo gerar_fixtures).
func copiarFixture(t *testing.T, nome string) string {
	t.Helper()
	dados, err := os.ReadFile(filepath.Join("..", "testdata", "rust", nome))
	if err != nil {
		t.Fatal(err)
	}
	destino := filepath.Join(t.TempDir(), nome)
	if err := os.WriteFile(destino, dados, 0644); err != nil {
		t.Fatal(err)
	}
	return destino
}

func TestLeBibliotecaGeradaPeloRust(t *testing.T) {
	lib, err := Open(copiarFixture(t, "library.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer lib.Close()
	if v, err := lib.SchemaVersion(); err != nil || v != 3 {
		t.Fatal(v, err)
	}
	jogos, err := lib.List()
	if err != nil || len(jogos) != 2 {
		t.Fatal(jogos, err)
	}
	a, err := lib.Get(1)
	if err != nil {
		t.Fatal(err)
	}
	if a.Name != "Ação & Aventura" || a.Exe != "C:/Jogos/aa.exe" || a.InstallDir == nil || *a.InstallDir != "C:/Jogos" ||
		a.CatalogID == nil || *a.CatalogID != 7 || len(a.SavePaths) != 2 || a.SavePaths[0] != "<winDocuments>/AA" || a.SavePaths[1] != "D:/Saves/AA" {
		t.Fatal(a)
	}
	if a.BaseDigest == nil || *a.BaseDigest != "abc123" || a.LastSyncAt == nil || *a.LastSyncAt != 1_700_000_500 ||
		a.PlaytimeSecs != 3725 || a.LastPlayedAt == nil || *a.LastPlayedAt != 1_700_001_000 {
		t.Fatal(a)
	}
	b, _ := lib.Get(2)
	if b.InstallDir != nil || b.CatalogID != nil || len(b.SavePaths) != 0 || b.BaseDigest != nil {
		t.Fatal(b)
	}
	for chave, esperado := range map[string]string{"sync_dir": "C:/Nuvem", "device": "PC-RUST", "sync_ok_streak": "3", "theme": "claro"} {
		if v, err := lib.Setting(chave); err != nil || v == nil || *v != esperado {
			t.Fatal(chave, v, err)
		}
	}
	cache, err := lib.AchievementsCache(1)
	if err != nil || cache == nil || cache.FetchedAt != 1_700_002_000 || cache.JSON == "" {
		t.Fatal(cache, err)
	}
}

func TestEscritaEmBibliotecaDoRustContinuaValida(t *testing.T) {
	caminho := copiarFixture(t, "library.db")
	lib, err := Open(caminho)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := lib.AddGame(NewGame{Name: "ação & aventura", Exe: "x.exe"}, 1); err != ErrDuplicateName {
		t.Fatal("nome duplicado (sem diferenciar maiúsculas) aceito:", err)
	}
	id, err := lib.AddGame(NewGame{Name: "Novo", Exe: "n.exe", SavePaths: []string{"<home>/n"}}, 1_800_000_000)
	if err != nil || id != 3 {
		t.Fatal(id, err)
	}
	if err := lib.AddPlaytime(1, 100, 1_800_000_001); err != nil {
		t.Fatal(err)
	}
	if err := lib.RemoveGame(2); err != nil {
		t.Fatal(err)
	}
	lib.Close()
	lib, err = Open(caminho)
	if err != nil {
		t.Fatal(err)
	}
	defer lib.Close()
	jogos, err := lib.List()
	if err != nil || len(jogos) != 2 || jogos[1].Name != "Novo" || jogos[0].PlaytimeSecs != 3825 {
		t.Fatal(jogos, err)
	}
}
