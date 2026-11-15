// Package cli implementa o comando sgs: abre um jogo com sync seguro, faz backup, restaura e
// consulta o estado. Os comandos, as saídas e os códigos de saída seguem a versão Rust.
package cli

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"sync-game-save/internal/core"
)

const help = `Uso: sgs <comando> --sync-dir <pasta> --game <nome> --save-dir <pasta do save ou <marcador>/caminho> [opções]

Comandos:
  run [--force] [--on-conflict local|cloud|abort] -- <programa> [args...]
        Sincroniza, abre o programa, espera ele fechar e envia o save.
  backup     Cria um snapshot se o save mudou.
  restore [--snapshot <id>]   Restaura o snapshot (padrão: o atual), com backup antes.
  status [--json]   Mostra o estado do sync.
  list [--json]     Lista os snapshots.

Opções:
  --state-dir <pasta>  Estado local (padrão: %LOCALAPPDATA%/SyncGameSave/<jogo>)
  --device <nome>      Nome deste PC (padrão: nome do computador)
  --cwd <pasta>        Pasta de trabalho do jogo (padrão: a pasta do executável)
  --quiet-secs <n>     Segundos sem nenhum processo do jogo para considerar fechado (padrão: 5)
`

type opts struct {
	cmd, syncDir, game, saveDir string
	stateDir, device, snapshot  string
	onConflict, cwd             string
	force, json                 bool
	program                     []string
	now                         *uint64
	quietSecs, heartbeatSecs    uint64
}

func parse(args []string) (opts, error) {
	if len(args) == 0 {
		return opts{}, errors.New("falta o comando")
	}
	o := opts{cmd: args[0], quietSecs: 5, heartbeatSecs: 120}
	rest := args[1:]
	var haveSync, haveGame, haveSave bool
	for i := 0; i < len(rest); i++ {
		a := rest[i]
		val := func() (string, error) {
			if i+1 >= len(rest) {
				return "", fmt.Errorf("falta o valor de %s", a)
			}
			i++
			return rest[i], nil
		}
		num := func() (uint64, error) {
			v, err := val()
			if err != nil {
				return 0, err
			}
			n, err := strconv.ParseUint(v, 10, 64)
			if err != nil {
				return 0, fmt.Errorf("%s precisa ser um número", a)
			}
			return n, nil
		}
		var err error
		switch a {
		case "--sync-dir":
			o.syncDir, err = val()
			haveSync = err == nil
		case "--game":
			o.game, err = val()
			haveGame = err == nil
		case "--save-dir":
			o.saveDir, err = val()
			haveSave = err == nil
		case "--state-dir":
			o.stateDir, err = val()
		case "--device":
			o.device, err = val()
		case "--snapshot":
			o.snapshot, err = val()
		case "--on-conflict":
			o.onConflict, err = val()
		case "--cwd":
			o.cwd, err = val()
		case "--now":
			var n uint64
			if n, err = num(); err == nil {
				o.now = &n
			}
		case "--quiet-secs":
			o.quietSecs, err = num()
		case "--heartbeat-secs":
			o.heartbeatSecs, err = num()
		case "--force":
			o.force = true
		case "--json":
			o.json = true
		case "--":
			o.program = append([]string(nil), rest[i+1:]...)
			i = len(rest)
		default:
			err = fmt.Errorf("opção desconhecida: %s", a)
		}
		if err != nil {
			return opts{}, err
		}
	}
	switch {
	case !haveSync:
		return opts{}, errors.New("falta --sync-dir")
	case !haveGame:
		return opts{}, errors.New("falta --game")
	case !haveSave:
		return opts{}, errors.New("falta --save-dir")
	}
	return o, nil
}

func deviceName() string {
	for _, k := range []string{"COMPUTERNAME", "HOSTNAME"} {
		if v := os.Getenv(k); v != "" {
			return v
		}
	}
	return "PC"
}

func (o opts) nowUnix() uint64 {
	if o.now != nil {
		return *o.now
	}
	return uint64(time.Now().Unix())
}

// resolveSaveDir aceita caminho comum ou lógico (<winDocuments>/SavedGames/Jogo).
func resolveSaveDir(p string) (string, error) {
	if !strings.HasPrefix(p, "<") {
		return p, nil
	}
	if r, ok := core.Resolve(p, core.SystemEnv("")); ok {
		return r, nil
	}
	return "", fmt.Errorf("não consegui resolver o caminho %s neste PC", p)
}

func engine(o opts) (core.Engine, error) {
	game := core.SafeName(o.game)
	if game == "" {
		return core.Engine{}, errors.New("o nome do jogo ficou vazio")
	}
	state := o.stateDir
	if state == "" {
		base := os.Getenv("LOCALAPPDATA")
		if base == "" {
			if home := os.Getenv("HOME"); home != "" {
				base = filepath.Join(home, ".local", "share")
			}
		}
		if base == "" {
			return core.Engine{}, errors.New("não achei a pasta de dados local; use --state-dir")
		}
		state = filepath.Join(base, "SyncGameSave", game)
	}
	save, err := resolveSaveDir(o.saveDir)
	if err != nil {
		return core.Engine{}, err
	}
	device := o.device
	if device == "" {
		device = deviceName()
	}
	return core.Engine{
		GameDir:  filepath.Join(o.syncDir, "SyncGameSave", "games", game),
		Roots:    []core.Root{{Label: "", Path: save}},
		StateDir: state,
		Device:   device,
		Now:      o.nowUnix(),
	}, nil
}

// Exec executa um comando e devolve o código de saída do processo.
func Exec(args []string, out io.Writer, in io.Reader) (int, error) {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" || args[0] == "help" {
		_, err := io.WriteString(out, help)
		return 0, err
	}
	o, err := parse(args)
	if err != nil {
		return 0, err
	}
	eng, err := engine(o)
	if err != nil {
		return 0, err
	}
	jdir := filepath.Join(eng.StateDir, "journal")
	pending, err := core.JournalPending(jdir)
	if err != nil {
		return 0, err
	}
	for _, p := range pending {
		fmt.Fprintf(out, "Aviso: a operação \"%s\" de %s não terminou na última vez. Confira o save.\n", p.Op, p.Game)
	}
	switch o.cmd {
	case "status":
		return status(eng, o, out)
	case "list":
		ids, err := core.ListSnapshots(eng.GameDir)
		if err != nil {
			return 0, err
		}
		switch {
		case o.json:
			if ids == nil {
				ids = []string{}
			}
			b, _ := json.Marshal(ids)
			fmt.Fprintln(out, string(b))
		case len(ids) == 0:
			fmt.Fprintln(out, "Nenhum snapshot ainda.")
		default:
			for _, id := range ids {
				fmt.Fprintln(out, id)
			}
		}
		return 0, nil
	case "backup":
		if err := core.JournalBegin(jdir, o.game, "backup", eng.Now); err != nil {
			return 0, err
		}
		created, err := eng.AfterExit(nil)
		if err != nil {
			return 0, err
		}
		if err := core.JournalEnd(jdir, o.game); err != nil {
			return 0, err
		}
		if created {
			fmt.Fprintln(out, "Snapshot criado.")
		} else {
			fmt.Fprintln(out, "Nada mudou; nenhum snapshot novo.")
		}
		return 0, nil
	case "restore":
		return restore(eng, o, jdir, out)
	case "run":
		return run(eng, o, jdir, out, in)
	}
	return 0, fmt.Errorf("comando desconhecido: %s. Use `sgs help`.", o.cmd)
}

func restore(eng core.Engine, o opts, jdir string, out io.Writer) (int, error) {
	head, err := core.ReadHead(eng.GameDir)
	if err != nil {
		return 0, err
	}
	id := o.snapshot
	if id == "" {
		if head == nil {
			return 0, errors.New("não há snapshot para restaurar")
		}
		id = head.Snapshot
	}
	snap := filepath.Join(eng.GameDir, "snapshots", id)
	backup := filepath.Join(eng.StateDir, "backups", strconv.FormatUint(eng.Now, 10)+"_restaurar")
	if err := core.JournalBegin(jdir, o.game, "restore", eng.Now); err != nil {
		return 0, err
	}
	if err := core.RestoreSnapshot(snap, eng.Roots, backup); err != nil {
		return 0, err
	}
	if err := core.JournalEnd(jdir, o.game); err != nil {
		return 0, err
	}
	fmt.Fprintf(out, "Snapshot %s restaurado. O save anterior está em %s.\n", id, backup)
	return 0, nil
}

func status(eng core.Engine, o opts, out io.Writer) (int, error) {
	head, err := core.ReadHead(eng.GameDir)
	if err != nil {
		return 0, err
	}
	st, lock, err := core.LockState(eng.GameDir, eng.Device, eng.Now)
	if err != nil {
		return 0, err
	}
	local, err := core.ScanRoots(eng.Roots)
	if err != nil {
		return 0, err
	}
	lockTxt := "livre"
	switch st {
	case core.LockOwn:
		lockTxt = "deste PC"
	case core.LockHeldByOther:
		lockTxt = "em uso em " + lock.Device
	case core.LockStale:
		lockTxt = "antiga de " + lock.Device + " (abandonada)"
	}
	if o.json {
		v := map[string]any{"head": nil, "lock": lockTxt, "localFiles": len(local.Files), "localDigest": local.Digest(), "cloudDigest": nil}
		if head != nil {
			v["head"], v["cloudDigest"] = head.Snapshot, head.IndexDigest
		}
		b, _ := json.Marshal(v)
		fmt.Fprintln(out, string(b))
		return 0, nil
	}
	atual := "nenhum"
	if head != nil {
		atual = head.Snapshot
	}
	fmt.Fprintf(out, "Snapshot atual: %s\nTrava: %s\nArquivos de save locais: %d\n", atual, lockTxt, len(local.Files))
	return 0, nil
}

func run(eng core.Engine, o opts, jdir string, out io.Writer, in io.Reader) (int, error) {
	if len(o.program) == 0 {
		return 0, errors.New("falta o programa depois de `--`")
	}
	if info, err := os.Stat(o.syncDir); err != nil || !info.IsDir() {
		return 0, fmt.Errorf("a pasta de sync %s não está disponível. Confira se a nuvem está montada, ou abra o jogo sem o sgs.", o.syncDir)
	}
	if err := core.JournalBegin(jdir, o.game, "abrir", eng.Now); err != nil {
		return 0, err
	}
	before, err := eng.BeforeLaunch(o.force)
	if err != nil {
		return 0, err
	}
	if before.Status == core.SyncConflict {
		var choice *core.Choice
		pick := func(c core.Choice) { choice = &c }
		switch o.onConflict {
		case "local":
			pick(core.UseLocal)
		case "cloud":
			pick(core.UseCloud)
		case "abort":
		default:
			fmt.Fprintln(out, "Conflito: o save deste PC e o da nuvem mudaram. Usar o [l]ocal, o da [n]uvem ou [c]ancelar?")
			line, _ := bufio.NewReader(in).ReadString('\n')
			switch strings.TrimSpace(line) {
			case "l":
				pick(core.UseLocal)
			case "n":
				pick(core.UseCloud)
			}
		}
		if choice == nil {
			if err := core.JournalEnd(jdir, o.game); err != nil {
				return 0, err
			}
			fmt.Fprintln(out, "Cancelado. Nada foi alterado.")
			return 2, nil
		}
		if err := eng.Resolve(*choice); err != nil {
			return 0, err
		}
		before = core.BeforeResult{Status: core.Ready}
	}
	if before.Status == core.LockedByOther {
		if err := core.JournalEnd(jdir, o.game); err != nil {
			return 0, err
		}
		fmt.Fprintf(out, "O jogo parece aberto em %s. Use --force para abrir mesmo assim.\n", before.Device)
		return 3, nil
	}
	if err := core.JournalEnd(jdir, o.game); err != nil {
		return 0, err
	}
	switch before.Status {
	case core.RestoredFromCloud:
		fmt.Fprintln(out, "Save da nuvem restaurado. Se havia um save neste PC, uma cópia ficou na pasta de backups.")
	case core.UploadedLocal:
		fmt.Fprintln(out, "Save deste PC enviado para a nuvem.")
	default:
		fmt.Fprintln(out, "Save em dia.")
	}
	beat := func() { _ = core.HeartbeatLock(eng.GameDir, eng.Device, o.nowUnix()) }
	every := time.Duration(o.heartbeatSecs) * time.Second
	code, err := core.RunProcess(o.program[0], o.program[1:], o.cwd, time.Duration(o.quietSecs)*time.Second, every, beat)
	if err != nil {
		// Uma falha de acompanhamento não comprova que todos os processos terminaram.
		return 0, fmt.Errorf("não consegui acompanhar %s: %v", o.program[0], err)
	}
	if err := core.JournalBegin(jdir, o.game, "fechar", eng.Now); err != nil {
		return 0, err
	}
	created, err := eng.AfterExit(beat)
	if err != nil {
		return 0, err
	}
	if err := core.JournalEnd(jdir, o.game); err != nil {
		return 0, err
	}
	if created {
		fmt.Fprintln(out, "Save enviado para a nuvem.")
	} else {
		fmt.Fprintln(out, "Save sem mudanças.")
	}
	return code, nil
}
