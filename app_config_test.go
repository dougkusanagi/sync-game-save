package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"sync-game-save/internal/core"
	"sync-game-save/internal/library"
)

func TestEstadoRetomaConfiguracoesEDiario(t *testing.T) {
	data := t.TempDir()
	t.Setenv("SGS_DATA_DIR", data)
	steam := t.TempDir()
	t.Setenv("SGS_STEAM_DIR", steam)
	if err := os.MkdirAll(filepath.Join(steam, "config"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(steam, "config", "loginusers.vdf"), []byte("\"76561198000000001\"\n{\n\"MostRecent\" \"0\"\n}\n\"76561198263490373\"\n{\n\"MostRecent\" \"1\"\n}"), 0644); err != nil {
		t.Fatal(err)
	}
	lib, err := library.Open(filepath.Join(data, "library.db"))
	if err != nil {
		t.Fatal(err)
	}
	for key, value := range map[string]string{"sync_dir": "D:/Nuvem", "device": "PC-A", "steamgriddb_api_key": "chave"} {
		if err := lib.SetSetting(key, value); err != nil {
			t.Fatal(err)
		}
	}
	lib.Close()
	if err := core.JournalBegin(filepath.Join(data, "state", "1", "journal"), "Jogo", "abrir", 1); err != nil {
		t.Fatal(err)
	}
	state, err := (App{}).Estado()
	if err != nil || state.PastaSync == nil || *state.PastaSync != "D:/Nuvem" || state.NomePC != "PC-A" || state.SteamID == nil || *state.SteamID != "76561198263490373" || !state.SteamGridDBConfigurada || len(state.OperacoesInterrompidas) != 1 || state.OperacoesInterrompidas[0] != "Jogo (abrir)" {
		t.Fatal(state, err)
	}
}

func TestConfiguracaoNuvemPreservaDispositivosEArquivos(t *testing.T) {
	data := t.TempDir()
	t.Setenv("SGS_DATA_DIR", data)
	cloud := t.TempDir()
	if err := os.MkdirAll(filepath.Join(cloud, "SyncGameSave", "games", "Outro"), 0755); err != nil {
		t.Fatal(err)
	}
	devices := filepath.Join(cloud, "SyncGameSave", "devices.json")
	if err := os.WriteFile(devices, []byte(`["PC-ANTIGO"]`), 0644); err != nil {
		t.Fatal(err)
	}
	app := App{}
	if err := app.DefinirNomePc(" PC-NOVO "); err != nil {
		t.Fatal(err)
	}
	check, err := app.DefinirPastaSync(cloud)
	if err != nil || !check.Writable || !check.HasData || len(check.Games) != 1 || check.Games[0] != "Outro" {
		t.Fatal(check, err)
	}
	raw, err := os.ReadFile(devices)
	if err != nil {
		t.Fatal(err)
	}
	var values []string
	if err := json.Unmarshal(raw, &values); err != nil || strings.Join(values, ",") != "PC-ANTIGO,PC-NOVO" {
		t.Fatal(string(raw), err)
	}
	if _, err := app.DefinirPastaSync(cloud); err != nil {
		t.Fatal(err)
	}
	raw, _ = os.ReadFile(devices)
	json.Unmarshal(raw, &values)
	if len(values) != 2 {
		t.Fatal(values)
	}
}

func TestConfiguracaoRecusaDevicesInvalidosSemSobrescrever(t *testing.T) {
	t.Setenv("SGS_DATA_DIR", t.TempDir())
	cloud := t.TempDir()
	root := filepath.Join(cloud, "SyncGameSave")
	if err := os.MkdirAll(root, 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "devices.json")
	if err := os.WriteFile(path, []byte("dados incompletos"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := (App{}).DefinirPastaSync(cloud); err == nil {
		t.Fatal("arquivo inválido aceito")
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != "dados incompletos" {
		t.Fatal(string(got), err)
	}
	lib, err := library.Open(filepath.Join(pastaDados(), "library.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer lib.Close()
	value, err := lib.Setting("sync_dir")
	if err != nil || value != nil {
		t.Fatal(value, err)
	}
}
