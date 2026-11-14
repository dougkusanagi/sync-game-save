package main

import (
	"embed"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed all:dist
var assets embed.FS

//go:embed build/tray.png
var iconeBandeja []byte

type App struct{}

func (App) Versao() string { return "1.0.0" }

type JogoEstado struct {
	ID     int64  `json:"id"`
	Estado string `json:"estado"`
}

func init() { application.RegisterEvent[JogoEstado]("jogo-estado") }

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
	capas := http.StripPrefix("/capa/", http.FileServer(http.Dir(arte)))

	var saindo bool
	app := application.New(application.Options{
		Name:     "Sync Game Save",
		Windows:  opcoesWindowsTeste(),
		Services: []application.Service{application.NewService(&App{})},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
			// A interface só lê as artes baixadas (/capa/<arquivo>), não o resto dos dados.
			Middleware: func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if strings.HasPrefix(r.URL.Path, "/capa/") {
						capas.ServeHTTP(w, r)
						return
					}
					next.ServeHTTP(w, r)
				})
			},
		},
		OnShutdown: func() { saindo = true },
	})

	win := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title: "Sync Game Save", Width: 1280, Height: 720, URL: "/",
	})
	win.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		if !saindo {
			e.Cancel()
			win.Hide()
		}
	})

	menu := app.NewMenu()
	menu.Add("Abrir").OnClick(func(*application.Context) { win.Show() })
	menu.AddSeparator()
	menu.Add("Sair").OnClick(func(*application.Context) { saindo = true; app.Quit() })
	app.SystemTray.New().SetIcon(iconeBandeja).SetMenu(menu).OnClick(func() { win.Show() })

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
