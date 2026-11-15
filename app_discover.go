package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"sync-game-save/internal/core"
)

// SaveDescoberto é uma pasta de save descoberta pela mudança entre a captura e o fim do jogo.
type SaveDescoberto struct {
	Logical  string `json:"logical"`
	Resolved string `json:"resolved"`
	Added    uint64 `json:"added"`
	Removed  uint64 `json:"removed"`
	Bytes    uint64 `json:"bytes"`
}

func envDoExecutavel(exe string) core.Env { return core.SystemEnv(filepath.Dir(exe)) }

func capturaDoExecutavel(dados, exe string) string {
	return filepath.Join(dados, "discover", core.DiscoveryKey(exe)+".json")
}

// DescobrirSave captura o estado das pastas comuns de save, abre o programa e espera ele fechar.
// Nada é sincronizado: a descoberta só registra o estado e lista os candidatos depois.
func (App) DescobrirSave(exe string) (int, error) {
	return descobrirSave(pastaDados(), exe, 5*time.Second)
}

func descobrirSave(dados, exe string, quiet time.Duration) (int, error) {
	if info, err := os.Stat(exe); err != nil || !info.Mode().IsRegular() {
		return 0, errors.New("Não encontrei este executável. Confira o caminho e tente de novo.")
	}
	if core.ProcessRunning(exe) {
		return 0, errors.New("Este programa já está aberto. Feche-o antes de descobrir onde ele salva.")
	}
	captura := core.CaptureNow(envDoExecutavel(exe))
	if err := os.MkdirAll(filepath.Join(dados, "discover"), 0755); err != nil {
		return 0, fmt.Errorf("Não consegui preparar a pasta de descoberta: %w", err)
	}
	bruto, err := json.Marshal(captura)
	if err == nil {
		err = os.WriteFile(capturaDoExecutavel(dados, exe), bruto, 0644)
	}
	if err != nil {
		return 0, fmt.Errorf("Não consegui guardar a captura: %w", err)
	}
	code, err := core.RunProcess(exe, nil, "", quiet, time.Hour, nil)
	if err != nil {
		return 0, fmt.Errorf("Não consegui acompanhar o jogo durante a descoberta: %v. Confira o caminho do executável.", err)
	}
	return code, nil
}

// CandidatosSave lista as pastas que mudaram desde a captura, das mais prováveis às menos.
// O usuário confirma ou edita: nada entra na biblioteca sem esse passo.
func (App) CandidatosSave(exe string) ([]SaveDescoberto, error) {
	bruto, err := os.ReadFile(capturaDoExecutavel(pastaDados(), exe))
	if err != nil {
		return nil, errors.New("Não há captura guardada deste jogo. Comece a descoberta de novo.")
	}
	var antes core.Capture
	if err := json.Unmarshal(bruto, &antes); err != nil {
		return nil, fmt.Errorf("A captura guardada não pôde ser lida: %v. Comece a descoberta de novo.", err)
	}
	env := envDoExecutavel(exe)
	out := []SaveDescoberto{}
	for _, c := range core.CompareCapture(antes, env) {
		resolvido, _ := core.Resolve(c.Logical, env)
		out = append(out, SaveDescoberto{Logical: c.Logical, Resolved: resolvido, Added: c.Added, Removed: c.Removed, Bytes: c.Bytes})
		if len(out) == 12 {
			break
		}
	}
	return out, nil
}
