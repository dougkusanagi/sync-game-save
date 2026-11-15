package main

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"

	"github.com/wailsapp/wails/v3/pkg/application"

	"sync-game-save/internal/catalog"
	"sync-game-save/internal/core"
)

var errSemCatalogo = errors.New("O catálogo de jogos ainda não foi baixado. Atualize o catálogo em Configurações.")

func abrirCatalogo() (*catalog.Catalog, error) {
	cat, err := catalog.Open(filepath.Join(pastaDados(), "catalog.db"))
	if errors.Is(err, os.ErrNotExist) {
		return nil, errSemCatalogo
	}
	return cat, err
}

// VarrerPasta procura jogos nas subpastas de uma pasta, até profundidade níveis.
func (App) VarrerPasta(pasta string, profundidade uint32) (catalog.ScanResult, error) {
	cat, err := abrirCatalogo()
	if err != nil {
		return catalog.ScanResult{}, err
	}
	defer cat.Close()
	return cat.ScanFolder(pasta, int(profundidade), core.SystemEnv(""), runtime.GOOS == "windows")
}

// EscolherCandidatoCatalogo aplica uma escolha explícita do catálogo à pasta selecionada.
func (App) EscolherCandidatoCatalogo(pasta string, id int64) (catalog.Recognition, error) {
	cat, err := abrirCatalogo()
	if err != nil {
		return catalog.Recognition{}, err
	}
	defer cat.Close()
	rec, err := cat.RecognizeChoice(pasta, core.SystemEnv(pasta), runtime.GOOS == "windows", id)
	if err != nil {
		return catalog.Recognition{}, err
	}
	if rec == nil {
		return catalog.Recognition{}, errors.New("O jogo escolhido não está mais no catálogo. Procure novamente.")
	}
	return *rec, nil
}

// saindo vira true quando o usuário pede para sair de verdade (menu da bandeja ou SairApp);
// fechar a janela só a esconde.
var saindo bool

func jogoAbertoOuSincronizando() bool {
	jogosAbertosMu.Lock()
	defer jogosAbertosMu.Unlock()
	return len(jogosAbertos) > 0
}

// SairApp fecha o app, só se nenhum jogo estiver aberto ou sincronizando (senão o save não
// seria enviado).
func (App) SairApp() error {
	if jogoAbertoOuSincronizando() {
		return errors.New("Há um jogo aberto ou sincronizando. Feche o jogo e espere o save ser enviado antes de sair.")
	}
	saindo = true
	if app := application.Get(); app != nil {
		app.Quit()
	}
	return nil
}
