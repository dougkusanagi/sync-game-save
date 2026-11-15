package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"sync-game-save/internal/core"
	"sync-game-save/internal/library"
)

type EstadoApp struct {
	PastaSync              *string  `json:"pasta_sync"`
	NomePC                 string   `json:"nome_pc"`
	CatalogoPronto         bool     `json:"catalogo_pronto"`
	OperacoesInterrompidas []string `json:"operacoes_interrompidas"`
	SteamConfigurada       bool     `json:"steam_configurada"`
	SteamID                *string  `json:"steam_id"`
	SteamGridDBConfigurada bool     `json:"steamgriddb_configurada"`
}

type VerificacaoPasta struct {
	Exists   bool     `json:"exists"`
	Writable bool     `json:"writable"`
	HasData  bool     `json:"has_data"`
	Games    []string `json:"games"`
	Message  *string  `json:"message"`
}

func abrirBiblioteca() (*library.Library, error) {
	return library.Open(filepath.Join(pastaDados(), "library.db"))
}
func nomePC(lib *library.Library) (string, error) {
	name, err := lib.Setting("device")
	if err != nil {
		return "", err
	}
	if name != nil {
		return *name, nil
	}
	return nomePCPadrao(), nil
}
func nomePCPadrao() string {
	name := os.Getenv("COMPUTERNAME")
	if name == "" {
		name = os.Getenv("HOSTNAME")
	}
	if name == "" {
		name = "PC"
	}
	return name
}
func valorConfigurado(lib *library.Library, key string) (bool, error) {
	v, err := lib.Setting(key)
	return v != nil && strings.TrimSpace(*v) != "", err
}
func (App) Estado() (EstadoApp, error) {
	lib, err := abrirBiblioteca()
	if err != nil {
		return EstadoApp{}, err
	}
	defer lib.Close()
	var state EstadoApp
	if state.PastaSync, err = lib.Setting("sync_dir"); err != nil {
		return state, err
	}
	if state.NomePC, err = nomePC(lib); err != nil {
		return state, err
	}
	if state.SteamID, err = lib.Setting("steam_id"); err != nil {
		return state, err
	}
	if state.SteamID == nil || strings.TrimSpace(*state.SteamID) == "" {
		state.SteamID = localSteamID()
	}
	key, err := valorConfigurado(lib, "steam_api_key")
	if err != nil {
		return state, err
	}
	configuredID, err := valorConfigurado(lib, "steam_id")
	if err != nil {
		return state, err
	}
	state.SteamConfigurada = key && configuredID
	if state.SteamGridDBConfigurada, err = valorConfigurado(lib, "steamgriddb_api_key"); err != nil {
		return state, err
	}
	if _, err := os.Stat(filepath.Join(pastaDados(), "catalog.db")); err == nil {
		state.CatalogoPronto = true
	} else if !errors.Is(err, os.ErrNotExist) {
		return state, err
	}
	state.OperacoesInterrompidas = []string{}
	dirs, err := os.ReadDir(filepath.Join(pastaDados(), "state"))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return state, err
	}
	for _, d := range dirs {
		entries, err := core.JournalPending(filepath.Join(pastaDados(), "state", d.Name(), "journal"))
		if err != nil {
			continue
		}
		for _, e := range entries {
			state.OperacoesInterrompidas = append(state.OperacoesInterrompidas, fmt.Sprintf("%s (%s)", e.Game, e.Op))
		}
	}
	return state, nil
}
func localSteamID() *string {
	var dirs []string
	if dir, ok := os.LookupEnv("SGS_STEAM_DIR"); ok {
		dirs = append(dirs, dir)
	} else {
		for _, key := range []string{"ProgramFiles(x86)", "ProgramFiles"} {
			if v := os.Getenv(key); v != "" {
				dirs = append(dirs, filepath.Join(v, "Steam"))
			}
		}
		home := os.Getenv("HOME")
		if home == "" {
			home = os.Getenv("USERPROFILE")
		}
		if home != "" {
			dirs = append(dirs, filepath.Join(home, ".steam", "steam"), filepath.Join(home, ".local", "share", "Steam"))
		}
	}
	for _, dir := range dirs {
		if data, err := os.ReadFile(filepath.Join(dir, "config", "loginusers.vdf")); err == nil {
			if id := steamIDFromLoginUsers(string(data)); id != nil {
				return id
			}
		}
	}
	return nil
}
func steamIDFromLoginUsers(vdf string) *string {
	var first, current *string
	for _, line := range strings.Split(vdf, "\n") {
		value := strings.Trim(strings.TrimSpace(line), `"`)
		if len(value) == 17 && strings.HasPrefix(value, "7656") && strings.Trim(value, "0123456789") == "" {
			v := value
			current = &v
			if first == nil {
				first = &v
			}
		}
		var fields []string
		for _, f := range strings.Split(line, `"`) {
			if strings.TrimSpace(f) != "" {
				fields = append(fields, f)
			}
		}
		if len(fields) == 2 && strings.EqualFold(fields[0], "MostRecent") && fields[1] == "1" && current != nil {
			return current
		}
	}
	return first
}
func (App) DefinirNomePc(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("O nome deste PC não pode ficar vazio.")
	}
	lib, err := abrirBiblioteca()
	if err != nil {
		return err
	}
	defer lib.Close()
	return lib.SetSetting("device", name)
}
func (App) VerificarPastaSync(dir string) VerificacaoPasta {
	result := VerificacaoPasta{Games: []string{}}
	info, err := os.Stat(dir)
	result.Exists = err == nil && info.IsDir()
	if result.Exists {
		f, err := os.CreateTemp(dir, ".sgs-teste-*")
		if err == nil {
			name := f.Name()
			_, writeErr := f.WriteString("ok")
			closeErr := f.Close()
			data, readErr := os.ReadFile(name)
			result.Writable = writeErr == nil && closeErr == nil && readErr == nil && string(data) == "ok"
			_ = os.Remove(name)
		}
	}
	root := filepath.Join(dir, "SyncGameSave")
	if entries, err := os.ReadDir(filepath.Join(root, "games")); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				result.Games = append(result.Games, e.Name())
			}
		}
	}
	_, err = os.Stat(filepath.Join(root, "devices.json"))
	result.HasData = err == nil || len(result.Games) > 0
	var message string
	if !result.Exists {
		message = "Esta pasta não existe."
	} else if !result.Writable {
		message = "Não consegui gravar nesta pasta. Escolha outra ou confira as permissões."
	}
	if message != "" {
		result.Message = &message
	}
	return result
}
func (a App) DefinirPastaSync(dir string) (VerificacaoPasta, error) {
	check := a.VerificarPastaSync(dir)
	if check.Message != nil {
		return check, errors.New(*check.Message)
	}
	root := filepath.Join(dir, "SyncGameSave")
	if err := os.MkdirAll(filepath.Join(root, "games"), 0755); err != nil {
		return check, fmt.Errorf("Não consegui preparar a pasta: %w", err)
	}
	lib, err := abrirBiblioteca()
	if err != nil {
		return check, err
	}
	defer lib.Close()
	name, err := nomePC(lib)
	if err != nil {
		return check, err
	}
	path := filepath.Join(root, "devices.json")
	var names []string
	data, err := os.ReadFile(path)
	if err == nil {
		if err := json.Unmarshal(data, &names); err != nil || names == nil {
			return check, errors.New("Não consegui ler a lista de PCs. Aguarde a sincronização da pasta e tente de novo.")
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return check, fmt.Errorf("Não consegui ler a lista de PCs: %w", err)
	}
	found := false
	for _, n := range names {
		if n == name {
			found = true
			break
		}
	}
	if !found {
		names = append(names, name)
		data, err = json.MarshalIndent(names, "", "  ")
		if err != nil {
			return check, err
		}
		tmp, err := os.CreateTemp(root, ".devices-*.tmp")
		if err != nil {
			return check, err
		}
		temp := tmp.Name()
		defer os.Remove(temp)
		_, err = tmp.Write(data)
		if err == nil {
			err = tmp.Sync()
		}
		closeErr := tmp.Close()
		if err == nil {
			err = closeErr
		}
		if err == nil {
			err = os.Rename(temp, path)
		}
		if err != nil {
			return check, fmt.Errorf("Não consegui registrar este PC: %w", err)
		}
	}
	if err := lib.SetSetting("sync_dir", dir); err != nil {
		return check, err
	}
	return check, nil
}
