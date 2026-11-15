package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTemasEmbutidosEscolhaEPersistencia(t *testing.T) {
	t.Setenv("SGS_DATA_DIR", t.TempDir())
	app := App{}
	l, err := app.Temas()
	if err != nil || l.Atual != "escuro" || len(l.Temas) != 2 || l.Temas[0].ID != "escuro" || l.Temas[1].Tokens["--sgs-color-scheme"] != "light" {
		t.Fatal(l, err)
	}
	if l, err = app.DefinirTema("claro"); err != nil || l.Atual != "claro" {
		t.Fatal(l, err)
	}
	if l, _ = app.Temas(); l.Atual != "claro" {
		t.Fatal("escolha não persistiu", l.Atual)
	}
	if _, err := app.DefinirTema("nao-existe"); err == nil {
		t.Fatal("tema inexistente aceito")
	}
}

func TestTemaInstaladoAceitaValidoEIgnoraRuim(t *testing.T) {
	dados := t.TempDir()
	t.Setenv("SGS_DATA_DIR", dados)
	escrever := func(id, conteudo string) {
		pasta := filepath.Join(dados, "themes", id)
		os.MkdirAll(pasta, 0755)
		os.WriteFile(filepath.Join(pasta, "theme.json"), []byte(conteudo), 0644)
	}
	escrever("meu-tema", `{"name":"Meu tema","author":"Ana","version":"2.0.0","tokens":{
		"--sgs-bg":"#123456","--sgs-accent":"var(--sgs-accent-default)","cor":"#fff","--sgs-text":"url(https://exemplo.test/x)"}}`)
	escrever("futuro", `{"name":"Futuro","min_app_version":"99.0.0","tokens":{"--sgs-bg":"#000"}}`)
	escrever("claro", `{"name":"Falso claro"}`)
	escrever("id inválido", `{"name":"X"}`)
	lista := temasInstalados(dados)
	if len(lista) != 1 {
		t.Fatal(lista)
	}
	tema := lista[0]
	if tema.ID != "meu-tema" || tema.Name != "Meu tema" || tema.Author != "Ana" || tema.Builtin || len(tema.Tokens) != 2 {
		t.Fatal(tema)
	}
	if _, ok := tema.Tokens["--sgs-text"]; ok {
		t.Fatal("url aceita")
	}
}

func TestTokensRecusamUrlECaracteresDeRegra(t *testing.T) {
	for chave, valor := range map[string]string{"--sgs-bg": "#fff", "--sgs-font": `"Outfit", system-ui`} {
		if !tokenValido(chave, valor) {
			t.Fatal(chave, valor)
		}
	}
	for _, c := range [][2]string{{"--sgs-bg", "url(https://x)"}, {"--sgs-bg", "red; color: blue"}, {"--sgs-bg", "red}body{color:red"}, {"--outra", "#fff"}, {"--sgs-", "#fff"}} {
		if tokenValido(c[0], c[1]) {
			t.Fatal("aceito:", c)
		}
	}
}

func TestComparaVersoes(t *testing.T) {
	if !versaoNoMinimo("0.1.0", "0.1.0") || !versaoNoMinimo("0.1", "0.1.2") || versaoNoMinimo("0.2.0", "0.1.9") || versaoNoMinimo("x", "0.1.0") {
		t.Fatal("comparação errada")
	}
}
