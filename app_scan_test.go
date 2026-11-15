package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"sync-game-save/internal/library"
)

func TestVarrerPastaEEscolhaExigemCatalogo(t *testing.T) {
	t.Setenv("SGS_DATA_DIR", t.TempDir())
	app := App{}
	if _, err := app.VarrerPasta(t.TempDir(), 2); err == nil || !strings.Contains(err.Error(), "catálogo") {
		t.Fatal(err)
	}
	if _, err := app.EscolherCandidatoCatalogo(t.TempDir(), 1); err == nil || !strings.Contains(err.Error(), "catálogo") {
		t.Fatal(err)
	}
}

func TestVarrerPastaComCatalogoGeradoPeloRust(t *testing.T) {
	dados := t.TempDir()
	t.Setenv("SGS_DATA_DIR", dados)
	bruto, err := os.ReadFile(filepath.Join("internal", "testdata", "rust", "catalog.db"))
	if err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(dados, "catalog.db"), bruto, 0644)
	biblioteca := t.TempDir()
	for _, f := range []string{"Aggelos/Aggelos.exe", "Desconhecido/x.exe"} {
		p := filepath.Join(biblioteca, filepath.FromSlash(f))
		os.MkdirAll(filepath.Dir(p), 0755)
		os.WriteFile(p, []byte("exe"), 0644)
	}
	app := App{}
	r, err := app.VarrerPasta(biblioteca, 1)
	if err != nil || len(r.Recognized) != 1 || len(r.Unknown) != 1 || r.Recognized[0].Recognition.Name != "Aggelos" {
		t.Fatal(r, err)
	}
	rec, err := app.EscolherCandidatoCatalogo(filepath.Join(biblioteca, "Aggelos"), *r.Recognized[0].Recognition.CatalogID)
	if err != nil || rec.Confidence != "High" || rec.Name != "Aggelos" {
		t.Fatal(rec, err)
	}
	if _, err := app.EscolherCandidatoCatalogo(filepath.Join(biblioteca, "Aggelos"), 9999); err == nil {
		t.Fatal("jogo fora do catálogo aceito")
	}
}

func TestSairSoSemJogoAbertoOuSincronizando(t *testing.T) {
	defer func() { saindo = false }()
	fim, err := iniciarOperacao(77)
	if err != nil {
		t.Fatal(err)
	}
	if err := (App{}).SairApp(); err == nil || saindo {
		t.Fatal("saiu com um jogo aberto", err)
	}
	fim()
	if err := (App{}).SairApp(); err != nil || !saindo {
		t.Fatal(err, saindo)
	}
}

func TestPrimeiroBackupDoApp(t *testing.T) {
	c := novoCenario(t, "")
	app := App{}
	r, err := app.PrimeiroBackup(c.id)
	if err != nil || r.Kind != "nothing_to_backup" {
		t.Fatal(r, err)
	}
	os.WriteFile(filepath.Join(c.save, "slot.sav"), []byte("primeiro"), 0644)
	if r, err = app.PrimeiroBackup(c.id); err != nil || r.Kind != "created" {
		t.Fatal(r, err)
	}
	lib, _ := abrirBiblioteca()
	game, _ := lib.Get(c.id)
	lib.Close()
	if game.BaseDigest == nil || game.LastSyncAt == nil {
		t.Fatal("sync não registrado", game)
	}
	if r, err = app.PrimeiroBackup(c.id); err != nil || r.Kind != "already_synced" {
		t.Fatal(r, err)
	}
	if v, _ := app.Versoes(c.id); len(v) != 1 {
		t.Fatal(v)
	}
	// Jogo em operação: recusa sem erro.
	fim, _ := iniciarOperacao(c.id)
	if r, err = app.PrimeiroBackup(c.id); err != nil || r.Kind != "in_use" {
		t.Fatal(r, err)
	}
	fim()
}

func TestPrimeiroBackupNaoTrocaVersaoDeOutroPC(t *testing.T) {
	c := novoCenario(t, "do-pc-a")
	app := App{}
	if r, err := app.PrimeiroBackup(c.id); err != nil || r.Kind != "created" {
		t.Fatal(r, err)
	}
	// Outro PC (outro banco local) com save diferente.
	dadosB := t.TempDir()
	t.Setenv("SGS_DATA_DIR", dadosB)
	pastaB := t.TempDir()
	os.WriteFile(filepath.Join(pastaB, "slot.sav"), []byte("do-pc-b"), 0644)
	idB, err := app.AdicionarJogo(library.NewGame{Name: "Jogo", Exe: "x.exe", SavePaths: []string{pastaB}})
	if err != nil {
		t.Fatal(err)
	}
	lib, _ := abrirBiblioteca()
	lib.SetSetting("sync_dir", c.nuvem)
	lib.SetSetting("device", "PC-B")
	lib.Close()
	if r, err := app.PrimeiroBackup(idB); err != nil || r.Kind != "needs_decision" {
		t.Fatal(r, err)
	}
	if got, _ := os.ReadFile(filepath.Join(pastaB, "slot.sav")); string(got) != "do-pc-b" {
		t.Fatal("o save local foi trocado:", string(got))
	}
}
