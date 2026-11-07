//! Conquistas: interface de provedores, provedores da Steam (perfil público, sem chave; ou chave de API)
//! e cálculo de níveis por raridade. A rede é injetada, para testar sem chave nem internet.

use serde::{Deserialize, Serialize};
use std::collections::HashMap;

/// Erros já escritos para o usuário final.
#[derive(Debug, thiserror::Error, PartialEq, Eq)]
pub enum Error {
    #[error("Não achei a sua conta da Steam neste PC. Entre na Steam neste PC (ou informe o seu SteamID64 em Configurações) para ver as conquistas.")]
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

/// Provedor da Steam sem chave: lê o XML público do perfil
/// (`steamcommunity.com/profiles/<id>/stats/<app>/achievements/?xml=1`) e a raridade global.
pub struct SteamCommunityProvider<H: Http> {
    pub steam_id: String,
    pub http: H,
}

impl<H: Http> AchievementProvider for SteamCommunityProvider<H> {
    fn id(&self) -> &'static str {
        "steam-community"
    }

    fn fetch(&self, game: &GameRef) -> Result<AchievementSet> {
        let id = self.steam_id.trim();
        if id.is_empty() || !id.chars().all(|c| c.is_ascii_digit()) {
            return Err(Error::NoCredentials);
        }
        let app = game.steam_app_id.ok_or(Error::NoAchievements)?;
        let url = format!("https://steamcommunity.com/profiles/{id}/stats/{app}/achievements/?xml=1&l=brazilian");
        let (status, body) = self.http.get(&url).map_err(Error::Network)?;
        if status != 200 {
            return Err(Error::Network(format!("resposta HTTP {status}")));
        }
        let mut set = parse_community_xml(&body)?;
        if let Ok((200, body)) = self.http.get(&format!("{BASE}/ISteamUserStats/GetGlobalAchievementPercentagesForApp/v2/?gameid={app}")) {
            if let Ok(p) = parse_percentages(&body) {
                // O XML traz o id em minúsculas; a raridade global usa o id original.
                let p: HashMap<String, f32> = p.into_iter().map(|(k, v)| (k.to_lowercase(), v)).collect();
                for a in &mut set.achievements {
                    if let Some(v) = p.get(&a.id.to_lowercase()) {
                        a.percent = Some(*v);
                        a.tier = Some(Tier::from_percent(*v));
                    }
                }
            }
        }
        Ok(set)
    }
}

/// Texto entre `<tag>` e `</tag>`, sem o `CDATA`.
fn tag<'a>(xml: &'a str, nome: &str) -> Option<&'a str> {
    let ini = xml.find(&format!("<{nome}>"))? + nome.len() + 2;
    let fim = ini + xml[ini..].find(&format!("</{nome}>"))?;
    let v = xml[ini..fim].trim();
    Some(v.strip_prefix("<![CDATA[").and_then(|s| s.strip_suffix("]]>")).unwrap_or(v))
}

/// Lê o XML público de conquistas de um perfil.
pub fn parse_community_xml(xml: &str) -> Result<AchievementSet> {
    if !xml.contains("<playerstats>") {
        // Perfil privado ou inexistente: a Steam devolve uma página de erro.
        return Err(if xml.contains("<error>") || xml.to_lowercase().contains("private") { Error::PrivateProfile } else { Error::BadResponse });
    }
    if let Some(p) = tag(xml, "privacyState") {
        if p != "public" {
            return Err(Error::PrivateProfile);
        }
    }
    let mut lista = Vec::new();
    for bloco in xml.split("<achievement ").skip(1) {
        let bloco = &bloco[..bloco.find("</achievement>").unwrap_or(bloco.len())];
        let feito = bloco.starts_with("closed=\"1\"");
        let id = tag(bloco, "apiname").unwrap_or_default().to_string();
        let nome = tag(bloco, "name").unwrap_or_default().to_string();
        if id.is_empty() && nome.is_empty() {
            continue;
        }
        let quando = tag(bloco, "unlockTimestamp").and_then(|t| t.parse::<u64>().ok()).filter(|t| *t > 0);
        lista.push(Achievement {
            id: if id.is_empty() { nome.clone() } else { id },
            name: nome,
            description: tag(bloco, "description").unwrap_or_default().to_string(),
            // No XML, "iconClosed" é o ícone colorido (desbloqueada) e "iconOpen" o cinza.
            icon: tag(bloco, "iconClosed").unwrap_or_default().to_string(),
            icon_gray: tag(bloco, "iconOpen").unwrap_or_default().to_string(),
            hidden: false,
            unlocked: feito,
            unlock_time: if feito { quando } else { None },
            percent: None,
            tier: None,
        });
    }
    if lista.is_empty() {
        return Err(Error::NoAchievements);
    }
    Ok(AchievementSet { achievements: lista, progress_known: true })
}

/// SteamID64 da conta usada por último na Steam deste PC, lido do `loginusers.vdf`.
pub fn steam_id_from_loginusers(vdf: &str) -> Option<String> {
    let mut primeiro = None;
    let mut atual: Option<String> = None;
    for linha in vdf.lines() {
        let t = linha.trim().trim_matches('"');
        if t.len() == 17 && t.starts_with("7656") && t.chars().all(|c| c.is_ascii_digit()) {
            atual = Some(t.to_string());
            primeiro.get_or_insert_with(|| t.to_string());
        }
        let campos: Vec<&str> = linha.split('"').filter(|s| !s.trim().is_empty()).collect();
        if campos.len() == 2 && campos[0].eq_ignore_ascii_case("MostRecent") && campos[1] == "1" {
            if let Some(id) = &atual {
                return Some(id.clone());
            }
        }
    }
    primeiro
}

#[cfg(test)]
mod comunidade {
    use super::*;

    #[test]
    fn le_o_xml_publico_do_perfil() {
        let xml = r#"<?xml version="1.0"?><playerstats><privacyState>public</privacyState><achievements>
            <achievement closed="1"><iconClosed><![CDATA[https://a/cor.jpg]]></iconClosed><iconOpen><![CDATA[https://a/cinza.jpg]]></iconOpen>
            <name><![CDATA[Biofobia]]></name><apiname><![CDATA[fear_of_life]]></apiname><description><![CDATA[Morra 20 vezes.]]></description>
            <unlockTimestamp>1700000000</unlockTimestamp></achievement>
            <achievement closed="0"><name><![CDATA[Outra]]></name><apiname><![CDATA[outra]]></apiname><description></description></achievement>
        </achievements></playerstats>"#;
        let s = parse_community_xml(xml).unwrap();
        assert_eq!(s.achievements.len(), 2);
        assert!(s.achievements[0].unlocked);
        assert_eq!(s.achievements[0].unlock_time, Some(1700000000));
        assert_eq!(s.achievements[0].icon, "https://a/cor.jpg");
        assert!(!s.achievements[1].unlocked);
        assert_eq!(parse_community_xml("<playerstats><privacyState>private</privacyState></playerstats>"), Err(Error::PrivateProfile));
        assert_eq!(parse_community_xml("<response><error>The specified profile is private.</error></response>"), Err(Error::PrivateProfile));
    }

    #[test]
    fn acha_a_conta_mais_recente_no_loginusers() {
        let vdf = "\"users\"\n{\n\t\"76561198000000001\"\n\t{\n\t\t\"MostRecent\"\t\t\"0\"\n\t}\n\t\"76561198263490373\"\n\t{\n\t\t\"AccountName\"\t\t\"x\"\n\t\t\"MostRecent\"\t\t\"1\"\n\t}\n}";
        assert_eq!(steam_id_from_loginusers(vdf).as_deref(), Some("76561198263490373"));
        assert_eq!(steam_id_from_loginusers("\"users\"\n{\n\t\"76561198000000001\"\n}").as_deref(), Some("76561198000000001"));
        assert_eq!(steam_id_from_loginusers(""), None);
    }
}
