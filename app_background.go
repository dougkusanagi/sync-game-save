package main

import (
	"time"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v3/pkg/services/notifications"

	"sync-game-save/internal/achievements"
)

var notificador = notifications.New()

// enviarNotificacao mostra um aviso do sistema. Os testes trocam por uma função própria.
var enviarNotificacao = func(titulo, texto string) {
	_ = notificador.SendNotification(notifications.NotificationOptions{ID: uuid.NewString(), Title: titulo, Body: texto})
}

// avisarResultadoDoJogo só avisa o que o usuário precisa saber: quando tudo dá certo, não há
// notificação. Devolve se enviou alguma.
func avisarResultadoDoJogo(r ResultadoJogar, err error) bool {
	titulo := "Sync Game Save"
	switch {
	case err != nil:
		avisar(titulo, err.Error())
	case r.Kind == "conflict":
		avisar(titulo, "O save deste PC e o da pasta de sincronização mudaram. Abra o app para escolher qual manter.")
	case r.Kind == "finished" && r.Synced != nil && !*r.Synced && r.Notice != nil:
		avisar(titulo, *r.Notice)
	default:
		return false
	}
	return true
}

// atualizarConquistasEmSegundoPlano atualiza o cache de conquistas dos jogos Steam da biblioteca.
// Falhas individuais são ignoradas para que um perfil privado ou uma falha de rede não
// interrompa as atualizações dos demais jogos.
func atualizarConquistasEmSegundoPlano(http achievements.HTTP) error {
	lib, err := abrirBiblioteca()
	if err != nil {
		return err
	}
	chave, err := valorConfigurado(lib, "steam_api_key")
	if err != nil {
		lib.Close()
		return err
	}
	steamID, err := valorConfigurado(lib, "steam_id")
	if err != nil {
		lib.Close()
		return err
	}
	if !chave || !steamID {
		lib.Close()
		return nil
	}
	jogos, err := lib.List()
	if err != nil {
		lib.Close()
		return err
	}
	var comSteam []int64
	for _, g := range jogos {
		if id, err := idSteamDoJogo(lib, g); err == nil && id != nil {
			comSteam = append(comSteam, g.ID)
		}
	}
	lib.Close()
	for _, id := range comSteam {
		_, _ = conquistasDoJogo(id, false, http, int64(agora()))
	}
	return nil
}

// iniciarTarefasEmSegundoPlano repete a atualização das conquistas a cada hora, com cache de
// 10 minutos; a rede fora do ar não afeta a interface.
func iniciarTarefasEmSegundoPlano() {
	go limparLogsAntigos(pastaDados(), time.Now())
	go func() {
		for {
			_ = atualizarConquistasEmSegundoPlano(achievements.NetHTTP{})
			time.Sleep(time.Hour)
		}
	}()
}
