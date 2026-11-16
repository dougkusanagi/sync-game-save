package core

import (
	"errors"
	"path/filepath"
	"strings"
)

// ErrFocoIndisponivel indica que este sistema não sabe trazer a janela do jogo para a frente.
var ErrFocoIndisponivel = errors.New("este sistema não permite trazer a janela do jogo para a frente")

// FocusGame traz para a frente a janela do jogo que o app está acompanhando. Devolve false se o
// jogo não está aberto ou se nenhuma janela visível dele foi encontrada. Só mexe em processos cujo
// programa está na pasta do executável do jogo.
func FocusGame(prog string) (bool, error) {
	if !ProcessRunning(prog) {
		return false, nil
	}
	procs, err := ListProcesses()
	if err != nil {
		return false, err
	}
	return focusPIDs(pidsNaPasta(procs, filepath.Dir(prog)))
}

func pidsNaPasta(procs []ProcessInfo, dir string) map[uint32]bool {
	base := processPath(dir)
	pids := map[uint32]bool{}
	if base == "" || base == "." {
		return pids
	}
	for _, p := range procs {
		if p.Executable != "" && strings.HasPrefix(processPath(p.Executable), base+"/") {
			pids[p.PID] = true
		}
	}
	return pids
}
