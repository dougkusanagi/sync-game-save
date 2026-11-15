//go:build windows && sgs_e2e

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// A porta de depuração existe apenas em builds feitos para testes E2E.
func opcoesWindowsTeste() application.WindowsOptions {
	port, err := strconv.Atoi(os.Getenv("SGS_E2E_PORT"))
	if err != nil || port < 1024 || port > 65535 {
		return application.WindowsOptions{}
	}
	return application.WindowsOptions{
		AdditionalBrowserArgs: []string{fmt.Sprintf("--remote-debugging-port=%d", port)},
		WebviewUserDataPath:   filepath.Join(pastaDados(), "webview-e2e"),
	}
}

func foraDaTela() bool {
	return os.Getenv("SGS_E2E_VISIVEL") == "" && opcoesWindowsTeste().AdditionalBrowserArgs != nil
}

// No E2E a janela nasce fora da área visível: o WebView continua renderizando para as capturas
// e o teste o controla pela porta de depuração, sem aparecer para quem usa o PC.
func posicaoJanelaTeste() int {
	if foraDaTela() {
		return -32000
	}
	return 0
}

func posicaoInicialTeste() application.WindowStartPosition {
	if foraDaTela() {
		return application.WindowXY
	}
	return application.WindowCentered
}
