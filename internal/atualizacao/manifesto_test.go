package atualizacao

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/wailsapp/wails/v3/pkg/updater"
	"github.com/wailsapp/wails/v3/pkg/updater/providers/endpoint"
)

func TestManifestoAssinadoEAceitoPeloProvedorDoWails(t *testing.T) {
	privPEM, pubPEM, err := GerarChaves()
	if err != nil {
		t.Fatal(err)
	}
	priv, err := LerChavePrivada(privPEM)
	if err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(t.TempDir(), "sync-game-save.exe")
	conteudo := []byte("executavel novo")
	os.WriteFile(exe, conteudo, 0o755)

	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/latest-wails.json" {
			m, err := Gerar(Entrada{Versao: "2.0.0", Notas: "Versão em Go", Arquivo: exe, URL: srv.URL + "/sync-game-save.exe"}, priv)
			if err != nil {
				t.Error(err)
			}
			w.Write(m)
			return
		}
		w.Write(conteudo)
	}))
	defer srv.Close()

	prov, err := endpoint.New(endpoint.Config{URL: srv.URL + "/latest-wails.json"})
	if err != nil {
		t.Fatal(err)
	}
	req := updater.CheckRequest{CurrentVersion: "1.1.0", Platform: "windows", Arch: "amd64"}
	rel, err := prov.Check(context.Background(), req)
	if err != nil || rel == nil {
		t.Fatal(rel, err)
	}
	if rel.Version != "2.0.0" || rel.Notes != "Versão em Go" || rel.Artifact.Filetype != "exe" || rel.Verification == nil {
		t.Fatal(rel)
	}
	resumo := sha256.Sum256(conteudo)
	if string(rel.Verification.Digest) != string(resumo[:]) || rel.Verification.SignatureAlgo != "ed25519" {
		t.Fatal("o resumo ou o algoritmo diferem do esperado", rel.Verification)
	}
	// A mesma conferência que o atualizador faz: Ed25519 sobre o resumo, com a chave pública pinada.
	bloco, _ := pem.Decode(pubPEM)
	pubAny, err := x509.ParsePKIXPublicKey(bloco.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	if !ed25519.Verify(pubAny.(ed25519.PublicKey), rel.Verification.Digest, rel.Verification.Signature) {
		t.Fatal("a assinatura não confere com a chave pública")
	}
	// Outra chave não valida.
	_, outraPub, _ := GerarChaves()
	b2, _ := pem.Decode(outraPub)
	p2, _ := x509.ParsePKIXPublicKey(b2.Bytes)
	if ed25519.Verify(p2.(ed25519.PublicKey), rel.Verification.Digest, rel.Verification.Signature) {
		t.Fatal("assinatura aceita por outra chave")
	}
	// Versão igual ou mais nova no PC: não há atualização.
	for _, atual := range []string{"2.0.0", "2.1.0"} {
		req.CurrentVersion = atual
		if rel, err := prov.Check(context.Background(), req); err != nil || rel != nil {
			t.Fatal(atual, rel, err)
		}
	}
}

func TestGerarRecusaDadosIncompletosEChaveInvalida(t *testing.T) {
	priv, _, _ := GerarChaves()
	k, _ := LerChavePrivada(priv)
	if _, err := Gerar(Entrada{Versao: "1.0.0"}, k); err == nil {
		t.Fatal("dados incompletos aceitos")
	}
	if _, err := LerChavePrivada([]byte("não é pem")); err == nil {
		t.Fatal("chave inválida aceita")
	}
}

func TestPonteTauriGeraOLatestJSONDaV1(t *testing.T) {
	agora := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	b, err := PonteTauri("2.0.0", "Versão em Go", "https://exemplo.test/instalador.exe", "ASSINATURA", agora)
	if err != nil {
		t.Fatal(err)
	}
	var m struct {
		Version   string                                     `json:"version"`
		Notes     string                                     `json:"notes"`
		PubDate   string                                     `json:"pub_date"`
		Platforms map[string]struct{ Signature, URL string } `json:"platforms"`
	}
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	p := m.Platforms["windows-x86_64"]
	if m.Version != "2.0.0" || m.PubDate != "2026-11-15T12:00:00Z" || p.Signature != "ASSINATURA" || p.URL != "https://exemplo.test/instalador.exe" || m.Platforms["windows-x86_64-nsis"] != p {
		t.Fatal(string(b))
	}
	if _, err := PonteTauri("2.0.0", "", "", "x", agora); err == nil {
		t.Fatal("dados incompletos aceitos")
	}
}
