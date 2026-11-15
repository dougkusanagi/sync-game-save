package main

import (
	"runtime/debug"
	"strings"
)

// SobreApp tem o mesmo formato JSON da versão Rust.
type SobreApp struct {
	Nome   string `json:"nome"`
	Versao string `json:"versao"`
	Build  string `json:"build"`
}

// formatarBuild monta o número do build: atalho do commit e data do commit, como na versão Rust.
func formatarBuild(revisao, data string) string {
	commit := "local"
	if len(revisao) >= 7 {
		commit = revisao[:7]
	}
	if i := strings.IndexByte(data, 'T'); i == 10 {
		return commit + " (" + data[:10] + ")"
	}
	return commit
}

func numeroDoBuild() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "local"
	}
	var revisao, data string
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			revisao = s.Value
		case "vcs.time":
			data = s.Value
		}
	}
	return formatarBuild(revisao, data)
}

// Sobre devolve o nome, a versão e o número do build para a tela Sobre.
func (App) Sobre() SobreApp {
	return SobreApp{Nome: "Sync Game Save", Versao: versaoApp, Build: numeroDoBuild()}
}
