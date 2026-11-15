package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/wailsapp/wails/v3/pkg/updater"
)

type atualizadorFalso struct {
	rel                 *updater.Release
	errCheck, errBaixar error
	chamadas            []string
}

func (a *atualizadorFalso) Check(context.Context) (*updater.Release, error) {
	a.chamadas = append(a.chamadas, "check")
	return a.rel, a.errCheck
}
func (a *atualizadorFalso) DownloadAndInstall(context.Context) error {
	a.chamadas = append(a.chamadas, "baixar")
	return a.errBaixar
}
func (a *atualizadorFalso) Restart(context.Context) error {
	a.chamadas = append(a.chamadas, "reiniciar")
	return nil
}

func TestVerificarAtualizacao(t *testing.T) {
	if _, err := verificarAtualizacao(nil); err == nil {
		t.Fatal("sem atualizador deve avisar")
	}
	f := &atualizadorFalso{}
	if a, err := verificarAtualizacao(f); a != nil || err != nil {
		t.Fatal("sem versão nova", a, err)
	}
	f.rel = &updater.Release{Version: "2.0.0", Notes: "  Notas  "}
	a, err := verificarAtualizacao(f)
	if err != nil || a == nil || a.Versao != "2.0.0" || a.Atual != versaoApp || a.Notas == nil || *a.Notas != "Notas" {
		t.Fatal(a, err)
	}
	f.errCheck = errors.New("dns: https://x/segredo")
	if _, err := verificarAtualizacao(f); err == nil || strings.Contains(err.Error(), "segredo") || !strings.Contains(err.Error(), "internet") {
		t.Fatal("o erro deve ser claro e sem detalhes técnicos:", err)
	}
}

func TestInstalarAtualizacaoSoSemJogoAbertoEReiniciaNoFim(t *testing.T) {
	defer func() { saindo = false }()
	f := &atualizadorFalso{rel: &updater.Release{Version: "2.0.0"}}
	fim, _ := iniciarOperacao(5)
	if err := instalarAtualizacao(f, func() { saindo = true }); err == nil || len(f.chamadas) != 0 || saindo {
		t.Fatal("atualizou com um jogo aberto", err, f.chamadas)
	}
	fim()
	if err := instalarAtualizacao(f, func() { saindo = true }); err != nil || !saindo {
		t.Fatal(err, saindo)
	}
	if strings.Join(f.chamadas, ",") != "check,baixar,reiniciar" {
		t.Fatal(f.chamadas)
	}
	// Falha ao baixar: não reinicia e a mensagem é clara.
	saindo = false
	g := &atualizadorFalso{rel: &updater.Release{Version: "2.0.0"}, errBaixar: errors.New("assinatura inválida")}
	if err := instalarAtualizacao(g, func() { saindo = true }); err == nil || saindo || strings.Contains(err.Error(), "assinatura") || strings.Contains(strings.Join(g.chamadas, ","), "reiniciar") {
		t.Fatal(err, g.chamadas)
	}
	if err := instalarAtualizacao(&atualizadorFalso{}, func() {}); err == nil || !strings.Contains(err.Error(), "Nenhuma") {
		t.Fatal(err)
	}
}
