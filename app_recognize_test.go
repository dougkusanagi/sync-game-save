package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReconhecerPastaSemCatalogo(t *testing.T) {
	t.Setenv("SGS_DATA_DIR", t.TempDir())
	dir := filepath.Join(t.TempDir(), "The_Game.v1.2.3 [GOG] (x64)")
	if err := os.MkdirAll(filepath.Join(dir, "redist"), 0755); err != nil {
		t.Fatal(err)
	}
	for name, data := range map[string]string{"pequeno.exe": "x", "jogo.exe": "12345", "redist/vcredist.exe": "123456789"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(data), 0644); err != nil {
			t.Fatal(err)
		}
	}
	got, err := (App{}).ReconhecerPasta(dir)
	if err != nil || got.Name != "The Game" || got.Confidence != "None" || got.Exe == nil || filepath.Base(*got.Exe) != "jogo.exe" || len(got.SaveCandidates) != 0 {
		t.Fatal(got, err)
	}
}
func TestReconhecimentoNaoIgnoraCatalogoExistente(t *testing.T) {
	data := t.TempDir()
	t.Setenv("SGS_DATA_DIR", data)
	if err := os.WriteFile(filepath.Join(data, "catalog.db"), []byte("existente"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := (App{}).ReconhecerPasta(t.TempDir()); err == nil {
		t.Fatal("catálogo existente foi ignorado")
	}
}
func TestConverterCaminhoParaMarcador(t *testing.T) {
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)
	got := (App{}).ConverterCaminho(filepath.Join(home, "saves", "Jogo"))
	if got != "<home>/saves/Jogo" {
		t.Fatal(got)
	}
}
