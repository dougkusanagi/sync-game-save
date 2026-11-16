package main

import (
	"testing"

	"sync-game-save/internal/library"
)

func TestAvisarGuardaOsMaisRecentesPrimeiroEComLimite(t *testing.T) {
	antes := enviarNotificacao
	enviados := 0
	enviarNotificacao = func(string, string) { enviados++ }
	avisosRec = nil
	t.Cleanup(func() { enviarNotificacao = antes; avisosRec = nil })
	for i := 0; i < maxAvisos+5; i++ {
		avisar("t", string(rune('a'+i%26)))
	}
	av := (App{}).Avisos()
	if len(av) != maxAvisos || enviados != maxAvisos+5 {
		t.Fatalf("len=%d enviados=%d", len(av), enviados)
	}
	if av[0].Texto != string(rune('a'+(maxAvisos+4)%26)) {
		t.Fatalf("o mais novo deve vir primeiro: %q", av[0].Texto)
	}
}

func TestVoltarAoJogoSemSessaoDevolveFalso(t *testing.T) {
	t.Setenv("SGS_DATA_DIR", t.TempDir())
	id, err := (App{}).AdicionarJogo(library.NewGame{Name: "Jogo", Exe: "x.exe"})
	if err != nil {
		t.Fatal(err)
	}
	ok, err := (App{}).VoltarAoJogo(id)
	if ok || err != nil {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
}
