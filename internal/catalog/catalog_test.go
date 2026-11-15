package catalog

import (
	"os"
	"path/filepath"
	"testing"

	"sync-game-save/internal/core"
)

const manifestYAML = `
"Aggelos":
  files:
    <winAppData>/Aggelos:
      tags: [save]
      when:
        - os: windows
    <base>/config.ini:
      tags: [config]
  installDir:
    Aggelos: {}
  launch:
    <base>/LAUNCHER_x64.exe:
      - when:
          - os: windows
  steam:
    id: 123
"Half-Life 2: Épisode One":
  installDir:
    Half-Life 2: {}
`

func put(t *testing.T, path string, size int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, make([]byte, size), 0644); err != nil {
		t.Fatal(err)
	}
}

func build(t *testing.T, yaml string) (string, *Catalog) {
	t.Helper()
	dir := t.TempDir()
	manifest := filepath.Join(dir, "m.yaml")
	if err := os.WriteFile(manifest, []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}
	db := filepath.Join(dir, "catalog.db")
	if _, err := Build(manifest, db); err != nil {
		t.Fatal(err)
	}
	c, err := Open(db)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Close() })
	return dir, c
}

func TestNormaliza(t *testing.T) {
	if got := Normalize("Épisode One!"); got != "episodeone" {
		t.Fatal(got)
	}
}

func TestBuildContaJogosEArquivos(t *testing.T) {
	dir := t.TempDir()
	manifest := filepath.Join(dir, "m.yaml")
	os.WriteFile(manifest, []byte(manifestYAML), 0644)
	stats, err := Build(manifest, filepath.Join(dir, "c.db"))
	if err != nil || stats != (Stats{Games: 2, Files: 2}) {
		t.Fatal(stats, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "c.db.tmp")); !os.IsNotExist(err) {
		t.Fatal("o arquivo temporário ficou")
	}
}

func TestAchaPelaPastaEPeloNome(t *testing.T) {
	_, c := build(t, manifestYAML)
	if hits, err := c.ByInstallDir("AGGELOS"); err != nil || len(hits) != 1 {
		t.Fatal(hits, err)
	}
	if hits, _ := c.ByInstallDir("half-life 2"); len(hits) != 1 || hits[0].Name != "Half-Life 2: Épisode One" {
		t.Fatal(hits)
	}
	if hits, _ := c.Search("agge", 5); len(hits) == 0 || hits[0].Name != "Aggelos" {
		t.Fatal(hits)
	}
	if hits, _ := c.Search("Ag", 5); len(hits) == 0 || hits[0].Name != "Aggelos" {
		t.Fatal("prefixo curto", hits)
	}
	if hits, _ := c.Search("zzzzz", 5); len(hits) != 0 {
		t.Fatal(hits)
	}
	if _, err := c.Search(`"agg`, 5); err != nil {
		t.Fatal("aspas quebraram a consulta:", err)
	}
}

func TestCaminhosDeSaveELaunch(t *testing.T) {
	_, c := build(t, manifestYAML)
	hits, _ := c.ByInstallDir("Aggelos")
	id := hits[0].ID
	paths, err := c.SavePaths(id)
	if err != nil || len(paths) != 2 {
		t.Fatal(paths, err)
	}
	var save *SavePath
	for i := range paths {
		if paths[i].Tags == "save" {
			save = &paths[i]
		}
	}
	if save == nil || save.Path != "<winAppData>/Aggelos" || save.OS == nil || *save.OS != "windows" {
		t.Fatal(paths)
	}
	if launch, _ := c.LaunchPaths(id); len(launch) != 1 || launch[0] != "<base>/LAUNCHER_x64.exe" {
		t.Fatal(launch)
	}
	if steam, _ := c.SteamID(id); steam == nil || *steam != 123 {
		t.Fatal(steam)
	}
}

func TestAceitaCaractereDeControleDoManifestoReal(t *testing.T) {
	dir := t.TempDir()
	manifest := filepath.Join(dir, "m.yaml")
	os.WriteFile(manifest, []byte("\"Jogo\":\n  launch:\n    <base>/a.exe:\n      - arguments: \"-x\x7f\"\n"), 0644)
	if stats, err := Build(manifest, filepath.Join(dir, "c.db")); err != nil || stats.Games != 1 {
		t.Fatal(stats, err)
	}
}

const recognizeYAML = `
"241600":
  files:
    <winDocuments>/SavedGames/Aggelos:
      tags: [save]
      when:
        - os: windows
    <winAppData>/Aggelos:
      tags: [config]
      when:
        - os: windows
    <xdgData>/Aggelos:
      tags: [save]
      when:
        - os: linux
  installDir:
    Aggelos: {}
  launch:
    <base>/Aggelos.exe:
      - when:
          - os: windows
`

func recognizeSetup(t *testing.T) (string, *Catalog, core.Env) {
	root, c := build(t, recognizeYAML)
	return root, c, core.Env{
		"home":         filepath.Join(root, "home"),
		"winDocuments": filepath.Join(root, "docs"),
		"winAppData":   filepath.Join(root, "appdata"),
	}
}

func TestLimpaNomeDaPasta(t *testing.T) {
	if got := CleanFolderName("The_Game.v1.2.3 [GOG] (x64)"); got != "The Game" {
		t.Fatal(got)
	}
	if got := CleanFolderName("Aggelos"); got != "Aggelos" {
		t.Fatal(got)
	}
}

func TestReconhecePelaPastaComSaveExistente(t *testing.T) {
	root, c, env := recognizeSetup(t)
	game := filepath.Join(root, "Games", "Aggelos")
	put(t, filepath.Join(game, "Aggelos.exe"), 10)
	put(t, filepath.Join(game, "unins000.exe"), 999)
	put(t, filepath.Join(root, "docs", "SavedGames", "Aggelos", "slot1.sav"), 5)
	r, err := c.Recognize(game, env, true)
	if err != nil {
		t.Fatal(err)
	}
	if r.Confidence != ConfidenceHigh || r.Name != "Aggelos" || r.Exe == nil || filepath.Base(*r.Exe) != "Aggelos.exe" {
		t.Fatal(r)
	}
	if len(r.SaveCandidates) != 1 {
		t.Fatal(r.SaveCandidates)
	}
	s := r.SaveCandidates[0]
	if s.Logical != "<winDocuments>/SavedGames/Aggelos" || !s.Exists || s.Files != 1 || !s.Selected {
		t.Fatal(s)
	}
}

func TestSaveInexistenteNaoVemMarcado(t *testing.T) {
	root, c, env := recognizeSetup(t)
	game := filepath.Join(root, "Aggelos")
	put(t, filepath.Join(game, "a.exe"), 3)
	r, err := c.Recognize(game, env, true)
	if err != nil || len(r.SaveCandidates) != 1 || r.SaveCandidates[0].Exists || r.SaveCandidates[0].Selected {
		t.Fatal(r, err)
	}
}

func TestJogoDesconhecidoCaiNoManual(t *testing.T) {
	root, c, env := recognizeSetup(t)
	game := filepath.Join(root, "Jogo Que Nao Existe 99")
	put(t, filepath.Join(game, "grande.exe"), 50)
	put(t, filepath.Join(game, "redist", "vcredist_x64.exe"), 500)
	r, err := c.Recognize(game, env, true)
	if err != nil || r.Confidence != ConfidenceNone || r.CatalogID != nil || len(r.SaveCandidates) != 0 {
		t.Fatal(r, err)
	}
	if exe := GuessExe(game); exe == nil || filepath.Base(*exe) != "grande.exe" {
		t.Fatal(exe)
	}
}

func TestVarrePastaComVariosJogos(t *testing.T) {
	root, c, env := recognizeSetup(t)
	lib := filepath.Join(root, "Biblioteca")
	put(t, filepath.Join(lib, "Aggelos", "Aggelos.exe"), 10)
	put(t, filepath.Join(lib, "Desconhecido Total", "x.exe"), 10)
	put(t, filepath.Join(lib, "Loja", "Dentro Do Nivel 2", "y.exe"), 10)
	put(t, filepath.Join(lib, "Docs", "leia.txt"), 1)
	r, err := c.ScanFolder(lib, 2, env, true)
	if err != nil || len(r.Recognized) != 1 || len(r.Unknown) != 2 {
		t.Fatal(r, err)
	}
	// Uma pasta com .exe em subpasta já conta como jogo, e não se desce dentro dela.
	r1, err := c.ScanFolder(lib, 1, env, true)
	if err != nil || len(r1.Recognized) != 1 || len(r1.Unknown) != 2 {
		t.Fatal(r1, err)
	}
}
