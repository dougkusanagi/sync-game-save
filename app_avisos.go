package main

import (
	"sync"
	"time"

	"sync-game-save/internal/core"
)

const maxAvisos = 20

// Aviso é uma notificação enviada neste uso do app, mostrada na central rápida.
type Aviso struct {
	Quando uint64 `json:"quando"`
	Texto  string `json:"texto"`
}

var (
	avisosMu  sync.Mutex
	avisosRec []Aviso
)

// avisar envia a notificação do sistema e guarda o texto, para o usuário reler na central
// caso tenha perdido o toast. O histórico existe só enquanto o app está aberto.
func avisar(titulo, texto string) {
	avisosMu.Lock()
	avisosRec = append([]Aviso{{Quando: uint64(time.Now().Unix()), Texto: texto}}, avisosRec...)
	if len(avisosRec) > maxAvisos {
		avisosRec = avisosRec[:maxAvisos]
	}
	avisosMu.Unlock()
	enviarNotificacao(titulo, texto)
}

// Avisos devolve os avisos recentes, do mais novo ao mais antigo.
func (App) Avisos() []Aviso {
	avisosMu.Lock()
	defer avisosMu.Unlock()
	return append([]Aviso{}, avisosRec...)
}

// VoltarAoJogo traz a janela do jogo aberto para a frente. Devolve false se o jogo não está
// aberto ou se a janela não foi encontrada.
func (App) VoltarAoJogo(id int64) (bool, error) {
	lib, err := abrirBiblioteca()
	if err != nil {
		return false, err
	}
	defer lib.Close()
	game, err := lib.Get(id)
	if err != nil {
		return false, err
	}
	ok, err := core.FocusGame(game.Exe)
	if err == core.ErrFocoIndisponivel {
		return false, nil
	}
	return ok, err
}
