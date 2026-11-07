//! Conquistas do HUB: marcos que valem para qualquer jogo, calculados a partir do uso real.
//! Horas contam o tempo de jogo; a sequência conta sincronizações concluídas sem erro.

use serde::Serialize;

#[derive(Debug, Clone, Serialize)]
pub struct HubAchievement {
    pub id: String,
    pub nome: String,
    pub descricao: String,
    pub nivel: String,
    pub progresso: i64,
    pub alvo: i64,
    pub unidade: String,
    pub desbloqueada: bool,
}

#[derive(Debug, Clone, Serialize)]
pub struct HubAchievements {
    pub conquistas: Vec<HubAchievement>,
    pub desbloqueadas: usize,
    pub total: usize,
    pub horas: i64,
    pub jogos: i64,
    pub sequencia_atual: i64,
    pub melhor_sequencia: i64,
    pub jogos_completos: i64,
}

#[derive(Debug, Clone, Copy)]
pub struct Uso {
    pub playtime_secs: i64,
    pub jogos: i64,
    pub sequencia: i64,
    pub melhor_sequencia: i64,
    pub jogos_completos: i64,
}

enum Tipo {
    Horas,
    Jogos,
    Sequencia,
    Completos,
}

struct Marco {
    id: &'static str,
    nome: &'static str,
    descricao: &'static str,
    nivel: &'static str,
    alvo: i64,
    unidade: &'static str,
    tipo: Tipo,
}

const MARCOS: &[Marco] = &[
    Marco {
        id: "primeira-hora",
        nome: "Primeira hora",
        descricao: "Jogue por 1 hora no total.",
        nivel: "bronze",
        alvo: 1,
        unidade: "h",
        tipo: Tipo::Horas,
    },
    Marco {
        id: "dez-horas",
        nome: "Dez horas de jogo",
        descricao: "Jogue por 10 horas no total.",
        nivel: "prata",
        alvo: 10,
        unidade: "h",
        tipo: Tipo::Horas,
    },
    Marco {
        id: "cinquenta-horas",
        nome: "Cinquenta horas",
        descricao: "Jogue por 50 horas no total.",
        nivel: "ouro",
        alvo: 50,
        unidade: "h",
        tipo: Tipo::Horas,
    },
    Marco {
        id: "duzentas-horas",
        nome: "Duzentas horas",
        descricao: "Jogue por 200 horas no total.",
        nivel: "ouro",
        alvo: 200,
        unidade: "h",
        tipo: Tipo::Horas,
    },
    Marco {
        id: "primeiro-jogo",
        nome: "Primeiro jogo",
        descricao: "Adicione 1 jogo à biblioteca.",
        nivel: "bronze",
        alvo: 1,
        unidade: "jogo",
        tipo: Tipo::Jogos,
    },
    Marco {
        id: "cinco-jogos",
        nome: "Coleção começando",
        descricao: "Adicione 5 jogos à biblioteca.",
        nivel: "prata",
        alvo: 5,
        unidade: "jogos",
        tipo: Tipo::Jogos,
    },
    Marco {
        id: "vinte-jogos",
        nome: "Biblioteca cheia",
        descricao: "Adicione 20 jogos à biblioteca.",
        nivel: "ouro",
        alvo: 20,
        unidade: "jogos",
        tipo: Tipo::Jogos,
    },
    Marco {
        id: "sequencia-5",
        nome: "Ritmo constante",
        descricao: "Conclua 5 sincronizações seguidas sem erro.",
        nivel: "bronze",
        alvo: 5,
        unidade: "syncs",
        tipo: Tipo::Sequencia,
    },
    Marco {
        id: "sequencia-25",
        nome: "Tudo em dia",
        descricao: "Conclua 25 sincronizações seguidas sem erro.",
        nivel: "prata",
        alvo: 25,
        unidade: "syncs",
        tipo: Tipo::Sequencia,
    },
    Marco {
        id: "sequencia-100",
        nome: "Sem falhar",
        descricao: "Conclua 100 sincronizações seguidas sem erro.",
        nivel: "ouro",
        alvo: 100,
        unidade: "syncs",
        tipo: Tipo::Sequencia,
    },
    Marco {
        id: "primeiro-completo",
        nome: "Jogo completo",
        descricao: "Desbloqueie todas as conquistas da Steam de um jogo.",
        nivel: "prata",
        alvo: 1,
        unidade: "jogo",
        tipo: Tipo::Completos,
    },
    Marco {
        id: "cinco-completos",
        nome: "Caçador de conquistas",
        descricao: "Complete as conquistas da Steam de 5 jogos.",
        nivel: "ouro",
        alvo: 5,
        unidade: "jogos",
        tipo: Tipo::Completos,
    },
];

pub fn calcular(uso: Uso) -> HubAchievements {
    let horas = uso.playtime_secs / 3600;
    let mut conquistas: Vec<HubAchievement> = MARCOS
        .iter()
        .map(|m| {
            let bruto = match m.tipo {
                Tipo::Horas => horas,
                Tipo::Jogos => uso.jogos,
                Tipo::Sequencia => uso.sequencia,
                Tipo::Completos => uso.jogos_completos,
            };
            HubAchievement {
                id: m.id.into(),
                nome: m.nome.into(),
                descricao: m.descricao.into(),
                nivel: m.nivel.into(),
                progresso: bruto.min(m.alvo),
                alvo: m.alvo,
                unidade: m.unidade.into(),
                desbloqueada: bruto >= m.alvo,
            }
        })
        .collect();
    conquistas.sort_by_key(|c| (!c.desbloqueada, c.alvo));
    let desbloqueadas = conquistas.iter().filter(|c| c.desbloqueada).count();
    HubAchievements {
        total: conquistas.len(),
        desbloqueadas,
        conquistas,
        horas,
        jogos: uso.jogos,
        sequencia_atual: uso.sequencia,
        melhor_sequencia: uso.melhor_sequencia,
        jogos_completos: uso.jogos_completos,
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn calcula_progresso_e_desbloqueia_os_marcos_alcancados() {
        let h = calcular(Uso {
            playtime_secs: 10 * 3600 + 1800,
            jogos: 5,
            sequencia: 25,
            melhor_sequencia: 30,
            jogos_completos: 1,
        });
        assert_eq!(h.horas, 10);
        assert_eq!(h.total, 12);
        assert_eq!(h.desbloqueadas, 7);
        let primeira = h.conquistas.iter().find(|c| c.id == "primeira-hora").unwrap();
        assert!(primeira.desbloqueada);
        assert_eq!(primeira.progresso, 1);
        let sequencia = h.conquistas.iter().find(|c| c.id == "sequencia-100").unwrap();
        assert!(!sequencia.desbloqueada);
        assert_eq!(sequencia.progresso, 25);
        assert_eq!(h.sequencia_atual, 25);
        assert_eq!(h.melhor_sequencia, 30);
        assert_eq!(h.jogos_completos, 1);
    }

    #[test]
    fn sem_uso_nada_fica_desbloqueado() {
        let h = calcular(Uso { playtime_secs: 0, jogos: 0, sequencia: 0, melhor_sequencia: 0, jogos_completos: 0 });
        assert_eq!(h.desbloqueadas, 0);
        assert!(h.conquistas.iter().all(|c| !c.desbloqueada && c.progresso == 0));
    }

    #[test]
    fn lista_mostra_primeiro_o_que_falta_desbloquear() {
        let h = calcular(Uso { playtime_secs: 3600, jogos: 1, sequencia: 0, melhor_sequencia: 0, jogos_completos: 0 });
        assert!(h.conquistas[0].desbloqueada);
        assert_eq!(h.desbloqueadas, 2);
    }
}
