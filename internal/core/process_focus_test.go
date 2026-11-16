package core

import "testing"

func TestPidsNaPastaSoPegaProgramasDaPastaDoJogo(t *testing.T) {
	procs := []ProcessInfo{
		{PID: 1, Executable: "C:/Jogos/Celeste/Celeste.exe"},
		{PID: 2, Executable: "C:/Jogos/Celeste/bin/helper.exe"},
		{PID: 3, Executable: "C:/Jogos/Celeste2/Outro.exe"},
		{PID: 4, Executable: ""},
	}
	got := pidsNaPasta(procs, "C:/Jogos/Celeste")
	if len(got) != 2 || !got[1] || !got[2] {
		t.Fatalf("pids = %v", got)
	}
	if len(pidsNaPasta(procs, "")) != 0 {
		t.Fatal("pasta vazia não deve casar com nada")
	}
}

func TestFocusGameSemJogoAbertoNaoFazNada(t *testing.T) {
	ok, err := FocusGame("C:/nao/existe/jogo.exe")
	if ok || err != nil {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
}
