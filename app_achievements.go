package main

import (
	"encoding/json"
	"errors"
	"strings"

	"sync-game-save/internal/achievements"
)

// VistaConquistas tem o mesmo formato JSON da versão Rust.
type VistaConquistas struct {
	Set       achievements.Set     `json:"set"`
	Summary   achievements.Summary `json:"summary"`
	FromCache bool                 `json:"from_cache"`
	FetchedAt int64                `json:"fetched_at"`
	// Notice é o motivo para mostrar dados incompletos ou antigos, em linguagem simples.
	Notice *string `json:"notice"`
}

func novaVista(set achievements.Set, em int64, cache bool, aviso *string) VistaConquistas {
	return VistaConquistas{Set: set, Summary: set.Summary(), FromCache: cache, FetchedAt: em, Notice: aviso}
}

// DefinirCredenciaisSteam guarda a chave de API e o SteamID do usuário. A chave nunca volta
// para a interface.
func (App) DefinirCredenciaisSteam(chave, steamID string) error {
	chave, steamID = strings.TrimSpace(chave), strings.TrimSpace(steamID)
	if steamID != "" && strings.Trim(steamID, "0123456789") != "" {
		return errors.New("O SteamID deve ter só números (o \"SteamID64\", com 17 dígitos).")
	}
	lib, err := abrirBiblioteca()
	if err != nil {
		return err
	}
	defer lib.Close()
	if err := lib.SetSetting("steam_api_key", chave); err != nil {
		return err
	}
	return lib.SetSetting("steam_id", steamID)
}

// DefinirChaveSteamGridDB guarda a chave do SteamGridDB, usada só para buscar capas.
// Vazia remove a chave.
func (App) DefinirChaveSteamGridDB(chave string) error {
	lib, err := abrirBiblioteca()
	if err != nil {
		return err
	}
	defer lib.Close()
	return lib.SetSetting("steamgriddb_api_key", strings.TrimSpace(chave))
}

// Conquistas devolve as conquistas do jogo. Usa o cache se foi buscado há menos de 10 minutos
// (a menos que forcar) e, sem internet ou com erro, devolve o que ficou guardado com o motivo.
func (App) Conquistas(id int64, forcar *bool) (VistaConquistas, error) {
	return conquistasDoJogo(id, forcar != nil && *forcar, achievements.NetHTTP{}, int64(agora()))
}

func conquistasDoJogo(id int64, forcar bool, http achievements.HTTP, agora int64) (VistaConquistas, error) {
	const validade = 600
	lib, err := abrirBiblioteca()
	if err != nil {
		return VistaConquistas{}, err
	}
	defer lib.Close()
	game, err := lib.Get(id)
	if err != nil {
		return VistaConquistas{}, err
	}
	var guardado *achievements.Set
	var guardadoEm int64
	cache, err := lib.AchievementsCache(id)
	if err != nil {
		return VistaConquistas{}, err
	}
	if cache != nil {
		var set achievements.Set
		if json.Unmarshal([]byte(cache.JSON), &set) == nil {
			guardado, guardadoEm = &set, cache.FetchedAt
		}
	}
	if guardado != nil && !forcar && agora-guardadoEm < validade {
		return novaVista(*guardado, guardadoEm, true, nil), nil
	}
	steamApp, err := idSteamDoJogo(lib, game)
	if err != nil {
		return VistaConquistas{}, err
	}
	chave := ""
	if v, err := lib.Setting("steam_api_key"); err != nil {
		return VistaConquistas{}, err
	} else if v != nil {
		chave = *v
	}
	steamID := ""
	if v, err := lib.Setting("steam_id"); err != nil {
		return VistaConquistas{}, err
	} else if v != nil && strings.TrimSpace(*v) != "" {
		steamID = *v
	} else if local := localSteamID(); local != nil {
		steamID = *local
	}
	jogo := achievements.GameRef{SteamAppID: steamApp}
	// Sem chave (o normal), lê o perfil público da Steam. A chave antiga, se existir, ainda é usada.
	var provedor achievements.Provider
	if strings.TrimSpace(chave) == "" {
		provedor = achievements.CommunityProvider{SteamID: steamID, HTTP: http}
	} else {
		provedor = achievements.SteamProvider{APIKey: chave, SteamID: steamID, HTTP: http}
	}
	set, err := provedor.Fetch(jogo)
	if err != nil {
		if guardado != nil {
			aviso := err.Error()
			return novaVista(*guardado, guardadoEm, true, &aviso), nil
		}
		return VistaConquistas{}, err
	}
	if bruto, err := json.Marshal(set); err == nil {
		if err := lib.SaveAchievements(id, string(bruto), agora); err != nil {
			return VistaConquistas{}, err
		}
	}
	var aviso *string
	if !set.ProgressKnown {
		msg := achievements.ErrPrivateProfile.Error()
		aviso = &msg
	}
	return novaVista(set, agora, false, aviso), nil
}
