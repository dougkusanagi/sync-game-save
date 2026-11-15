package main

import (
	"embed"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"sync-game-save/internal/gamepad"
)

//go:embed all:dist
var assets embed.FS

//go:embed build/tray.png
var iconeBandeja []byte

type App struct{}

const versaoApp = "1.1.0"

func (App) Versao() string { return versaoApp }

type JogoEstado struct {
	ID     int64  `json:"id"`
	Estado string `json:"estado"`
}

func init() {
	application.RegisterEvent[JogoEstado]("jogo-estado")
	application.RegisterEvent[gamepad.Output]("controle-acao")
}

func pastaDados() string {
	if d := os.Getenv("SGS_DATA_DIR"); d != "" {
		return d
	}
	base, _ := os.UserConfigDir()
	return filepath.Join(base, "com.syncgamesave.app")
}

func main() {
	arte := filepath.Join(pastaDados(), "art")
	_ = os.MkdirAll(arte, 0o755)

	app := application.New(application.Options{
		Name:     "Sync Game Save",
		Windows:  opcoesWindowsTeste(),
		Services: []application.Service{application.NewService(&App{}), application.NewService(notificador)},
		Assets: application.AssetOptions{
			Handler:    application.AssetFileServerFS(assets),
			Middleware: middlewareCapas(arte),
		},
		OnShutdown: func() { saindo = true },
	})

	win := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title: "Sync Game Save", Width: 1280, Height: 720, URL: "/", X: posicaoJanelaTeste(), Y: posicaoJanelaTeste(), InitialPosition: posicaoInicialTeste(),
	})
	win.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		if !saindo {
			e.Cancel()
			win.Hide()
		}
	})

	menu := app.NewMenu()
	menu.Add("Abrir Sync Game Save").OnClick(func(*application.Context) { mostrarJanela(win) })
	menu.AddSeparator()
	menu.Add("Sair").OnClick(func(*application.Context) { saindo = true; app.Quit() })
	bandeja := app.SystemTray.New().SetIcon(iconeBandeja).SetMenu(menu).OnClick(func() { mostrarJanela(win) })
	bandeja.SetTooltip("Sync Game Save")

	iniciarAtualizador(app)
	iniciarControle(app, win)
	iniciarTarefasEmSegundoPlano()

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

// iniciarControle lê o controle em segundo plano e envia as ações à interface. As ações só valem
// com a janela visível e em primeiro plano: um controle usado dentro de um jogo nunca deve clicar
// em "Remover" no launcher escondido. Sem suporte a controle, o app segue com teclado e mouse.
func iniciarControle(app *application.App, win *application.WebviewWindow) {
	src, err := gamepad.New()
	if err != nil {
		log.Printf("controle: %v", err)
		return
	}
	pronta := func() bool {
		return win.IsVisible() && !win.IsMinimised() && janelaEmPrimeiroPlano(win.NativeWindow(), win.IsFocused)
	}
	emitir := func(o gamepad.Output) { app.Event.Emit("controle-acao", o) }
	go gamepad.Loop(src, pronta, emitir, time.Now, make(chan struct{}), 8*time.Millisecond)
}

func mostrarJanela(win *application.WebviewWindow) {
	win.Show()
	win.UnMinimise()
	win.Focus()
}
