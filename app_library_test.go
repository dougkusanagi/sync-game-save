package main

import (
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
