package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"sync-game-save/internal/core"
	"sync-game-save/internal/library"
)

type OpcoesJogar struct {
	Forcar         bool    `json:"forcar"`
	Conflito       *string `json:"conflito"`
	SemSincronizar bool    `json:"sem_sincronizar"`
}

// ResultadoJogar tem o mesmo formato JSON da versão Rust (campo kind).
type ResultadoJogar struct {
	Kind     string  `json:"kind"`
	PC       *string `json:"pc,omitempty"`
	ExitCode *int    `json:"exit_code,omitempty"`
	Synced   *bool   `json:"synced,omitempty"`
	Notice   *string `json:"notice,omitempty"`
}

var (
	jogosAbertos   = map[int64]bool{}
	jogosAbertosMu sync.Mutex
	// emitirEstado envia o andamento ao frontend; os testes trocam por uma função própria.
	emitirEstado = func(e JogoEstado) {
		if app := application.Get(); app != nil {
			app.Event.Emit("jogo-estado", e)
		}
	}
)

func iniciarOperacao(id int64) (func(), error) {
	jogosAbertosMu.Lock()
	defer jogosAbertosMu.Unlock()
	if jogosAbertos[id] {
		return nil, errors.New("Este jogo está aberto ou sendo sincronizado. Aguarde terminar antes de tentar novamente.")
	}
	jogosAbertos[id] = true
	return func() {
		jogosAbertosMu.Lock()
		delete(jogosAbertos, id)
		jogosAbertosMu.Unlock()
	}, nil
}

func nomeSeguro(name string) string {
	s := strings.Map(func(c rune) rune {
		if strings.ContainsRune(`<>:"/\|?*`, c) || c < 0x20 || (c >= 0x7f && c < 0xa0) {
			return '_'
		}
		return c
	}, name)
	return strings.TrimRight(strings.TrimSpace(s), ".")
}

func (App) Jogar(id int64, opcoes *OpcoesJogar) (ResultadoJogar, error) {
	var o OpcoesJogar
	if opcoes != nil {
		o = *opcoes
	}
	emit := func(estado string) { emitirEstado(JogoEstado{ID: id, Estado: estado}) }
	// Jogos da Steam fecham e voltam pela Steam: a espera cobre esse intervalo.
	r, err := jogar(pastaDados(), id, o, 15*time.Second, 2*time.Minute, emit)
	if err != nil {
		emit("erro")
	} else {
		emit("pronto")
	}
	return r, err
}

func (App) PararJogo(id int64) (*bool, error) {
	lib, err := abrirBiblioteca()
	if err != nil {
		return nil, err
	}
	defer lib.Close()
	game, err := lib.Get(id)
	if err != nil {
		return nil, err
	}
	found, still, err := core.StopProcess(game.Exe)
	if err != nil || !found {
		return nil, err
	}
	return &still, nil
}

func (App) ForcarParadaJogo(id int64) (bool, error) {
	lib, err := abrirBiblioteca()
	if err != nil {
		return false, err
	}
	defer lib.Close()
	game, err := lib.Get(id)
	if err != nil {
		return false, err
	}
	return core.ForceStopProcess(game.Exe)
}

func agora() uint64 { return uint64(time.Now().Unix()) }

func contagemSync(dados string, ok bool) {
	lib, err := library.Open(filepath.Join(dados, "library.db"))
	if err != nil {
		return
	}
	defer lib.Close()
	ler := func(chave string) int64 {
		v, err := lib.Setting(chave)
		if err != nil || v == nil {
			return 0
		}
		n, _ := strconv.ParseInt(*v, 10, 64)
		return n
	}
	if !ok {
		_ = lib.SetSetting("sync_ok_streak", "0")
		return
	}
	atual := ler("sync_ok_streak") + 1
	melhor := max(ler("sync_best_streak"), atual)
	_ = lib.SetSetting("sync_ok_streak", strconv.FormatInt(atual, 10))
	_ = lib.SetSetting("sync_best_streak", strconv.FormatInt(melhor, 10))
}

func jogar(dados string, id int64, o OpcoesJogar, quiet, every time.Duration, emit func(string)) (ResultadoJogar, error) {
	fim, err := iniciarOperacao(id)
	if err != nil {
		return ResultadoJogar{}, err
	}
	defer fim()
	r, err := jogarInterno(dados, id, o, quiet, every, emit)
	if err != nil {
		contagemSync(dados, false)
	}
	return r, err
}

func raizesDoJogo(game library.Game) ([]core.Root, error) {
	if len(game.SavePaths) == 0 {
		return nil, errors.New("Este jogo ainda não tem uma pasta de save. Indique onde ele guarda o progresso para poder sincronizar.")
	}
	base := ""
	if game.InstallDir != nil {
		base = *game.InstallDir
	}
	env := core.SystemEnv(base)
	var roots []core.Root
	for _, logical := range game.SavePaths {
		// Pasta escolhida à mão e sem marcador conhecido: o caminho absoluto vale como está.
		if !strings.HasPrefix(logical, "<") {
			roots = append(roots, core.Root{Label: logical, Path: logical})
			continue
		}
		p, ok := core.Resolve(logical, env)
		if !ok {
			return nil, fmt.Errorf("Não consegui encontrar a pasta de save \"%s\" neste PC.", logical)
		}
		roots = append(roots, core.Root{Label: logical, Path: p})
	}
	return roots, nil
}

func jogarInterno(dados string, id int64, o OpcoesJogar, quiet, every time.Duration, emit func(string)) (ResultadoJogar, error) {
	lib, err := library.Open(filepath.Join(dados, "library.db"))
	if err != nil {
		return ResultadoJogar{}, err
	}
	game, err := lib.Get(id)
	var syncDir, device *string
	if err == nil {
		syncDir, err = lib.Setting("sync_dir")
	}
	if err == nil {
		device, err = lib.Setting("device")
	}
	lib.Close()
	if err != nil {
		return ResultadoJogar{}, err
	}
	nomeDoPC := nomePCPadrao()
	if device != nil {
		nomeDoPC = *device
	}
	executar := func(beat func()) (int, error) {
		emit("jogando")
		inicio := time.Now()
		code, err := core.RunProcess(game.Exe, nil, "", quiet, every, beat)
		// O tempo conta mesmo se o jogo caiu; a espera final por processos restantes não entra.
		secs := int64((time.Since(inicio) - quiet) / time.Second)
		if l, e := library.Open(filepath.Join(dados, "library.db")); e == nil {
			_ = l.AddPlaytime(id, secs, int64(agora()))
			l.Close()
		}
		if err != nil {
			return 0, fmt.Errorf("Não consegui abrir o jogo: %v. Confira o caminho do executável.", err)
		}
		return code, nil
	}
	finalizado := func(code int, synced bool, notice string) ResultadoJogar {
		return ResultadoJogar{Kind: "finished", ExitCode: &code, Synced: &synced, Notice: &notice}
	}
	if o.SemSincronizar {
		code, err := executar(nil)
		if err != nil {
			return ResultadoJogar{}, err
		}
		return finalizado(code, false, "Jogo fechado. O save não foi sincronizado."), nil
	}

	eng, err := montarEngine(dados, game, syncDir, nomeDoPC)
	if err != nil {
		return ResultadoJogar{}, err
	}
	jdir := filepath.Join(eng.StateDir, "journal")
	emit("sincronizando")
	if err := core.JournalBegin(jdir, game.Name, "abrir", eng.Now); err != nil {
		return ResultadoJogar{}, err
	}
	before, beforeErr := eng.BeforeLaunch(o.Forcar)
	if beforeErr == nil && before.Status == core.SyncConflict && o.Conflito != nil {
		choice := core.UseCloud
		if *o.Conflito == "local" {
			choice = core.UseLocal
		}
		if beforeErr = eng.Resolve(choice); beforeErr == nil {
			before = core.BeforeResult{Status: core.Ready}
		}
	}
	if err := core.JournalEnd(jdir, game.Name); err != nil {
		return ResultadoJogar{}, err
	}
	if beforeErr != nil {
		return ResultadoJogar{}, beforeErr
	}
	aoAbrir := ""
	switch before.Status {
	case core.SyncConflict:
		return ResultadoJogar{Kind: "conflict"}, nil
	case core.LockedByOther:
		pc := before.Device
		return ResultadoJogar{Kind: "locked", PC: &pc}, nil
	case core.RestoredFromCloud:
		aoAbrir = "O save da pasta de sincronização foi restaurado antes de abrir (o anterior ficou em backup). "
	case core.UploadedLocal:
		aoAbrir = "O save deste PC foi enviado antes de abrir. "
	}
	roots := eng.Roots
	batida := func() { _ = core.HeartbeatLock(eng.GameDir, eng.Device, agora()) }
	// Um erro de acompanhamento não garante que o grupo tenha terminado:
	// a trava e os saves ficam como estão até ser possível confirmar o encerramento.
	code, err := executar(batida)
	if err != nil {
		return ResultadoJogar{}, err
	}
	emit("salvando")
	if err := core.JournalBegin(jdir, game.Name, "fechar", eng.Now); err != nil {
		return ResultadoJogar{}, err
	}
	eng.Now = agora()
	created, err := eng.AfterExit(batida)
	if err != nil {
		return ResultadoJogar{}, err
	}
	if err := core.JournalEnd(jdir, game.Name); err != nil {
		return ResultadoJogar{}, err
	}
	digest := ""
	if idx, err := core.ScanRoots(roots); err == nil {
		digest = idx.Digest()
	}
	l, err := library.Open(filepath.Join(dados, "library.db"))
	if err != nil {
		return ResultadoJogar{}, err
	}
	err = l.RecordSync(id, digest, int64(agora()))
	l.Close()
	if err != nil {
		return ResultadoJogar{}, err
	}
	contagemSync(dados, true)
	aoFechar := "Save sem mudanças depois de jogar."
	if created {
		aoFechar = "Save enviado para a pasta de sincronização."
	}
	return finalizado(code, true, aoAbrir+aoFechar), nil
}

// montarEngine reúne pasta de sync, saves e estado local de um jogo.
func montarEngine(dados string, game library.Game, syncDir *string, device string) (core.Engine, error) {
	if syncDir == nil {
		return core.Engine{}, errors.New("Escolha primeiro a pasta de sincronização, em Configurações.")
	}
	if info, err := os.Stat(*syncDir); err != nil || !info.IsDir() {
		return core.Engine{}, fmt.Errorf("A pasta de sincronização (%s) não está disponível. Confira se o OneDrive ou o serviço de nuvem está ativo, ou jogue sem sincronizar.", *syncDir)
	}
	roots, err := raizesDoJogo(game)
	if err != nil {
		return core.Engine{}, err
	}
	return core.Engine{
		GameDir:  filepath.Join(*syncDir, "SyncGameSave", "games", nomeSeguro(game.Name)),
		Roots:    roots,
		StateDir: filepath.Join(dados, "state", strconv.FormatInt(game.ID, 10)),
		Device:   device,
		Now:      agora(),
	}, nil
}
