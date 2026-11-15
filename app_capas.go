package main

import (
	"net/http"
	"strings"
)

// middlewareCapas serve /capa/<arquivo> só a partir da pasta de arte. A interface lê as artes
// baixadas, não o resto dos dados.
func middlewareCapas(arte string) func(http.Handler) http.Handler {
	capas := http.StripPrefix("/capa/", http.FileServer(http.Dir(arte)))
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/capa/") {
				capas.ServeHTTP(w, r)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
