package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Os registros de sync ficam em <dados>/logs/sync-AAAA-MM-DD.log, um arquivo por dia, para
// investigar uma falha. Nunca recebem segredos: só o ID do jogo, o resultado e a mensagem de erro.
const diasDeLog = 30

var logMu sync.Mutex

func pastaDeLogs(dados string) string { return filepath.Join(dados, "logs") }

// registrar grava uma linha no log do dia. Falhas ao gravar o log nunca interrompem o sync.
func registrar(dados, nivel, formato string, args ...any) {
	logMu.Lock()
	defer logMu.Unlock()
	pasta := pastaDeLogs(dados)
	if os.MkdirAll(pasta, 0o755) != nil {
		return
	}
	agora := time.Now()
	f, err := os.OpenFile(filepath.Join(pasta, "sync-"+agora.Format("2006-01-02")+".log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	msg := strings.ReplaceAll(fmt.Sprintf(formato, args...), "\n", " ")
	fmt.Fprintf(f, "%s %-5s %s\n", agora.Format("2006-01-02T15:04:05"), nivel, msg)
}

// limparLogsAntigos apaga só os arquivos sync-*.log com mais de diasDeLog dias.
func limparLogsAntigos(dados string, agora time.Time) {
	logMu.Lock()
	defer logMu.Unlock()
	itens, err := os.ReadDir(pastaDeLogs(dados))
	if err != nil {
		return
	}
	limite := agora.AddDate(0, 0, -diasDeLog)
	for _, it := range itens {
		n := it.Name()
		if it.IsDir() || !strings.HasPrefix(n, "sync-") || !strings.HasSuffix(n, ".log") {
			continue
		}
		if info, err := it.Info(); err == nil && info.ModTime().Before(limite) {
			_ = os.Remove(filepath.Join(pastaDeLogs(dados), n))
		}
	}
}

// AbrirPastaDeLogs abre a pasta dos registros no gerenciador de arquivos do sistema.
func (App) AbrirPastaDeLogs() error {
	pasta := pastaDeLogs(pastaDados())
	if err := os.MkdirAll(pasta, 0o755); err != nil {
		return fmt.Errorf("Não consegui criar a pasta dos registros (%s): %v", pasta, err)
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", pasta)
	case "darwin":
		cmd = exec.Command("open", pasta)
	default:
		cmd = exec.Command("xdg-open", pasta)
	}
	// O explorer devolve código 1 mesmo quando abre a pasta: só falha de iniciar conta.
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("Não consegui abrir a pasta dos registros (%s): %v", pasta, err)
	}
	go func() { _ = cmd.Wait() }()
	return nil
}
