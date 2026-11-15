package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync-game-save/internal/library"
	"testing"
)

func TestAppListaEAdicionaJogo(t *testing.T) {
	t.Setenv("SGS_DATA_DIR", t.TempDir())
	app := App{}
	games, err := app.ListarJogos()
	if err != nil || len(games) != 0 {
		t.Fatal(games, err)
	}
	id, err := app.AdicionarJogo(library.NewGame{Name: " Jogo ", Exe: "jogo.exe", SavePaths: []string{"<base>/save"}})
	if err != nil {
		t.Fatal(id, err)
	}
	games, err = app.ListarJogos()
	if err != nil || len(games) != 1 || games[0].ID != id || games[0].Name != "Jogo" || len(games[0].SavePaths) != 1 {
		t.Fatal(games, err)
	}
}

func TestEditarJogoInvalidaBaseSomenteAoMudarSave(t *testing.T) {
	data := t.TempDir()
	t.Setenv("SGS_DATA_DIR", data)
	app := App{}
	old := library.NewGame{Name: "Jogo", Exe: "jogo.exe", SavePaths: []string{"<base>/save"}}
	id, err := app.AdicionarJogo(old)
	if err != nil {
		t.Fatal(err)
	}
	lib, err := library.Open(filepath.Join(data, "library.db"))
	if err != nil {
		t.Fatal(err)
	}
	if err := lib.RecordSync(id, "abc", 42); err != nil {
		t.Fatal(err)
	}
	lib.Close()
	state := filepath.Join(data, "state", fmt.Sprint(id))
	if err := os.MkdirAll(state, 0755); err != nil {
		t.Fatal(err)
	}
	base := filepath.Join(state, "base.json")
	if err := os.WriteFile(base, []byte(`{"digest":"abc"}`), 0644); err != nil {
		t.Fatal(err)
	}
	renamed := old
	renamed.Name = "Renomeado"
	if err := app.EditarJogo(id, renamed); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(base); err != nil {
		t.Fatal("base removida sem mudança do save", err)
	}
	changed := renamed
	changed.SavePaths = []string{"<base>/novo"}
	if err := app.EditarJogo(id, changed); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(base); !os.IsNotExist(err) {
		t.Fatal("base não invalidada", err)
	}
	lib, err = library.Open(filepath.Join(data, "library.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer lib.Close()
	game, err := lib.Get(id)
	if err != nil || game.BaseDigest != nil || game.LastSyncAt != nil || game.SavePaths[0] != "<base>/novo" {
		t.Fatal(game, err)
	}
}

func TestEdicaoInvalidaPreservaBaseERegistro(t *testing.T) {
	data := t.TempDir()
	t.Setenv("SGS_DATA_DIR", data)
	app := App{}
	old := library.NewGame{Name: "Primeiro", Exe: "a.exe", SavePaths: []string{"<base>/save"}}
	id, err := app.AdicionarJogo(old)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.AdicionarJogo(library.NewGame{Name: "Segundo", Exe: "b.exe"}); err != nil {
		t.Fatal(err)
	}
	state := filepath.Join(data, "state", fmt.Sprint(id))
	if err := os.MkdirAll(state, 0755); err != nil {
		t.Fatal(err)
	}
	base := filepath.Join(state, "base.json")
	if err := os.WriteFile(base, []byte("base anterior"), 0644); err != nil {
		t.Fatal(err)
	}
	changed := old
	changed.Name = "Segundo"
	changed.SavePaths = []string{"<base>/novo"}
	if err := app.EditarJogo(id, changed); !errors.Is(err, library.ErrDuplicateName) {
		t.Fatal(err)
	}
	content, err := os.ReadFile(base)
	if err != nil || string(content) != "base anterior" {
		t.Fatal(string(content), err)
	}
	games, err := app.ListarJogos()
	if err != nil || games[0].Name != "Primeiro" || games[0].SavePaths[0] != "<base>/save" {
		t.Fatal(games, err)
	}
}

func TestRemoverJogoPreservaSavesELimpaCache(t *testing.T) {
	data := t.TempDir()
	t.Setenv("SGS_DATA_DIR", data)
	app := App{}
	save := filepath.Join(t.TempDir(), "slot.sav")
	if err := os.WriteFile(save, []byte("progresso"), 0644); err != nil {
		t.Fatal(err)
	}
	id, err := app.AdicionarJogo(library.NewGame{Name: "Jogo", Exe: "a.exe", SavePaths: []string{save}})
	if err != nil {
		t.Fatal(err)
	}
	art := filepath.Join(data, "art")
	if err := os.MkdirAll(art, 0755); err != nil {
		t.Fatal(err)
	}
	cover := filepath.Join(art, fmt.Sprintf("%d.jpg", id))
	if err := os.WriteFile(cover, []byte("arte"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := app.RemoverJogo(id); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(save)
	if err != nil || string(content) != "progresso" {
		t.Fatal(string(content), err)
	}
	if _, err := os.Stat(cover); !os.IsNotExist(err) {
		t.Fatal("cache não removido", err)
	}
}
