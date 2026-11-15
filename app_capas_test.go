package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const bs = "\x5c"

func TestCapasSoSaemDaPastaDeArte(t *testing.T) {
	dados := t.TempDir()
	arte := filepath.Join(dados, "art")
	os.MkdirAll(arte, 0755)
	os.WriteFile(filepath.Join(arte, "1.jpg"), []byte("capa"), 0644)
	os.WriteFile(filepath.Join(dados, "library.db"), []byte("SEGREDO"), 0644)
	os.WriteFile(filepath.Join(dados, "segredo.txt"), []byte("SEGREDO"), 0644)
	chamouProximo := false
	h := middlewareCapas(arte)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		chamouProximo = true
		w.Write([]byte("interface"))
	}))
	pegar := func(alvo string) (int, string) {
		req := httptest.NewRequest("GET", "http://localhost/", nil)
		req.URL.Path = alvo
		req.URL.RawPath = ""
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec.Code, rec.Body.String()
	}
	if c, b := pegar("/capa/1.jpg"); c != 200 || b != "capa" {
		t.Fatal(c, b)
	}
	for _, alvo := range []string{
		"/capa/../library.db", "/capa/../segredo.txt", "/capa/..%2flibrary.db", "/capa/%2e%2e/library.db",
		"/capa/.." + bs + "library.db", "/capa//library.db", "/capa/" + strings.ReplaceAll(filepath.Join(dados, "library.db"), bs, "/"),
		"/capa/C:/Windows/win.ini", "/capa/C:" + bs + "Windows" + bs + "win.ini", "/capa/./../library.db", "/capa/1.jpg/../../library.db",
	} {
		if c, b := pegar(alvo); strings.Contains(b, "SEGREDO") || strings.Contains(b, "[fonts]") || c == 200 && b != "" && b != "capa" {
			t.Fatal("saiu da pasta de arte:", alvo, c, b)
		}
	}
	if chamouProximo {
		t.Fatal("rotas /capa/ não devem ir para a interface")
	}
	if c, b := pegar("/index.html"); c != 200 || b != "interface" || !chamouProximo {
		t.Fatal(c, b)
	}
}
