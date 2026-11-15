package main

// Temas: pacotes locais que trocam só os tokens visuais (--sgs-*). O pacote é JSON; nenhum CSS
// ou JavaScript do tema é executado, e os valores passam por uma lista restrita para não abrir
// URL externa nem injetar regras.

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	temaPadrao    = "escuro"
	limiteTokens  = 200
	limiteValor   = 160
	versaoTemaPad = "0.1.0"
)

type Tema struct {
	ID      string            `json:"id"`
	Name    string            `json:"name"`
	Author  string            `json:"author"`
	Version string            `json:"version"`
	Builtin bool              `json:"builtin"`
	Tokens  map[string]string `json:"tokens"`
}

type ListaTemas struct {
	Atual string `json:"atual"`
	Temas []Tema `json:"temas"`
}

func temaEmbutido(id, nome string, tokens map[string]string) Tema {
	if tokens == nil {
		tokens = map[string]string{}
	}
	return Tema{ID: id, Name: nome, Author: "Sync Game Save", Version: versaoTemaPad, Builtin: true, Tokens: tokens}
}

// temasEmbutidos devolve o tema escuro (o padrão, sem trocas) e o claro oficial.
func temasEmbutidos() []Tema {
	return []Tema{
		temaEmbutido(temaPadrao, "Escuro (padrão)", nil),
		temaEmbutido("claro", "Claro", map[string]string{
			"--sgs-bg":             "#f2f5fb",
			"--sgs-bg-rgb":         "242 245 251",
			"--sgs-surface":        "rgb(255 255 255 / 0.72)",
			"--sgs-surface-strong": "rgb(255 255 255 / 0.92)",
			"--sgs-surface-solid":  "#ffffff",
			"--sgs-surface-hover":  "rgb(16 22 37 / 0.06)",
			"--sgs-border":         "rgb(16 22 37 / 0.14)",
			"--sgs-border-strong":  "rgb(16 22 37 / 0.32)",
			"--sgs-text":           "#101625",
			"--sgs-text-dim":       "rgb(16 22 37 / 0.72)",
			"--sgs-text-faint":     "rgb(16 22 37 / 0.55)",
			"--sgs-accent-default": "#1d6fe0",
			"--sgs-ok":             "#0f8a60",
			"--sgs-info":           "#1d6fe0",
			"--sgs-warn":           "#9a6500",
			"--sgs-danger":         "#c62d45",
			"--sgs-tier-common":    "#5b6b82",
			"--sgs-tier-uncommon":  "#0f8a60",
			"--sgs-tier-rare":      "#1d6fe0",
			"--sgs-tier-ultra":     "#9a6500",
			"--sgs-focus-ring":     "0 0 0 0.1875rem #1d6fe0, 0 0.75rem 2.5rem rgb(16 22 37 / 0.2)",
			"--sgs-capa-bg":        "linear-gradient(135deg, #dfe7f5, #eef2f9)",
			"--sgs-color-scheme":   "light",
		}),
	}
}

type arquivoTema struct {
	Name          string            `json:"name"`
	Author        string            `json:"author"`
	Version       string            `json:"version"`
	MinAppVersion string            `json:"min_app_version"`
	Tokens        map[string]string `json:"tokens"`
}

func idTemaValido(id string) bool {
	if id == "" || len(id) > 40 {
		return false
	}
	for _, c := range id {
		ascii := c < 128 && (unicode.IsLetter(c) || unicode.IsDigit(c))
		if !ascii && c != '-' && c != '_' {
			return false
		}
	}
	return true
}

func tokenValido(chave, valor string) bool {
	nome, ok := strings.CutPrefix(chave, "--sgs-")
	if !ok || nome == "" || len(chave) > 48 {
		return false
	}
	for _, c := range nome {
		if !(c >= 'a' && c <= 'z') && !(c >= '0' && c <= '9') && c != '-' {
			return false
		}
	}
	valor = strings.TrimSpace(valor)
	if valor == "" || len(valor) > limiteValor {
		return false
	}
	baixo := strings.ToLower(valor)
	if strings.Contains(baixo, "@") || strings.Contains(baixo, "url(") || strings.Contains(baixo, "expression(") {
		return false
	}
	for _, c := range valor {
		if unicode.IsControl(c) || strings.ContainsRune(";{}<>\\", c) {
			return false
		}
	}
	return true
}

func analisarVersao(v string) ([3]uint64, bool) {
	nucleo := strings.TrimSpace(v)
	if i := strings.IndexAny(nucleo, "-+"); i >= 0 {
		nucleo = nucleo[:i]
	}
	partes := strings.Split(nucleo, ".")
	var nums [3]uint64
	if len(partes) > 3 {
		return nums, false
	}
	for i, p := range partes {
		n, err := strconv.ParseUint(p, 10, 64)
		if err != nil {
			return nums, false
		}
		nums[i] = n
	}
	return nums, true
}

func versaoNoMinimo(minima, atual string) bool {
	m, ok1 := analisarVersao(minima)
	a, ok2 := analisarVersao(atual)
	if !ok1 || !ok2 {
		return false
	}
	for i := range m {
		if a[i] != m[i] {
			return a[i] > m[i]
		}
	}
	return true
}

func limitar(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n])
}

func temaDoArquivo(id string, a arquivoTema) (Tema, bool) {
	nome := strings.TrimSpace(a.Name)
	if nome == "" || utf8.RuneCountInString(nome) > 60 {
		return Tema{}, false
	}
	if min := strings.TrimSpace(a.MinAppVersion); min != "" && !versaoNoMinimo(min, versaoApp) {
		return Tema{}, false
	}
	autor, versao := strings.TrimSpace(a.Author), strings.TrimSpace(a.Version)
	if autor == "" {
		autor = "Autor não informado"
	}
	if versao == "" {
		versao = "1.0.0"
	}
	chaves := make([]string, 0, len(a.Tokens))
	for k, v := range a.Tokens {
		if tokenValido(k, v) {
			chaves = append(chaves, k)
		}
	}
	sort.Strings(chaves)
	if len(chaves) > limiteTokens {
		chaves = chaves[:limiteTokens]
	}
	tokens := map[string]string{}
	for _, k := range chaves {
		tokens[k] = a.Tokens[k]
	}
	return Tema{ID: id, Name: nome, Author: limitar(autor, 60), Version: limitar(versao, 20), Tokens: tokens}, true
}

// temasInstalados lê themes/<id>/theme.json na pasta de dados. Pacotes inválidos são ignorados.
func temasInstalados(dados string) []Tema {
	entradas, err := os.ReadDir(filepath.Join(dados, "themes"))
	if err != nil {
		return nil
	}
	var lista []Tema
	for _, e := range entradas {
		pasta := filepath.Join(dados, "themes", e.Name())
		if info, err := os.Stat(pasta); err != nil || !info.IsDir() {
			continue
		}
		id := e.Name()
		if !idTemaValido(id) || id == temaPadrao || id == "claro" {
			continue
		}
		bruto, err := os.ReadFile(filepath.Join(pasta, "theme.json"))
		if err != nil {
			continue
		}
		var a arquivoTema
		if json.Unmarshal(bruto, &a) != nil {
			continue
		}
		if t, ok := temaDoArquivo(id, a); ok {
			lista = append(lista, t)
		}
	}
	sort.SliceStable(lista, func(i, j int) bool {
		return strings.ToLower(lista[i].Name) < strings.ToLower(lista[j].Name)
	})
	return lista
}

func listarTemas(dados string) (ListaTemas, error) {
	lib, err := abrirBiblioteca()
	if err != nil {
		return ListaTemas{}, err
	}
	defer lib.Close()
	escolhido, err := lib.Setting("theme")
	if err != nil {
		return ListaTemas{}, err
	}
	temas := append(temasEmbutidos(), temasInstalados(dados)...)
	atual := temaPadrao
	for _, t := range temas {
		if escolhido != nil && t.ID == *escolhido {
			atual = t.ID
		}
	}
	return ListaTemas{Atual: atual, Temas: temas}, nil
}

func (App) Temas() (ListaTemas, error) { return listarTemas(pastaDados()) }

// DefinirTema só aceita um tema da lista, para não guardar um id inválido.
func (App) DefinirTema(id string) (ListaTemas, error) {
	lista, err := listarTemas(pastaDados())
	if err != nil {
		return ListaTemas{}, err
	}
	for _, t := range lista.Temas {
		if t.ID != id {
			continue
		}
		lib, err := abrirBiblioteca()
		if err != nil {
			return ListaTemas{}, err
		}
		defer lib.Close()
		if err := lib.SetSetting("theme", id); err != nil {
			return ListaTemas{}, err
		}
		lista.Atual = id
		return lista, nil
	}
	return ListaTemas{}, errors.New("Este tema não está instalado. Abra as configurações e escolha outro.")
}
