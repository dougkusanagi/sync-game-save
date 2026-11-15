package main

import (
	"encoding/json"
	"sort"
	"strconv"

	"sync-game-save/internal/achievements"
)

// Conquistas do HUB: marcos que valem para qualquer jogo, calculados a partir do uso real.
// Horas contam o tempo de jogo; a sequência conta sincronizações concluídas sem erro.

type ConquistaHub struct {
	ID           string `json:"id"`
	Nome         string `json:"nome"`
	Descricao    string `json:"descricao"`
	Nivel        string `json:"nivel"`
	Progresso    int64  `json:"progresso"`
	Alvo         int64  `json:"alvo"`
	Unidade      string `json:"unidade"`
	Desbloqueada bool   `json:"desbloqueada"`
}

type ConquistasHub struct {
	Conquistas      []ConquistaHub `json:"conquistas"`
	Desbloqueadas   int            `json:"desbloqueadas"`
	Total           int            `json:"total"`
	Horas           int64          `json:"horas"`
	Jogos           int64          `json:"jogos"`
	SequenciaAtual  int64          `json:"sequencia_atual"`
	MelhorSequencia int64          `json:"melhor_sequencia"`
	JogosCompletos  int64          `json:"jogos_completos"`
}

type usoHub struct{ playtimeSecs, jogos, sequencia, melhorSequencia, jogosCompletos int64 }

type tipoMarco int

const (
	marcoHoras tipoMarco = iota
	marcoJogos
	marcoSequencia
	marcoCompletos
)

type marcoHub struct {
	id, nome, descricao, nivel string
	alvo                       int64
	unidade                    string
	tipo                       tipoMarco
}

var marcosHub = []marcoHub{
	{"primeira-hora", "Primeira hora", "Jogue por 1 hora no total.", "bronze", 1, "h", marcoHoras},
	{"dez-horas", "Dez horas de jogo", "Jogue por 10 horas no total.", "prata", 10, "h", marcoHoras},
	{"cinquenta-horas", "Cinquenta horas", "Jogue por 50 horas no total.", "ouro", 50, "h", marcoHoras},
	{"duzentas-horas", "Duzentas horas", "Jogue por 200 horas no total.", "ouro", 200, "h", marcoHoras},
	{"primeiro-jogo", "Primeiro jogo", "Adicione 1 jogo à biblioteca.", "bronze", 1, "jogo", marcoJogos},
	{"cinco-jogos", "Coleção começando", "Adicione 5 jogos à biblioteca.", "prata", 5, "jogos", marcoJogos},
	{"vinte-jogos", "Biblioteca cheia", "Adicione 20 jogos à biblioteca.", "ouro", 20, "jogos", marcoJogos},
	{"sequencia-5", "Ritmo constante", "Conclua 5 sincronizações seguidas sem erro.", "bronze", 5, "syncs", marcoSequencia},
	{"sequencia-25", "Tudo em dia", "Conclua 25 sincronizações seguidas sem erro.", "prata", 25, "syncs", marcoSequencia},
	{"sequencia-100", "Sem falhar", "Conclua 100 sincronizações seguidas sem erro.", "ouro", 100, "syncs", marcoSequencia},
	{"primeiro-completo", "Jogo completo", "Desbloqueie todas as conquistas da Steam de um jogo.", "prata", 1, "jogo", marcoCompletos},
	{"cinco-completos", "Caçador de conquistas", "Complete as conquistas da Steam de 5 jogos.", "ouro", 5, "jogos", marcoCompletos},
}

func calcularHub(uso usoHub) ConquistasHub {
	horas := uso.playtimeSecs / 3600
	lista := make([]ConquistaHub, 0, len(marcosHub))
	for _, m := range marcosHub {
		var bruto int64
		switch m.tipo {
		case marcoHoras:
			bruto = horas
		case marcoJogos:
			bruto = uso.jogos
		case marcoSequencia:
			bruto = uso.sequencia
		case marcoCompletos:
			bruto = uso.jogosCompletos
		}
		lista = append(lista, ConquistaHub{
			ID: m.id, Nome: m.nome, Descricao: m.descricao, Nivel: m.nivel,
			Progresso: min(bruto, m.alvo), Alvo: m.alvo, Unidade: m.unidade, Desbloqueada: bruto >= m.alvo,
		})
	}
	// O que falta desbloquear vem depois; dentro de cada grupo, o menor alvo primeiro.
	sort.SliceStable(lista, func(i, j int) bool {
		a, b := lista[i], lista[j]
		if a.Desbloqueada != b.Desbloqueada {
			return a.Desbloqueada
		}
		return a.Alvo < b.Alvo
	})
	desbloqueadas := 0
	for _, c := range lista {
		if c.Desbloqueada {
			desbloqueadas++
		}
	}
	return ConquistasHub{
		Conquistas: lista, Desbloqueadas: desbloqueadas, Total: len(lista), Horas: horas, Jogos: uso.jogos,
		SequenciaAtual: uso.sequencia, MelhorSequencia: uso.melhorSequencia, JogosCompletos: uso.jogosCompletos,
	}
}

// ConquistasHub calcula os marcos de uso a partir da biblioteca e do histórico de sincronizações.
func (App) ConquistasHub() (ConquistasHub, error) {
	lib, err := abrirBiblioteca()
	if err != nil {
		return ConquistasHub{}, err
	}
	defer lib.Close()
	jogos, err := lib.List()
	if err != nil {
		return ConquistasHub{}, err
	}
	uso := usoHub{jogos: int64(len(jogos))}
	ler := func(chave string) int64 {
		v, err := lib.Setting(chave)
		if err != nil || v == nil {
			return 0
		}
		n, _ := strconv.ParseInt(*v, 10, 64)
		return n
	}
	uso.sequencia, uso.melhorSequencia = ler("sync_ok_streak"), ler("sync_best_streak")
	for _, g := range jogos {
		uso.playtimeSecs += g.PlaytimeSecs
		cache, err := lib.AchievementsCache(g.ID)
		if err != nil || cache == nil {
			continue
		}
		var set achievements.Set
		if json.Unmarshal([]byte(cache.JSON), &set) == nil && set.Summary().Complete {
			uso.jogosCompletos++
		}
	}
	return calcularHub(uso), nil
}
