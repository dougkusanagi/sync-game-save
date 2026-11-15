// Package achievements reúne provedores de conquistas da Steam (perfil público, sem chave; ou
// chave de API) e o cálculo de níveis por raridade. A rede é injetada, para testar sem chave
// nem internet. O JSON dos tipos é o mesmo da versão Rust, porque o cache da biblioteca o guarda.
package achievements

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Erros já escritos para o usuário final.
var (
	ErrNoCredentials  = errors.New("Não achei a sua conta da Steam neste PC. Entre na Steam neste PC (ou informe o seu SteamID64 em Configurações) para ver as conquistas.")
	ErrPrivateProfile = errors.New("O seu perfil da Steam não está público, então não consigo ver o que você desbloqueou. Abra o perfil na Steam, vá em Privacidade e deixe os detalhes dos jogos como Públicos. Enquanto isso, mostro só a lista de conquistas do jogo.")
	ErrNoAchievements = errors.New("Este jogo não tem conquistas na Steam.")
	ErrBadKey         = errors.New("A Steam recusou a chave de API. Confira se ela foi copiada inteira.")
	ErrBadResponse    = errors.New("A Steam respondeu num formato que não reconheço.")
)

func networkError(detail string) error {
	return fmt.Errorf("Não consegui falar com a Steam agora (%s). Mostro o que ficou guardado da última vez.", detail)
}

// Tier é a raridade global, calculada aqui (a Steam não tem níveis).
type Tier string

const (
	Common    Tier = "Common"
	Uncommon  Tier = "Uncommon"
	Rare      Tier = "Rare"
	UltraRare Tier = "UltraRare"
)

// TierFromPercent: Comum (> 50%), Incomum (20–50%), Rara (5–20%), Ultrarrara (< 5%).
func TierFromPercent(p float32) Tier {
	switch {
	case p > 50:
		return Common
	case p >= 20:
		return Uncommon
	case p >= 5:
		return Rare
	}
	return UltraRare
}

func (t Tier) Label() string {
	switch t {
	case Common:
		return "Comum"
	case Uncommon:
		return "Incomum"
	case Rare:
		return "Rara"
	}
	return "Ultrarrara"
}

type Achievement struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Icon        string   `json:"icon"`
	IconGray    string   `json:"icon_gray"`
	Hidden      bool     `json:"hidden"`
	Unlocked    bool     `json:"unlocked"`
	UnlockTime  *uint64  `json:"unlock_time"`
	Percent     *float32 `json:"percent"`
	Tier        *Tier    `json:"tier"`
}

type Set struct {
	Achievements []Achievement `json:"achievements"`
	// ProgressKnown é false quando só o catálogo está disponível (perfil privado).
	ProgressKnown bool `json:"progress_known"`
}

type Summary struct {
	Unlocked int            `json:"unlocked"`
	Total    int            `json:"total"`
	Percent  uint32         `json:"percent"`
	Complete bool           `json:"complete"`
	ByTier   map[string]int `json:"by_tier"`
}

// Summary calcula o progresso. Conquistas sem raridade conhecida ficam fora da contagem por nível.
func (s Set) Summary() Summary {
	out := Summary{Total: len(s.Achievements), ByTier: map[string]int{}}
	for _, a := range s.Achievements {
		if !a.Unlocked {
			continue
		}
		out.Unlocked++
		if a.Tier != nil {
			out.ByTier[a.Tier.Label()]++
		}
	}
	if out.Total > 0 {
		out.Percent = uint32(out.Unlocked * 100 / out.Total)
	}
	out.Complete = out.Total > 0 && out.Unlocked == out.Total
	return out
}

// GameRef é o que um provedor precisa para achar as conquistas de um jogo.
type GameRef struct{ SteamAppID *uint64 }

type Provider interface {
	ID() string
	Fetch(game GameRef) (Set, error)
}

// HTTP é o acesso à rede injetável: devolve o código de status e o corpo.
type HTTP interface {
	Get(url string) (int, string, error)
}

type NetHTTP struct{}

func (NetHTTP) Get(url string) (int, string, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		// A URL leva a chave: não vaza na mensagem de erro.
		return 0, "", errors.New(strings.ReplaceAll(err.Error(), url, "<url>"))
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 5<<20))
	if err != nil {
		return 0, "", err
	}
	return resp.StatusCode, string(body), nil
}

const base = "https://api.steampowered.com"

type SteamProvider struct {
	APIKey, SteamID string
	HTTP            HTTP
}

func (p SteamProvider) ID() string { return "steam" }

func (p SteamProvider) get(url string) (int, string, error) {
	status, body, err := p.HTTP.Get(url)
	if err != nil {
		key := strings.TrimSpace(p.APIKey)
		msg := err.Error()
		if key != "" {
			msg = strings.ReplaceAll(msg, key, "***")
		}
		msg = strings.ReplaceAll(msg, p.APIKey, "***")
		return 0, "", networkError(msg)
	}
	return status, body, nil
}

func checkStatus(s int) error {
	switch s {
	case 200:
		return nil
	case 401, 403:
		return ErrBadKey
	}
	return networkError(fmt.Sprintf("resposta HTTP %d", s))
}

func applyPercentages(list []Achievement, p map[string]float32, lower bool) {
	for i := range list {
		id := list[i].ID
		if lower {
			id = strings.ToLower(id)
		}
		if v, ok := p[id]; ok {
			v := v
			tier := TierFromPercent(v)
			list[i].Percent, list[i].Tier = &v, &tier
		}
	}
}

func (p SteamProvider) Fetch(game GameRef) (Set, error) {
	if strings.TrimSpace(p.APIKey) == "" || strings.TrimSpace(p.SteamID) == "" {
		return Set{}, ErrNoCredentials
	}
	if game.SteamAppID == nil {
		return Set{}, ErrNoAchievements
	}
	app, key, steam := *game.SteamAppID, strings.TrimSpace(p.APIKey), strings.TrimSpace(p.SteamID)

	// 1. Catálogo (nome, descrição, ícones, oculta).
	status, body, err := p.get(fmt.Sprintf("%s/ISteamUserStats/GetSchemaForGame/v2/?key=%s&appid=%d&l=brazilian", base, key, app))
	if err != nil {
		return Set{}, err
	}
	if err := checkStatus(status); err != nil {
		return Set{}, err
	}
	list, err := ParseSchema(body)
	if err != nil {
		return Set{}, err
	}
	if len(list) == 0 {
		return Set{}, ErrNoAchievements
	}

	// 2. Raridade global (não precisa de perfil público; se falhar, segue sem níveis).
	if status, body, err := p.get(fmt.Sprintf("%s/ISteamUserStats/GetGlobalAchievementPercentagesForApp/v2/?gameid=%d", base, app)); err == nil && status == 200 {
		if pct, err := ParsePercentages(body); err == nil {
			applyPercentages(list, pct, false)
		}
	}

	// 3. O que o usuário desbloqueou (exige perfil de jogos público).
	status, body, err = p.get(fmt.Sprintf("%s/ISteamUserStats/GetPlayerAchievements/v1/?key=%s&steamid=%s&appid=%d", base, key, steam, app))
	if err != nil {
		return Set{}, err
	}
	switch status {
	case 200:
	case 403:
		return Set{Achievements: list}, nil
	default:
		if err := checkStatus(status); err != nil {
			return Set{}, err
		}
	}
	done, err := ParsePlayer(body)
	if errors.Is(err, ErrPrivateProfile) {
		return Set{Achievements: list}, nil
	}
	if err != nil {
		return Set{}, err
	}
	for i := range list {
		if t, ok := done[list[i].ID]; ok {
			list[i].Unlocked = true
			if t > 0 {
				t := t
				list[i].UnlockTime = &t
			}
		}
	}
	return Set{Achievements: list, ProgressKnown: true}, nil
}

func ParseSchema(body string) ([]Achievement, error) {
	var raiz struct {
		Game *struct {
			Stats *struct {
				Achievements []struct {
					Name        string `json:"name"`
					DisplayName string `json:"displayName"`
					Description string `json:"description"`
					Icon        string `json:"icon"`
					IconGray    string `json:"icongray"`
					Hidden      uint8  `json:"hidden"`
				} `json:"achievements"`
			} `json:"availableGameStats"`
		} `json:"game"`
	}
	if err := json.Unmarshal([]byte(body), &raiz); err != nil {
		return nil, ErrBadResponse
	}
	out := []Achievement{}
	if raiz.Game == nil || raiz.Game.Stats == nil {
		return out, nil
	}
	for _, i := range raiz.Game.Stats.Achievements {
		name := i.DisplayName
		if name == "" {
			name = i.Name
		}
		out = append(out, Achievement{ID: i.Name, Name: name, Description: i.Description, Icon: i.Icon, IconGray: i.IconGray, Hidden: i.Hidden == 1})
	}
	return out, nil
}

// ParsePlayer devolve as conquistas desbloqueadas: id → hora do desbloqueio
// (segundos desde 1970; 0 se desconhecida).
func ParsePlayer(body string) (map[string]uint64, error) {
	var raiz struct {
		Playerstats *struct {
			Success      bool   `json:"success"`
			Error        string `json:"error"`
			Achievements []struct {
				APIName    string `json:"apiname"`
				Achieved   uint8  `json:"achieved"`
				UnlockTime uint64 `json:"unlocktime"`
			} `json:"achievements"`
		} `json:"playerstats"`
	}
	if err := json.Unmarshal([]byte(body), &raiz); err != nil || raiz.Playerstats == nil {
		return nil, ErrBadResponse
	}
	p := raiz.Playerstats
	if !p.Success {
		erro := strings.ToLower(p.Error)
		if strings.Contains(erro, "not public") || strings.Contains(erro, "private") {
			return nil, ErrPrivateProfile
		}
		return nil, ErrBadResponse
	}
	out := map[string]uint64{}
	for _, a := range p.Achievements {
		if a.Achieved == 1 {
			out[a.APIName] = a.UnlockTime
		}
	}
	return out, nil
}

// ParsePercentages devolve a porcentagem global por conquista. A Steam manda número ou texto.
func ParsePercentages(body string) (map[string]float32, error) {
	var raiz struct {
		Lista *struct {
			Achievements []struct {
				Name    string          `json:"name"`
				Percent json.RawMessage `json:"percent"`
			} `json:"achievements"`
		} `json:"achievementpercentages"`
	}
	if err := json.Unmarshal([]byte(body), &raiz); err != nil || raiz.Lista == nil {
		return nil, ErrBadResponse
	}
	out := map[string]float32{}
	for _, i := range raiz.Lista.Achievements {
		var v float64
		raw := strings.TrimSpace(string(i.Percent))
		if strings.HasPrefix(raw, `"`) {
			var s string
			if json.Unmarshal(i.Percent, &s) != nil {
				continue
			}
			n, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
			if err != nil {
				continue
			}
			v = n
		} else if err := json.Unmarshal(i.Percent, &v); err != nil {
			continue
		}
		out[i.Name] = float32(v)
	}
	return out, nil
}

// CommunityProvider lê o XML público do perfil
// (steamcommunity.com/profiles/<id>/stats/<app>/achievements/?xml=1) e a raridade global.
type CommunityProvider struct {
	SteamID string
	HTTP    HTTP
	// BaseURL permite apontar para um servidor de teste; vazio usa a Steam.
	BaseURL string
}

func (p CommunityProvider) ID() string { return "steam-community" }

func (p CommunityProvider) Fetch(game GameRef) (Set, error) {
	id := strings.TrimSpace(p.SteamID)
	if id == "" || strings.Trim(id, "0123456789") != "" {
		return Set{}, ErrNoCredentials
	}
	if game.SteamAppID == nil {
		return Set{}, ErrNoAchievements
	}
	app := *game.SteamAppID
	root := p.BaseURL
	if root == "" {
		root = "https://steamcommunity.com"
	}
	status, body, err := p.HTTP.Get(fmt.Sprintf("%s/profiles/%s/stats/%d/achievements/?xml=1&l=brazilian", root, id, app))
	if err != nil {
		return Set{}, networkError(err.Error())
	}
	if status != 200 {
		return Set{}, networkError(fmt.Sprintf("resposta HTTP %d", status))
	}
	set, err := ParseCommunityXML(body)
	if err != nil {
		return Set{}, err
	}
	if status, body, err := p.HTTP.Get(fmt.Sprintf("%s/ISteamUserStats/GetGlobalAchievementPercentagesForApp/v2/?gameid=%d", base, app)); err == nil && status == 200 {
		if pct, err := ParsePercentages(body); err == nil {
			// O XML traz o id em minúsculas; a raridade global usa o id original.
			lower := make(map[string]float32, len(pct))
			for k, v := range pct {
				lower[strings.ToLower(k)] = v
			}
			applyPercentages(set.Achievements, lower, true)
		}
	}
	return set, nil
}

// xmlTag devolve o texto entre <tag> e </tag>, sem o CDATA.
func xmlTag(xml, name string) (string, bool) {
	open := "<" + name + ">"
	ini := strings.Index(xml, open)
	if ini < 0 {
		return "", false
	}
	ini += len(open)
	fim := strings.Index(xml[ini:], "</"+name+">")
	if fim < 0 {
		return "", false
	}
	v := strings.TrimSpace(xml[ini : ini+fim])
	if s, ok := strings.CutPrefix(v, "<![CDATA["); ok {
		if s, ok := strings.CutSuffix(s, "]]>"); ok {
			return s, true
		}
	}
	return v, true
}

func ParseCommunityXML(xml string) (Set, error) {
	if !strings.Contains(xml, "<playerstats>") {
		// Perfil privado ou inexistente: a Steam devolve uma página de erro.
		if strings.Contains(xml, "<error>") || strings.Contains(strings.ToLower(xml), "private") {
			return Set{}, ErrPrivateProfile
		}
		return Set{}, ErrBadResponse
	}
	if p, ok := xmlTag(xml, "privacyState"); ok && p != "public" {
		return Set{}, ErrPrivateProfile
	}
	list := []Achievement{}
	parts := strings.Split(xml, "<achievement ")
	for _, bloco := range parts[1:] {
		if i := strings.Index(bloco, "</achievement>"); i >= 0 {
			bloco = bloco[:i]
		}
		feito := strings.HasPrefix(bloco, `closed="1"`)
		id, _ := xmlTag(bloco, "apiname")
		nome, _ := xmlTag(bloco, "name")
		if id == "" && nome == "" {
			continue
		}
		var quando *uint64
		if t, ok := xmlTag(bloco, "unlockTimestamp"); ok {
			if n, err := strconv.ParseUint(t, 10, 64); err == nil && n > 0 && feito {
				quando = &n
			}
		}
		if id == "" {
			id = nome
		}
		desc, _ := xmlTag(bloco, "description")
		// No XML, iconClosed é o ícone colorido (desbloqueada) e iconOpen o cinza.
		icon, _ := xmlTag(bloco, "iconClosed")
		gray, _ := xmlTag(bloco, "iconOpen")
		list = append(list, Achievement{ID: id, Name: nome, Description: desc, Icon: icon, IconGray: gray, Unlocked: feito, UnlockTime: quando})
	}
	if len(list) == 0 {
		return Set{}, ErrNoAchievements
	}
	return Set{Achievements: list, ProgressKnown: true}, nil
}

// SteamIDFromLoginUsers devolve o SteamID64 da conta usada por último na Steam deste PC,
// lido do loginusers.vdf.
func SteamIDFromLoginUsers(vdf string) string {
	var primeiro, atual string
	for _, linha := range strings.Split(vdf, "\n") {
		t := strings.Trim(strings.TrimSpace(linha), `"`)
		if len(t) == 17 && strings.HasPrefix(t, "7656") && strings.Trim(t, "0123456789") == "" {
			atual = t
			if primeiro == "" {
				primeiro = t
			}
		}
		var campos []string
		for _, c := range strings.Split(linha, `"`) {
			if strings.TrimSpace(c) != "" {
				campos = append(campos, c)
			}
		}
		if len(campos) == 2 && strings.EqualFold(campos[0], "MostRecent") && campos[1] == "1" && atual != "" {
			return atual
		}
	}
	return primeiro
}
