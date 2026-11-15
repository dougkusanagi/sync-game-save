// Package atualizacao gera e assina o manifesto de atualização que o atualizador do Wails lê
// (provedor "endpoint"). A assinatura é Ed25519 sobre o SHA-256 do arquivo, que é o que o
// atualizador confere antes de trocar o executável.
package atualizacao

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Artefato descreve um arquivo publicado no release.
type Artefato struct {
	URL           string `json:"url"`
	Filename      string `json:"filename"`
	Filetype      string `json:"filetype"`
	Size          int64  `json:"size"`
	Platform      string `json:"platform"`
	Arch          string `json:"arch"`
	DigestAlgo    string `json:"digestAlgo"`
	Digest        string `json:"digest"`
	SignatureAlgo string `json:"signatureAlgo"`
	Signature     string `json:"signature"`
}

type Manifesto struct {
	SchemaVersion int        `json:"schemaVersion"`
	Version       string     `json:"version"`
	Notes         string     `json:"notes,omitempty"`
	PublishedAt   string     `json:"publishedAt"`
	Artifacts     []Artefato `json:"artifacts"`
}

// GerarChaves cria um par Ed25519: a chave privada em PKCS#8 e a pública em PKIX, ambas em PEM.
func GerarChaves() (privada, publica []byte, err error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	privDER, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return nil, nil, err
	}
	pubDER, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return nil, nil, err
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privDER}),
		pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER}), nil
}

// LerChavePrivada lê uma chave Ed25519 em PEM (PKCS#8).
func LerChavePrivada(pemBytes []byte) (ed25519.PrivateKey, error) {
	bloco, _ := pem.Decode(pemBytes)
	if bloco == nil {
		return nil, errors.New("a chave privada não está em PEM")
	}
	k, err := x509.ParsePKCS8PrivateKey(bloco.Bytes)
	if err != nil {
		return nil, fmt.Errorf("chave privada inválida: %w", err)
	}
	priv, ok := k.(ed25519.PrivateKey)
	if !ok {
		return nil, errors.New("a chave privada não é Ed25519")
	}
	return priv, nil
}

// Entrada reúne o que o manifesto precisa saber de um release.
type Entrada struct {
	Versao                  string
	Notas                   string
	Arquivo                 string // caminho local do executável
	URL                     string // onde o arquivo ficará publicado
	Plataforma, Arquitetura string
	Agora                   time.Time
}

// Gerar monta o manifesto do release e assina o arquivo com a chave privada.
func Gerar(e Entrada, priv ed25519.PrivateKey) ([]byte, error) {
	if e.Versao == "" || e.URL == "" || e.Arquivo == "" {
		return nil, errors.New("faltam versão, arquivo ou URL")
	}
	dados, err := os.ReadFile(e.Arquivo)
	if err != nil {
		return nil, err
	}
	resumo := sha256.Sum256(dados)
	plat, arq := e.Plataforma, e.Arquitetura
	if plat == "" {
		plat = "windows"
	}
	if arq == "" {
		arq = "amd64"
	}
	agora := e.Agora
	if agora.IsZero() {
		agora = time.Now()
	}
	m := Manifesto{
		SchemaVersion: 1,
		Version:       e.Versao,
		Notes:         e.Notas,
		PublishedAt:   agora.UTC().Format(time.RFC3339),
		Artifacts: []Artefato{{
			URL: e.URL, Filename: filepath.Base(e.Arquivo), Filetype: "exe", Size: int64(len(dados)),
			Platform: plat, Arch: arq,
			DigestAlgo: "sha256", Digest: base64.StdEncoding.EncodeToString(resumo[:]),
			SignatureAlgo: "ed25519", Signature: base64.StdEncoding.EncodeToString(ed25519.Sign(priv, resumo[:])),
		}},
	}
	return json.MarshalIndent(m, "", "  ")
}
