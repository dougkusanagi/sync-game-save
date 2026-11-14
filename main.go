// Prova de conceito do app em MyGo (passo 1 de docs/plans/migracao-mygo.md).
// Convive com src-tauri/ até a migração terminar.
package main

import (
	_ "embed"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/egoist/mygo"
)

const versao = "1.0.0"

//go:embed resources/tray.png
var iconeBandeja []byte

// App reúne os comandos que a interface chama.
type App struct{}

// Versao devolve a versão do app.
func (App) Versao() string { return versao }

// JogoEstado avisa a interface quando um jogo abre ou fecha.
type JogoEstado struct {
	ID     int64  `json:"id"`
	Estado string `json:"estado"`
}

var EventoJogoEstado = mygo.NewEvent[JogoEstado]("jogo-estado")

// pastaDados é a mesma pasta usada pelo Tauri (app_data_dir com o identificador do app).
func pastaDados() (string, error) {
	if d := os.Getenv("SGS_DATA_DIR"); d != "" {
		return d, nil
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "com.syncgamesave.app"), nil
}

func main() {
	dados, err := pastaDados()
	if err != nil {
		log.Fatal(err)
	}
	arte := filepath.Join(dados, "art")
	_ = os.MkdirAll(arte, 0o755)
	// A interface só lê as artes baixadas (capa://localhost/<arquivo>), não o resto dos dados.
	if err := mygo.Protocol.Handle("capa", http.FileServer(http.Dir(arte))); err != nil {
		log.Fatal(err)
	}

	mygo.Bind(App{})

	var saindo bool
	mygo.App.OnBeforeQuit(func(*mygo.QuitEvent) { saindo = true })

	mygo.App.WhenReady(func() {
		win := mygo.NewWindow(mygo.WindowOptions{
			Title:    "Sync Game Save",
			Width:    1280,
			Height:   720,
			StateKey: "principal",
			URL:      "/",
		})
		// Fechar a janela esconde o app na bandeja; sair só pelo menu da bandeja.
		win.OnClose(func(e *mygo.CloseEvent) {
			if !saindo {
				e.PreventDefault()
				win.Hide()
			}
		})
		tray, err := mygo.NewTray(mygo.TrayOptions{
			Icon:    iconeBandeja,
			ToolTip: "Sync Game Save",
			Menu: mygo.NewMenu([]*mygo.MenuItem{
				{Label: "Abrir", Click: func(*mygo.MenuItem, *mygo.Window) { win.Show() }},
				mygo.Separator(),
				{Label: "Sair", Role: mygo.RoleQuit},
			}),
		})
		if err != nil {
			log.Println("bandeja:", err)
			return
		}
		_ = tray
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
