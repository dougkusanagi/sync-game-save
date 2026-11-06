//! Conquistas: interface de provedores, provedor da Steam (chave de API do próprio usuário)
//! e cálculo de níveis por raridade. A rede é injetada, para testar sem chave nem internet.

use serde::{Deserialize, Serialize};
use std::collections::HashMap;

/// Erros já escritos para o usuário final.
#[derive(Debug, thiserror::Error, PartialEq, Eq)]
pub enum Error {
    #[error("Para ver as conquistas da Steam, informe a sua chave de API e o seu SteamID em Configurações.")]
    NoCredentials,
    #[error("O seu perfil da Steam não está público, então não consigo ver o que você desbloqueou. Abra o perfil na Steam, vá em Privacidade e deixe os detalhes dos jogos como Públicos. Enquanto isso, mostro só a lista de conquistas do jogo.")]
    PrivateProfile,
    #[error("Este jogo não tem conquistas na Steam.")]
    NoAchievements,
    #[error("A Steam recusou a chave de API. Confira se ela foi copiada inteira.")]
    BadKey,
    #[error("Não consegui falar com a Steam agora ({0}). Mostro o que ficou guardado da última vez.")]
    Network(String),
    #[error("A Steam respondeu num formato que não reconheço.")]
    BadResponse,
}

pub type Result<T> = std::result::Result<T, Error>;

/// Raridade global, calculada aqui (a Steam não tem níveis).
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize, Hash)]
pub enum Tier {
    Common,
    Uncommon,
    Rare,
    UltraRare,
}

impl Tier {
    /// Comum (> 50%), Incomum (20–50%), Rara (5–20%), Ultrarrara (< 5%).
    pub fn from_percent(p: f32) -> Tier {
        if p > 50.0 {
            Tier::Common
        } else if p >= 20.0 {
            Tier::Uncommon
        } else if p >= 5.0 {
            Tier::Rare
        } else {
            Tier::UltraRare
        }
    }

    pub fn label(self) -> &'static str {
        match self {
            Tier::Common => "Comum",
            Tier::Uncommon => "Incomum",
            Tier::Rare => "Rara",
            Tier::UltraRare => "Ultrarrara",
        }
    }
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct Achievement {
    pub id: String,
    pub name: String,
    pub description: String,
    pub icon: String,
    pub icon_gray: String,
    pub hidden: bool,
    pub unlocked: bool,
    pub unlock_time: Option<u64>,
    /// Porcentagem global de jogadores que desbloquearam.
    pub percent: Option<f32>,
    pub tier: Option<Tier>,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct AchievementSet {
    pub achievements: Vec<Achievement>,
    /// `false` quando só o catálogo está disponível (perfil privado).
    pub progress_known: bool,
}

#[derive(Debug, Clone, PartialEq, Serialize)]
pub struct Summary {
    pub unlocked: usize,
    pub total: usize,
    pub percent: u32,
    pub complete: bool,
    pub by_tier: HashMap<String, usize>,
}

impl AchievementSet {
    /// Progresso (ex.: "12 de 40 · 30%"). Conquistas sem raridade conhecida ficam fora da contagem por nível.
    pub fn summary(&self) -> Summary {
        let total = self.achievements.len();
        let unlocked = self.achievements.iter().filter(|a| a.unlocked).count();
        let mut by_tier: HashMap<String, usize> = HashMap::new();
        for a in self.achievements.iter().filter(|a| a.unlocked) {
            if let Some(t) = a.tier {
                *by_tier.entry(t.label().to_string()).or_default() += 1;
            }
        }
        Summary {
            unlocked,
            total,
            percent: if total == 0 { 0 } else { (unlocked * 100 / total) as u32 },
            complete: total > 0 && unlocked == total,
            by_tier,
        }
    }
}

/// O que um provedor precisa para achar as conquistas de um jogo.
#[derive(Debug, Clone)]
pub struct GameRef {
    pub steam_app_id: Option<u64>,
}

/// Interface para novas fontes (RetroAchievements, conquistas do próprio app...).
pub trait AchievementProvider {
    fn id(&self) -> &'static str;
    fn fetch(&self, game: &GameRef) -> Result<AchievementSet>;
}

/// Acesso HTTP injetável: devolve (código de status, corpo).
pub trait Http {
    fn get(&self, url: &str) -> std::result::Result<(u16, String), String>;
}

pub struct UreqHttp;

impl Http for UreqHttp {
    fn get(&self, url: &str) -> std::result::Result<(u16, String), String> {
        let mut r = ureq::get(url)
            .config()
            .http_status_as_error(false)
            .timeout_global(Some(std::time::Duration::from_secs(10)))
            .build()
            .call()
            // A URL leva a chave: não vaza na mensagem de erro.
            .map_err(|e| e.to_string().replace(url, "<url>"))?;
        let status = r.status().as_u16();
        let body = r.body_mut().with_config().limit(5 * 1024 * 1024).read_to_string().map_err(|e| e.to_string())?;
        Ok((status, body))
    }
}

pub struct SteamProvider<H: Http> {
    pub api_key: String,
    pub steam_id: String,
    pub http: H,
}

const BASE: &str = "https://api.steampowered.com";

impl<H: Http> SteamProvider<H> {
    fn get(&self, url: &str) -> Result<(u16, String)> {
        self.http.get(url).map_err(|e| Error::Network(e.replace(&self.api_key, "***")))
    }
}

impl<H: Http> AchievementProvider for SteamProvider<H> {
    fn id(&self) -> &'static str {
        "steam"
    }

    fn fetch(&self, game: &GameRef) -> Result<AchievementSet> {
        if self.api_key.trim().is_empty() || self.steam_id.trim().is_empty() {
            return Err(Error::NoCredentials);
        }
        let app = game.steam_app_id.ok_or(Error::NoAchievements)?;
        let key = self.api_key.trim();

        // 1. Catálogo (nome, descrição, ícones, oculta).
        let (status, body) = self.get(&format!("{BASE}/ISteamUserStats/GetSchemaForGame/v2/?key={key}&appid={app}&l=brazilian"))?;
        check_status(status)?;
        let mut list = parse_schema(&body)?;
        if list.is_empty() {
            return Err(Error::NoAchievements);
        }

        // 2. Raridade global (não precisa de perfil público; se falhar, segue sem níveis).
        if let Ok((200, body)) = self.get(&format!("{BASE}/ISteamUserStats/GetGlobalAchievementPercentagesForApp/v2/?gameid={app}")) {
            if let Ok(p) = parse_percentages(&body) {
                for a in &mut list {
                    if let Some(v) = p.get(&a.id) {
                        a.percent = Some(*v);
                        a.tier = Some(Tier::from_percent(*v));
                    }
                }
            }
        }

        // 3. O que o usuário desbloqueou (exige perfil de jogos público).
        let url = format!("{BASE}/ISteamUserStats/GetPlayerAchievements/v1/?key={key}&steamid={}&appid={app}", self.steam_id.trim());
        let (status, body) = self.get(&url)?;
        match status {
            200 => {}
            403 => return Ok(AchievementSet { achievements: list, progress_known: false }),
            s => check_status(s)?,
        }
        match parse_player(&body) {
            Ok(done) => {
                for a in &mut list {
                    if let Some(t) = done.get(&a.id) {
                        a.unlocked = true;
                        a.unlock_time = if *t > 0 { Some(*t) } else { None };
                    }
                }
                Ok(AchievementSet { achievements: list, progress_known: true })
            }
            Err(Error::PrivateProfile) => Ok(AchievementSet { achievements: list, progress_known: false }),
            Err(e) => Err(e),
        }
    }
}

fn check_status(s: u16) -> Result<()> {
    match s {
        200 => Ok(()),
        401 | 403 => Err(Error::BadKey),
        s => Err(Error::Network(format!("resposta HTTP {s}"))),
    }
}

pub fn parse_schema(body: &str) -> Result<Vec<Achievement>> {
    #[derive(Deserialize)]
    struct Raiz {
        game: Option<Jogo>,
    }
    #[derive(Deserialize)]
    struct Jogo {
        #[serde(rename = "availableGameStats")]
        stats: Option<Stats>,
    }
    #[derive(Deserialize)]
    struct Stats {
        #[serde(default)]
        achievements: Vec<Item>,
    }
    #[derive(Deserialize)]
    struct Item {
        name: String,
        #[serde(rename = "displayName", default)]
        display_name: String,
        #[serde(default)]
        description: String,
        #[serde(default)]
        icon: String,
        #[serde(default)]
        icongray: String,
        #[serde(default)]
        hidden: u8,
    }
    let raiz: Raiz = serde_json::from_str(body).map_err(|_| Error::BadResponse)?;
    let itens = raiz.game.and_then(|g| g.stats).map(|s| s.achievements).unwrap_or_default();
    Ok(itens
        .into_iter()
        .map(|i| Achievement {
            id: i.name.clone(),
            name: if i.display_name.is_empty() { i.name } else { i.display_name },
            description: i.description,
            icon: i.icon,
            icon_gray: i.icongray,
            hidden: i.hidden == 1,
            unlocked: false,
            unlock_time: None,
            percent: None,
            tier: None,
        })
        .collect())
}

/// Conquistas desbloqueadas: id → hora do desbloqueio (segundos desde 1970; 0 se desconhecida).
pub fn parse_player(body: &str) -> Result<HashMap<String, u64>> {
    #[derive(Deserialize)]
    struct Raiz {
        playerstats: Option<Player>,
    }
    #[derive(Deserialize)]
    struct Player {
        #[serde(default)]
        success: bool,
        #[serde(default)]
        error: Option<String>,
        #[serde(default)]
        achievements: Vec<Item>,
    }
    #[derive(Deserialize)]
    struct Item {
        apiname: String,
        #[serde(default)]
        achieved: u8,
        #[serde(default)]
        unlocktime: u64,
    }
    let raiz: Raiz = serde_json::from_str(body).map_err(|_| Error::BadResponse)?;
    let p = raiz.playerstats.ok_or(Error::BadResponse)?;
    if !p.success {
        let erro = p.error.unwrap_or_default().to_lowercase();
        return Err(if erro.contains("not public") || erro.contains("private") { Error::PrivateProfile } else { Error::BadResponse });
    }
    Ok(p.achievements.into_iter().filter(|a| a.achieved == 1).map(|a| (a.apiname, a.unlocktime)).collect())
}

/// Porcentagem global por conquista. A Steam manda número ou texto, conforme o jogo.
pub fn parse_percentages(body: &str) -> Result<HashMap<String, f32>> {
    #[derive(Deserialize)]
    struct Raiz {
        achievementpercentages: Option<Lista>,
    }
    #[derive(Deserialize)]
    struct Lista {
        #[serde(default)]
        achievements: Vec<Item>,
    }
    #[derive(Deserialize)]
    struct Item {
        name: String,
        percent: serde_json::Value,
    }
    let raiz: Raiz = serde_json::from_str(body).map_err(|_| Error::BadResponse)?;
    let lista = raiz.achievementpercentages.ok_or(Error::BadResponse)?;
    Ok(lista
        .achievements
        .into_iter()
        .filter_map(|i| {
            let v = match &i.percent {
                serde_json::Value::Number(n) => n.as_f64()?,
                serde_json::Value::String(s) => s.trim().parse().ok()?,
                _ => return None,
            };
            Some((i.name, v as f32))
        })
        .collect())
}
