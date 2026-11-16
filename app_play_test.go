package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"sync-game-save/internal/core"
	"sync-game-save/internal/library"
)

// O binário de teste serve de "jogo": com SGS_PLAY_HELPER, grava um save e sai.
func TestMain(m *testing.M) {
	if save := os.Getenv("SGS_PLAY_HELPER"); save != "" {
		if err := os.WriteFile(save, []byte("progresso do jogo"), 0644); err != nil {
			os.Exit(3)
		}
		if os.Getenv("SGS_PLAY_HOLD") != "" {
			// Jogo que ignora o pedido de fechamento: só termina se for encerrado à força.
			time.Sleep(5 * time.Minute)
		}
		os.Exit(5)
	}
	os.Exit(m.Run())
}

type cenarioJogo struct {
	dados, nuvem, save string
	id                 int64
	estados            []string
}

func novoCenario(t *testing.T, saveInicial string) *cenarioJogo {
	t.Helper()
	c := &cenarioJogo{dados: t.TempDir(), nuvem: t.TempDir()}
	t.Setenv("SGS_DATA_DIR", c.dados)
	pasta := t.TempDir()
	c.save = filepath.Join(pasta, "save")
	if err := os.MkdirAll(c.save, 0755); err != nil {
		t.Fatal(err)
	}
	if saveInicial != "" {
		os.WriteFile(filepath.Join(c.save, "slot.sav"), []byte(saveInicial), 0644)
	}
	t.Setenv("SGS_PLAY_HELPER", filepath.Join(c.save, "slot.sav"))
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(exe)
	jogo := filepath.Join(t.TempDir(), "jogo-teste"+filepath.Ext(exe))
	if err := os.WriteFile(jogo, data, 0755); err != nil {
		t.Fatal(err)
	}
	app := App{}
	if c.id, err = app.AdicionarJogo(library.NewGame{Name: "Jogo", Exe: jogo, SavePaths: []string{c.save}}); err != nil {
		t.Fatal(err)
	}
	lib, _ := abrirBiblioteca()
	lib.SetSetting("sync_dir", c.nuvem)
	lib.SetSetting("device", "PC-A")
	lib.Close()
	return c
}

func (c *cenarioJogo) jogar(o OpcoesJogar) (ResultadoJogar, error) {
	return jogar(c.dados, c.id, o, 100*time.Millisecond, time.Hour, func(e string) { c.estados = append(c.estados, e) })
}

func TestJogarSincronizaSaveDepoisDeFechar(t *testing.T) {
	if testing.Short() {
		t.Skip("espera a estabilização do save")
	}
	c := novoCenario(t, "")
	r, err := c.jogar(OpcoesJogar{})
	if err != nil || r.Kind != "finished" || *r.ExitCode != 5 || !*r.Synced || *r.Notice != "Save enviado para a pasta de sincronização." {
		t.Fatal(r, err)
	}
	if got := c.estados; len(got) != 3 || got[0] != "sincronizando" || got[1] != "jogando" || got[2] != "salvando" {
		t.Fatal(got)
	}
	head, err := core.ReadHead(filepath.Join(c.nuvem, "SyncGameSave", "games", "Jogo"))
	if err != nil || head == nil {
		t.Fatal(head, err)
	}
	lib, _ := abrirBiblioteca()
	defer lib.Close()
	game, _ := lib.Get(c.id)
	if game.BaseDigest == nil || game.LastSyncAt == nil {
		t.Fatal("sync não registrado", game)
	}
	if v, _ := lib.Setting("sync_ok_streak"); v == nil || *v != "1" {
		t.Fatal(v)
	}
	if lock, _ := core.ReadLock(filepath.Join(c.nuvem, "SyncGameSave", "games", "Jogo")); lock != nil {
		t.Fatal("trava ficou", lock)
	}
}

func TestJogarSemSincronizarNaoTocaANuvem(t *testing.T) {
	c := novoCenario(t, "")
	r, err := c.jogar(OpcoesJogar{SemSincronizar: true})
	if err != nil || *r.Synced || *r.Notice != "Jogo fechado. O save não foi sincronizado." {
		t.Fatal(r, err)
	}
	if entries, _ := os.ReadDir(c.nuvem); len(entries) != 0 {
		t.Fatal("a nuvem foi alterada", entries)
	}
}

func TestJogarRecusaTravaDeOutroPC(t *testing.T) {
	c := novoCenario(t, "inicial")
	gameDir := filepath.Join(c.nuvem, "SyncGameSave", "games", "Jogo")
	if err := core.AcquireLock(gameDir, "PC-B", agora()); err != nil {
		t.Fatal(err)
	}
	r, err := c.jogar(OpcoesJogar{})
	if err != nil || r.Kind != "locked" || r.PC == nil || *r.PC != "PC-B" {
		t.Fatal(r, err)
	}
	if got, _ := os.ReadFile(filepath.Join(c.save, "slot.sav")); string(got) != "inicial" {
		t.Fatal("o jogo abriu mesmo travado")
	}
}

func TestJogarSemPastaDeSyncOuSave(t *testing.T) {
	c := novoCenario(t, "")
	lib, _ := abrirBiblioteca()
	lib.SetSetting("sync_dir", filepath.Join(c.nuvem, "nao-existe"))
	lib.Close()
	if _, err := c.jogar(OpcoesJogar{}); err == nil {
		t.Fatal("pasta de sync ausente aceita")
	}
	if v := lerConfiguracao(t, "sync_ok_streak"); v != "0" {
		t.Fatal(v)
	}
}

func TestJogarRecusaDuasAberturasDoMesmoJogo(t *testing.T) {
	c := novoCenario(t, "")
	fim, err := iniciarOperacao(c.id)
	if err != nil {
		t.Fatal(err)
	}
	defer fim()
	if _, err := c.jogar(OpcoesJogar{SemSincronizar: true}); err == nil {
		t.Fatal("segunda abertura aceita")
	}
}

func lerConfiguracao(t *testing.T, chave string) string {
	t.Helper()
	lib, err := abrirBiblioteca()
	if err != nil {
		t.Fatal(err)
	}
	defer lib.Close()
	v, _ := lib.Setting(chave)
	if v == nil {
		return ""
	}
	return *v
}

func TestVersoesERestauracaoDeSave(t *testing.T) {
	c := novoCenario(t, "primeira")
	// Duas versões: o save inicial sobe antes de abrir e o jogo grava outra ao fechar.
	if _, err := c.jogar(OpcoesJogar{}); err != nil {
		t.Fatal(err)
	}
	app := App{}
	versoes, err := app.Versoes(c.id)
	if err != nil || len(versoes) != 2 {
		t.Fatal(versoes, err)
	}
	if _, err := app.RestaurarVersao(c.id, "inexistente"); err == nil {
		t.Fatal("versão inexistente aceita")
	}
	// Restaura a mais antiga, que tem o save inicial.
	antiga := versoes[0]
	for _, v := range versoes {
		if v < antiga {
			antiga = v
		}
	}
	backup, err := app.RestaurarVersao(c.id, antiga)
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(filepath.Join(c.save, "slot.sav")); string(got) != "primeira" {
		t.Fatal("save não restaurado:", string(got))
	}
	if _, err := os.Stat(backup); err != nil {
		t.Fatal("sem backup do save anterior", err)
	}
}

func TestPararJogoQueIgnoraFechamentoExigeForcaESincronizaDepois(t *testing.T) {
	if testing.Short() {
		t.Skip("espera o prazo de 10 s do fechamento normal")
	}
	t.Setenv("SGS_PLAY_HOLD", "1")
	c := novoCenario(t, "")
	app := App{}
	lib, _ := abrirBiblioteca()
	game, _ := lib.Get(c.id)
	lib.Close()

	type saida struct {
		r   ResultadoJogar
		err error
	}
	fim := make(chan saida, 1)
	go func() {
		r, err := c.jogar(OpcoesJogar{})
		fim <- saida{r, err}
	}()
	// Espera o jogo ser acompanhado.
	limite := time.Now().Add(20 * time.Second)
	for !core.ProcessRunning(game.Exe) {
		if time.Now().After(limite) {
			t.Fatal("o jogo não começou a ser acompanhado")
		}
		time.Sleep(100 * time.Millisecond)
	}
	// Força antes do prazo é recusada.
	if _, err := app.ForcarParadaJogo(c.id); err == nil {
		t.Fatal("a força antes do pedido normal deveria ser recusada")
	}
	inicio := time.Now()
	resta, err := app.PararJogo(c.id)
	if err != nil || resta == nil || !*resta {
		t.Fatal("o jogo ignora o fechamento e deveria continuar aberto", resta, err)
	}
	if time.Since(inicio) < 10*time.Second {
		t.Fatal("o prazo de 10 s não foi respeitado", time.Since(inicio))
	}
	select {
	case s := <-fim:
		t.Fatal("o sync não pode começar com o jogo aberto", s)
	default:
	}
	if ok, err := app.ForcarParadaJogo(c.id); err != nil || !ok {
		t.Fatal(ok, err)
	}
	select {
	case s := <-fim:
		if s.err != nil || s.r.Kind != "finished" || !*s.r.Synced {
			t.Fatal(s)
		}
	case <-time.After(60 * time.Second):
		t.Fatal("o jogo não terminou depois da força")
	}
	head, err := core.ReadHead(filepath.Join(c.nuvem, "SyncGameSave", "games", "Jogo"))
	if err != nil || head == nil {
		t.Fatal("o sync deveria ter rodado depois do fim do grupo", head, err)
	}
	if core.ProcessRunning(game.Exe) {
		t.Fatal("ainda há processo do jogo")
	}
}
