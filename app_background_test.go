package main

import (
	"strings"
	"testing"

	"sync-game-save/internal/library"
)

func capturarNotificacoes(t *testing.T) *[]string {
	t.Helper()
	var enviados []string
	antes := enviarNotificacao
	enviarNotificacao = func(titulo, texto string) { enviados = append(enviados, titulo+": "+texto) }
	t.Cleanup(func() { enviarNotificacao = antes })
	return &enviados
}

func TestAvisosSoQuandoOUsuarioPrecisaSaber(t *testing.T) {
	enviados := capturarNotificacoes(t)
	sim, nao := true, false
	aviso := "Jogo fechado. O save não foi sincronizado."
	if avisarResultadoDoJogo(ResultadoJogar{Kind: "finished", Synced: &sim}, nil) {
		t.Fatal("tudo certo não deve avisar")
	}
	if avisarResultadoDoJogo(ResultadoJogar{Kind: "locked"}, nil) {
		t.Fatal("trava é tratada pela interface")
	}
	if !avisarResultadoDoJogo(ResultadoJogar{Kind: "finished", Synced: &nao, Notice: &aviso}, nil) {
		t.Fatal("sem sync deve avisar")
	}
	if !avisarResultadoDoJogo(ResultadoJogar{Kind: "conflict"}, nil) || !avisarResultadoDoJogo(ResultadoJogar{}, errString("falhou")) {
		t.Fatal("conflito e erro devem avisar")
	}
	if len(*enviados) != 3 || !strings.Contains((*enviados)[0], aviso) || !strings.Contains((*enviados)[1], "Abra o app") || !strings.HasSuffix((*enviados)[2], "falhou") {
		t.Fatal(*enviados)
	}
}

type errString string

func (e errString) Error() string { return string(e) }

func TestAtualizacaoEmSegundoPlanoSoComSteamConfigurada(t *testing.T) {
	t.Setenv("SGS_DATA_DIR", t.TempDir())
	t.Setenv("SGS_STEAM_DIR", t.TempDir())
	app := App{}
	id, _ := app.AdicionarJogo(library.NewGame{Name: "Jogo", Exe: "x.exe"})
	lib, _ := abrirBiblioteca()
	lib.SetSetting("steam_appid:"+itoa(id), "241600")
	lib.Close()
	rede := &steamFalsa{perfil: perfilXML}
	if err := atualizarConquistasEmSegundoPlano(rede); err != nil || rede.chamadas != 0 {
		t.Fatal("sem credenciais não deve usar a rede", err, rede.chamadas)
	}
	if err := app.DefinirCredenciaisSteam("chave", "76561198000000001"); err != nil {
		t.Fatal(err)
	}
	if err := atualizarConquistasEmSegundoPlano(rede); err != nil || rede.chamadas == 0 {
		t.Fatal(err, rede.chamadas)
	}
	// Segunda rodada dentro da validade usa o cache.
	antes := rede.chamadas
	if err := atualizarConquistasEmSegundoPlano(rede); err != nil || rede.chamadas != antes {
		t.Fatal("o cache de 10 minutos não foi respeitado", rede.chamadas, antes)
	}
	// Falha de rede não interrompe nem vira erro.
	rede.falha = true
	if err := atualizarConquistasEmSegundoPlano(rede); err != nil {
		t.Fatal(err)
	}
}
