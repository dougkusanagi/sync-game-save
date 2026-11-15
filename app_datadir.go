package main

import (
	"os"
	"path/filepath"
	"runtime"
)

const identificadorApp = "com.syncgamesave.app"

// pastaDadosDe devolve a pasta de dados do app, a mesma que a versão Tauri usava, para quem já tem
// uma biblioteca. No Windows e no macOS é a pasta de configuração do usuário (%APPDATA%,
// ~/Library/Application Support); no Linux, a pasta de dados (XDG_DATA_HOME ou ~/.local/share).
func pastaDadosDe(sistema, xdgData, home, config string) string {
	if sistema == "linux" {
		base := xdgData
		if base == "" && home != "" {
			base = filepath.Join(home, ".local", "share")
		}
		return filepath.Join(base, identificadorApp)
	}
	return filepath.Join(config, identificadorApp)
}

func pastaDados() string {
	if d := os.Getenv("SGS_DATA_DIR"); d != "" {
		return d
	}
	config, _ := os.UserConfigDir()
	home, _ := os.UserHomeDir()
	return pastaDadosDe(runtime.GOOS, os.Getenv("XDG_DATA_HOME"), home, config)
}
