package main

import (
	"context"
	_ "embed"
	"errors"
	"strings"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/updater"
	"github.com/wailsapp/wails/v3/pkg/updater/providers/endpoint"
)

// chavePublicaAtualizador é a chave que confere a assinatura de cada atualização. A chave
// privada fica fora do repositório (ver MEMORY.md). Sem a chave, o atualizador fica desligado.
//
//go:embed build/updater-public.pem
var chavePublicaAtualizador []byte

// urlManifesto aponta para o manifesto do último release (nome diferente do latest.json do Tauri,
// para as duas gerações conviverem durante a transição).
const urlManifesto = "https://github.com/dougkusanagi/sync-game-save/releases/latest/download/latest-wails.json"

// Atualizacao tem o mesmo formato que o frontend já usa.
type Atualizacao struct {
	Versao string  `json:"versao"`
	Atual  string  `json:"atual"`
	Notas  *string `json:"notas"`
}

type atualizador interface {
	Check(ctx context.Context) (*updater.Release, error)
	DownloadAndInstall(ctx context.Context) error
	Restart(ctx context.Context) error
}

// atualizadorEmUso é preenchido ao iniciar o app; os testes trocam por um falso.
var atualizadorEmUso atualizador

var errAtualizacaoIndisponivel = errors.New("A atualização não está disponível nesta instalação.")

// iniciarAtualizador liga o atualizador do Wails ao manifesto assinado. A janela própria do
// atualizador fica desligada: a interface do app mostra o andamento.
func iniciarAtualizador(app *application.App) {
	if len(strings.TrimSpace(string(chavePublicaAtualizador))) == 0 {
		return
	}
	prov, err := endpoint.New(endpoint.Config{URL: urlManifesto})
	if err != nil {
		return
	}
	err = app.Updater.Init(updater.Config{
		CurrentVersion: versaoApp,
		Providers:      []updater.Provider{prov},
		PublicKey:      chavePublicaAtualizador,
		Window:         updater.WindowNone,
	})
	if err == nil {
		atualizadorEmUso = app.Updater
	}
}

// VerificarAtualizacao procura uma versão mais nova. Sem atualização, devolve nada.
func (App) VerificarAtualizacao() (*Atualizacao, error) {
	return verificarAtualizacao(atualizadorEmUso)
}

func verificarAtualizacao(u atualizador) (*Atualizacao, error) {
	if u == nil {
		return nil, errAtualizacaoIndisponivel
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rel, err := u.Check(ctx)
	if err != nil {
		return nil, errors.New("Não consegui procurar atualizações agora. Confira a internet e tente de novo.")
	}
	if rel == nil {
		return nil, nil
	}
	var notas *string
	if n := strings.TrimSpace(rel.Notes); n != "" {
		notas = &n
	}
	return &Atualizacao{Versao: rel.Version, Atual: versaoApp, Notas: notas}, nil
}

// InstalarAtualizacao baixa, confere a assinatura, troca o executável e reabre o app. Só volta
// com erro: se tudo der certo, o app reinicia. Recusa enquanto um jogo estiver aberto ou
// sincronizando, para não interromper o envio do save.
func (App) InstalarAtualizacao() error {
	return instalarAtualizacao(atualizadorEmUso, func() { saindo = true })
}

func instalarAtualizacao(u atualizador, sair func()) error {
	if u == nil {
		return errAtualizacaoIndisponivel
	}
	if jogoAbertoOuSincronizando() {
		return errors.New("Há um jogo aberto ou sincronizando. Feche o jogo e espere o save ser enviado antes de atualizar.")
	}
	ctx := context.Background()
	rel, err := u.Check(ctx)
	if err != nil {
		return errors.New("Não consegui procurar atualizações agora. Confira a internet e tente de novo.")
	}
	if rel == nil {
		return errors.New("Nenhuma atualização disponível.")
	}
	if err := u.DownloadAndInstall(ctx); err != nil {
		return errors.New("Não consegui instalar a atualização. A versão atual continua funcionando; tente de novo mais tarde.")
	}
	// Fechar a janela só a esconde; o reinício precisa sair de verdade.
	sair()
	return u.Restart(ctx)
}
