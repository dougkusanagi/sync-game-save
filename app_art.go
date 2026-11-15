package main

// Artes (capa, fundo, logo e capa vertical): Steam CDN quando o catálogo tem steam.id e
// SteamGridDB como alternativa. As imagens baixadas ficam no cache local (pasta art); a chave
// do SteamGridDB nunca é devolvida.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"

	"sync-game-save/internal/catalog"
	"sync-game-save/internal/library"
)

const sgdb = "https://www.steamgriddb.com/api/v2"

type tipoArte string

const (
	arteHeader tipoArte = "header" // capa horizontal 460×215
	arteHero   tipoArte = "hero"   // arte larga de fundo
	arteLogo   tipoArte = "logo"   // logo com fundo transparente
	arteCover  tipoArte = "cover"  // capa vertical 600×900
)

func analisarTipoArte(s string) (tipoArte, bool) {
	switch t := tipoArte(s); t {
	case arteHeader, arteHero, arteLogo, arteCover:
		return t, true
	}
	return "", false
}

// nomeNoCache mantém o nome antigo da capa horizontal ({id}.jpg).
func (t tipoArte) nomeNoCache(id int64) string {
	n := strconv.FormatInt(id, 10)
	switch t {
	case arteHero:
		return n + "-hero.jpg"
	case arteLogo:
		return n + "-logo.png"
	case arteCover:
		return n + "-cover.jpg"
	}
	return n + ".jpg"
}

func (t tipoArte) arquivoSteam() string {
	switch t {
	case arteHero:
		return "library_hero.jpg"
	case arteLogo:
		return "logo.png"
	case arteCover:
		return "library_600x900.jpg"
	}
	return "header.jpg"
}

func (t tipoArte) caminhoSGDB(jogo uint64) string {
	switch t {
	case arteHero:
		return fmt.Sprintf("heroes/game/%d", jogo)
	case arteLogo:
		return fmt.Sprintf("logos/game/%d", jogo)
	case arteCover:
		return fmt.Sprintf("grids/game/%d?dimensions=600x900", jogo)
	}
	return fmt.Sprintf("grids/game/%d?dimensions=460x215,920x430", jogo)
}

func urlArteSteam(steamID uint64, t tipoArte) string {
	return fmt.Sprintf("https://cdn.cloudflare.steamstatic.com/steam/apps/%d/%s", steamID, t.arquivoSteam())
}

// codificarBusca mantém letras, números e -_.~; o resto vira %XX.
func codificarBusca(texto string) string {
	var out strings.Builder
	for _, b := range []byte(strings.TrimSpace(texto)) {
		if b < 128 && (unicode.IsLetter(rune(b)) || unicode.IsDigit(rune(b)) || strings.IndexByte("-_.~", b) >= 0) {
			out.WriteByte(b)
		} else {
			fmt.Fprintf(&out, "%%%02X", b)
		}
	}
	return out.String()
}

func normalizarNome(s string) string {
	var out strings.Builder
	for _, c := range s {
		if unicode.IsLetter(c) || unicode.IsDigit(c) || unicode.IsNumber(c) {
			out.WriteString(strings.ToLower(string(c)))
		}
	}
	return out.String()
}

// idDaBuscaDaLoja aceita só o nome igual, ignorando espaços, maiúsculas e pontuação
// ("RogueLegacy" = "Rogue Legacy", mas não "Rogue Legacy 2").
func idDaBuscaDaLoja(corpo []byte, nome string) (uint64, bool) {
	var r struct {
		Items []struct {
			Type string `json:"type"`
			Name string `json:"name"`
			ID   uint64 `json:"id"`
		} `json:"items"`
	}
	if json.Unmarshal(corpo, &r) != nil {
		return 0, false
	}
	alvo := normalizarNome(nome)
	for _, i := range r.Items {
		if i.Type == "app" && normalizarNome(i.Name) == alvo && i.ID > 0 {
			return i.ID, true
		}
	}
	return 0, false
}

type resultadoBusca int

const (
	buscaAchou resultadoBusca = iota
	buscaNada
	buscaFalhou
)

var clienteArte = &http.Client{Timeout: 10 * time.Second}

func lerCorpo(resp *http.Response, limite int64) ([]byte, bool) {
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, false
	}
	corpo, err := io.ReadAll(io.LimitReader(resp.Body, limite))
	return corpo, err == nil
}

// buscarNaLoja procura o jogo na loja da Steam (sem chave). Offline ou com a loja fora do ar,
// devolve buscaFalhou para tentar de novo na próxima vez.
func buscarNaLoja(nome string) (uint64, resultadoBusca) {
	if strings.TrimSpace(nome) == "" {
		return 0, buscaNada
	}
	resp, err := clienteArte.Get("https://store.steampowered.com/api/storesearch/?term=" + codificarBusca(nome) + "&l=portuguese&cc=BR")
	if err != nil {
		return 0, buscaFalhou
	}
	corpo, ok := lerCorpo(resp, 1<<20)
	if !ok {
		return 0, buscaFalhou
	}
	var teste map[string]any
	if json.Unmarshal(corpo, &teste) != nil {
		return 0, buscaFalhou
	}
	if id, ok := idDaBuscaDaLoja(corpo, nome); ok {
		return id, buscaAchou
	}
	return 0, buscaNada
}

func pegarSGDB(url, chave string) []byte {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("Authorization", "Bearer "+chave)
	resp, err := clienteArte.Do(req)
	if err != nil {
		return nil
	}
	corpo, ok := lerCorpo(resp, 2<<20)
	if !ok {
		return nil
	}
	return corpo
}

func primeiroIDSGDB(corpo []byte) (uint64, bool) {
	var r struct {
		Data []struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	if json.Unmarshal(corpo, &r) != nil {
		return 0, false
	}
	for _, d := range r.Data {
		if d.ID > 0 {
			return d.ID, true
		}
	}
	return 0, false
}

func primeiraURLSGDB(corpo []byte) (string, bool) {
	var r struct {
		Data []struct {
			URL string `json:"url"`
		} `json:"data"`
	}
	if json.Unmarshal(corpo, &r) != nil {
		return "", false
	}
	for _, d := range r.Data {
		if d.URL != "" {
			return d.URL, strings.HasPrefix(d.URL, "https://")
		}
	}
	return "", false
}

// arteSteamGridDB busca no SteamGridDB a primeira arte do tipo pedido para o nome do jogo.
func arteSteamGridDB(chave, nome string, t tipoArte) (string, bool) {
	chave = strings.TrimSpace(chave)
	if chave == "" || strings.TrimSpace(nome) == "" {
		return "", false
	}
	busca := pegarSGDB(sgdb+"/search/autocomplete/"+codificarBusca(nome), chave)
	if busca == nil {
		return "", false
	}
	id, ok := primeiroIDSGDB(busca)
	if !ok {
		return "", false
	}
	grades := pegarSGDB(sgdb+"/"+t.caminhoSGDB(id), chave)
	if grades == nil {
		return "", false
	}
	return primeiraURLSGDB(grades)
}

// baixarArte baixa a imagem para dest por troca atômica. Só aceita JPEG ou PNG.
func baixarArte(urlImagem, dest string) (string, bool) {
	resp, err := clienteArte.Get(urlImagem)
	if err != nil {
		return "", false
	}
	corpo, ok := lerCorpo(resp, 5<<20)
	if !ok {
		return "", false
	}
	if !bytes.HasPrefix(corpo, []byte{0xFF, 0xD8, 0xFF}) && !bytes.HasPrefix(corpo, []byte{0x89, 'P', 'N', 'G'}) {
		return "", false
	}
	if os.MkdirAll(filepath.Dir(dest), 0755) != nil {
		return "", false
	}
	tmp := dest + ".tmp"
	if os.WriteFile(tmp, corpo, 0644) != nil {
		return "", false
	}
	if os.Rename(tmp, dest) != nil {
		_ = os.Remove(tmp)
		return "", false
	}
	return dest, true
}

// idSteamDoJogo é o id do catálogo; sem ele, o da busca da loja pelo nome exato. O resultado da
// busca (também "não achei", como 0) fica guardado para não repetir a consulta.
func idSteamDoJogo(lib *library.Library, game library.Game) (*uint64, error) {
	dados := pastaDados()
	if game.CatalogID != nil {
		if cat, err := catalog.Open(filepath.Join(dados, "catalog.db")); err == nil {
			defer cat.Close()
			if steam, err := cat.SteamID(*game.CatalogID); err != nil {
				return nil, err
			} else if steam != nil {
				return steam, nil
			}
		}
	}
	chave := fmt.Sprintf("steam_appid:%d", game.ID)
	if v, err := lib.Setting(chave); err != nil {
		return nil, err
	} else if v != nil && *v != "" {
		if id, err := strconv.ParseUint(*v, 10, 64); err == nil && id > 0 {
			return &id, nil
		}
		return nil, nil
	}
	id, resultado := buscarNaLoja(game.Name)
	switch resultado {
	case buscaAchou:
		_ = lib.SetSetting(chave, strconv.FormatUint(id, 10))
		return &id, nil
	case buscaNada:
		_ = lib.SetSetting(chave, "0")
	}
	return nil, nil
}

func arteDoJogo(id int64, t tipoArte) (*string, error) {
	lib, err := abrirBiblioteca()
	if err != nil {
		return nil, err
	}
	defer lib.Close()
	game, err := lib.Get(id)
	if err != nil {
		return nil, err
	}
	cache := filepath.Join(pastaDados(), "art", t.nomeNoCache(id))
	if info, err := os.Stat(cache); err == nil && info.Mode().IsRegular() {
		return &cache, nil
	}
	steam, err := idSteamDoJogo(lib, game)
	if err != nil {
		return nil, err
	}
	if steam != nil {
		if p, ok := baixarArte(urlArteSteam(*steam, t), cache); ok {
			return &p, nil
		}
	}
	chave, err := lib.Setting("steamgriddb_api_key")
	if err != nil {
		return nil, err
	}
	if chave == nil {
		return nil, nil
	}
	if u, ok := arteSteamGridDB(*chave, game.Name, t); ok {
		if p, ok := baixarArte(u, cache); ok {
			return &p, nil
		}
	}
	return nil, nil
}

// CapaDoJogo devolve a capa horizontal em cache, ou nada.
func (App) CapaDoJogo(id int64) (*string, error) { return arteDoJogo(id, arteHeader) }

// ArteDoJogo devolve a arte de um tipo (header, hero, logo ou cover) em cache, ou nada.
func (App) ArteDoJogo(id int64, tipo string) (*string, error) {
	t, ok := analisarTipoArte(tipo)
	if !ok {
		return nil, fmt.Errorf("Tipo de arte desconhecido: %s", tipo)
	}
	return arteDoJogo(id, t)
}
