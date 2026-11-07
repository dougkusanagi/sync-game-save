//! Artes (capa, fundo, logo e capa vertical): Steam CDN quando o catálogo tem `steam.id` e SteamGridDB como alternativa.
//! As imagens baixadas ficam no cache local; a chave do SteamGridDB nunca é devolvida.

use serde_json::Value;
use std::time::Duration;

const SGDB: &str = "https://www.steamgriddb.com/api/v2";

/// Tipo de arte de um jogo.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ArtKind {
    /// Capa horizontal 460×215 (faixa do Início).
    Header,
    /// Arte larga de fundo.
    Hero,
    /// Logo com fundo transparente.
    Logo,
    /// Capa vertical 600×900 (grade da biblioteca).
    Cover,
}

impl ArtKind {
    pub fn parse(s: &str) -> Option<Self> {
        Some(match s {
            "header" => Self::Header,
            "hero" => Self::Hero,
            "logo" => Self::Logo,
            "cover" => Self::Cover,
            _ => return None,
        })
    }

    /// Nome do arquivo no cache. A capa horizontal mantém o nome antigo (`{id}.jpg`).
    pub fn cache_name(self, id: i64) -> String {
        match self {
            Self::Header => format!("{id}.jpg"),
            Self::Hero => format!("{id}-hero.jpg"),
            Self::Logo => format!("{id}-logo.png"),
            Self::Cover => format!("{id}-cover.jpg"),
        }
    }

    fn steam_file(self) -> &'static str {
        match self {
            Self::Header => "header.jpg",
            Self::Hero => "library_hero.jpg",
            Self::Logo => "logo.png",
            Self::Cover => "library_600x900.jpg",
        }
    }

    fn sgdb_path(self, game: u64) -> String {
        match self {
            Self::Header => format!("grids/game/{game}?dimensions=460x215,920x430"),
            Self::Hero => format!("heroes/game/{game}"),
            Self::Logo => format!("logos/game/{game}"),
            Self::Cover => format!("grids/game/{game}?dimensions=600x900"),
        }
    }
}

pub fn steam_header_url(steam_id: u64) -> String {
    steam_art_url(steam_id, ArtKind::Header)
}

pub fn steam_art_url(steam_id: u64, kind: ArtKind) -> String {
    format!("https://cdn.cloudflare.steamstatic.com/steam/apps/{steam_id}/{}", kind.steam_file())
}

pub enum Busca {
    Achou(u64),
    Nada,
    Falhou,
}

/// Procura o jogo na loja da Steam (sem chave) e aceita só o nome igual, ignorando
/// espaços, maiúsculas e pontuação ("RogueLegacy" = "Rogue Legacy", mas não "Rogue Legacy 2").
pub fn steam_store_search(name: &str) -> Busca {
    if name.trim().is_empty() {
        return Busca::Nada;
    }
    let url = format!("https://store.steampowered.com/api/storesearch/?term={}&l=portuguese&cc=BR", query_encode(name));
    let Ok(mut r) = ureq::get(&url).config().http_status_as_error(false).timeout_global(Some(Duration::from_secs(10))).build().call() else {
        return Busca::Falhou;
    };
    if r.status().as_u16() != 200 {
        return Busca::Falhou;
    }
    let Some(json) = r.body_mut().with_config().limit(1024 * 1024).read_to_string().ok().and_then(|c| serde_json::from_str::<Value>(&c).ok()) else {
        return Busca::Falhou;
    };
    parse_store_search(&json, name).map_or(Busca::Nada, Busca::Achou)
}

fn normalizar(s: &str) -> String {
    s.chars().filter(|c| c.is_alphanumeric()).flat_map(char::to_lowercase).collect()
}

fn parse_store_search(json: &Value, name: &str) -> Option<u64> {
    let alvo = normalizar(name);
    json.get("items")?.as_array()?.iter().find_map(|item| {
        let ok = item.get("type").and_then(Value::as_str) == Some("app") && normalizar(item.get("name")?.as_str()?) == alvo;
        if ok { item.get("id")?.as_u64() } else { None }
    })
}

/// Busca no SteamGridDB a primeira arte do tipo pedido para o nome do jogo.
pub fn steamgriddb_art(api_key: &str, name: &str, kind: ArtKind) -> Option<String> {
    let chave = api_key.trim();
    if chave.is_empty() || name.trim().is_empty() {
        return None;
    }
    let busca = pegar(&format!("{SGDB}/search/autocomplete/{}", query_encode(name)), chave)?;
    let id = parse_search_id(&busca)?;
    let grades = pegar(&format!("{SGDB}/{}", kind.sgdb_path(id)), chave)?;
    parse_grid_url(&grades)
}

fn pegar(url: &str, api_key: &str) -> Option<Value> {
    let mut r = ureq::get(url)
        .header("Authorization", &format!("Bearer {api_key}"))
        .config()
        .http_status_as_error(false)
        .timeout_global(Some(Duration::from_secs(10)))
        .build()
        .call()
        // A URL não leva a chave, mas a mensagem de erro não deve expor o cabeçalho.
        .map_err(|e| e.to_string().replace(api_key, "***"))
        .ok()?;
    if r.status().as_u16() != 200 {
        return None;
    }
    let corpo = r.body_mut().with_config().limit(2 * 1024 * 1024).read_to_string().ok()?;
    serde_json::from_str(&corpo).ok()
}

fn parse_search_id(json: &Value) -> Option<u64> {
    let lista = json.get("data")?.as_array()?;
    lista.iter().find_map(|jogo| jogo.get("id")?.as_u64())
}

fn parse_grid_url(json: &Value) -> Option<String> {
    let lista = json.get("data")?.as_array()?;
    lista
        .iter()
        .find_map(|grade| grade.get("url")?.as_str())
        .filter(|url| url.starts_with("https://"))
        .map(str::to_owned)
}

/// Codifica o nome para a busca. Mantém letras, números e `-_.~`; o resto vira `%XX`.
fn query_encode(texto: &str) -> String {
    let mut saida = String::new();
    for b in texto.trim().bytes() {
        if b.is_ascii_alphanumeric() || matches!(b, b'-' | b'_' | b'.' | b'~') {
            saida.push(b as char);
        } else {
            saida.push_str(&format!("%{b:02X}"));
        }
    }
    saida
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn cada_tipo_de_arte_tem_url_e_cache_proprios() {
        assert_eq!(ArtKind::parse("hero"), Some(ArtKind::Hero));
        assert_eq!(ArtKind::parse("fundo"), None);
        assert_eq!(steam_art_url(367520, ArtKind::Logo), "https://cdn.cloudflare.steamstatic.com/steam/apps/367520/logo.png");
        assert_eq!(steam_header_url(1), "https://cdn.cloudflare.steamstatic.com/steam/apps/1/header.jpg");
        // A capa horizontal mantém o nome antigo, para aproveitar o cache já baixado.
        assert_eq!(ArtKind::Header.cache_name(7), "7.jpg");
        assert_eq!(ArtKind::Cover.cache_name(7), "7-cover.jpg");
        assert_eq!(ArtKind::Hero.sgdb_path(42), "heroes/game/42");
    }

    #[test]
    fn busca_da_loja_aceita_so_o_nome_igual() {
        let json = serde_json::json!({"items":[
            {"type":"app","name":"Rogue Legacy 2","id":1253920},
            {"type":"app","name":"Rogue Legacy","id":241600}
        ]});
        assert_eq!(parse_store_search(&json, "RogueLegacy"), Some(241600));
        assert_eq!(parse_store_search(&json, "rogue legacy"), Some(241600));
        assert_eq!(parse_store_search(&json, "Rogue"), None);
    }

    #[test]
    fn entende_as_respostas_do_steamgriddb() {
        let busca: Value = serde_json::from_str(
            r#"{"success":true,"data":[{"id":1234,"name":"Hollow Knight"},{"id":99,"name":"Outro"}]}"#,
        )
        .unwrap();
        assert_eq!(parse_search_id(&busca), Some(1234));

        let grades: Value =
            serde_json::from_str(r#"{"success":true,"data":[{"url":"https://cdn2.steamgriddb.com/abc.png"}]}"#)
                .unwrap();
        assert_eq!(parse_grid_url(&grades).as_deref(), Some("https://cdn2.steamgriddb.com/abc.png"));

        assert_eq!(parse_grid_url(&serde_json::json!({"data":[]})), None);
        assert_eq!(parse_grid_url(&serde_json::json!({"data":[{"url":"http://inseguro"}]})), None);
    }

    #[test]
    fn codifica_o_nome_para_a_busca() {
        assert_eq!(query_encode("Hollow Knight"), "Hollow%20Knight");
        assert_eq!(query_encode("Ação & Aventura"), "A%C3%A7%C3%A3o%20%26%20Aventura");
        assert_eq!(query_encode("  jogo-1.2_ok~  "), "jogo-1.2_ok~");
    }

    #[test]
    fn url_da_steam() {
        assert_eq!(
            steam_header_url(241600),
            "https://cdn.cloudflare.steamstatic.com/steam/apps/241600/header.jpg"
        );
    }
}
