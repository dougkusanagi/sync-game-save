use sgs_achievements::*;
use std::cell::RefCell;

const SCHEMA: &str = r#"{"game":{"gameName":"Jogo","availableGameStats":{"achievements":[
 {"name":"ACH_A","defaultvalue":0,"displayName":"Primeiros passos","hidden":0,"description":"Comece o jogo","icon":"https://x/a.jpg","icongray":"https://x/ag.jpg"},
 {"name":"ACH_B","defaultvalue":0,"displayName":"Segredo","hidden":1,"icon":"https://x/b.jpg","icongray":"https://x/bg.jpg"},
 {"name":"ACH_C","defaultvalue":0,"displayName":"Lenda","hidden":0,"description":"Termine tudo","icon":"https://x/c.jpg","icongray":"https://x/cg.jpg"}]}}}"#;
const PERCENT: &str = r#"{"achievementpercentages":{"achievements":[{"name":"ACH_A","percent":91.5},{"name":"ACH_B","percent":"12.25"},{"name":"ACH_C","percent":0.8}]}}"#;
const PLAYER: &str = r#"{"playerstats":{"steamID":"1","gameName":"Jogo","success":true,"achievements":[
 {"apiname":"ACH_A","achieved":1,"unlocktime":1700000000},{"apiname":"ACH_B","achieved":0,"unlocktime":0},{"apiname":"ACH_C","achieved":1,"unlocktime":0}]}}"#;
const PRIVATE: &str = r#"{"playerstats":{"error":"Profile is not public","success":false}}"#;

struct Falso {
    player: (u16, &'static str),
    chamadas: RefCell<Vec<String>>,
}

impl Http for Falso {
    fn get(&self, url: &str) -> std::result::Result<(u16, String), String> {
        self.chamadas.borrow_mut().push(url.to_string());
        if url.contains("GetSchemaForGame") {
            Ok((200, SCHEMA.into()))
        } else if url.contains("GlobalAchievementPercentages") {
            Ok((200, PERCENT.into()))
        } else {
            Ok((self.player.0, self.player.1.into()))
        }
    }
}

fn provedor(player: (u16, &'static str)) -> SteamProvider<Falso> {
    SteamProvider { api_key: "CHAVE-SECRETA".into(), steam_id: "7656".into(), http: Falso { player, chamadas: RefCell::new(vec![]) } }
}

fn jogo() -> GameRef {
    GameRef { steam_app_id: Some(241600) }
}

#[test]
fn niveis_por_raridade_nos_limites_do_plano() {
    assert_eq!(Tier::from_percent(50.1), Tier::Common);
    assert_eq!(Tier::from_percent(50.0), Tier::Uncommon);
    assert_eq!(Tier::from_percent(20.0), Tier::Uncommon);
    assert_eq!(Tier::from_percent(19.9), Tier::Rare);
    assert_eq!(Tier::from_percent(5.0), Tier::Rare);
    assert_eq!(Tier::from_percent(4.9), Tier::UltraRare);
    assert_eq!(Tier::UltraRare.label(), "Ultrarrara");
}

#[test]
fn junta_catalogo_raridade_e_progresso() {
    let set = provedor((200, PLAYER)).fetch(&jogo()).unwrap();
    assert!(set.progress_known);
    let a = |id: &str| set.achievements.iter().find(|x| x.id == id).unwrap();
    assert!(a("ACH_A").unlocked && a("ACH_A").unlock_time == Some(1700000000));
    assert_eq!(a("ACH_A").tier, Some(Tier::Common));
    assert!(!a("ACH_B").unlocked && a("ACH_B").hidden);
    assert_eq!(a("ACH_B").tier, Some(Tier::Rare), "porcentagem em texto também vale");
    assert!(a("ACH_C").unlocked && a("ACH_C").unlock_time.is_none(), "sem data conhecida");
    assert_eq!(a("ACH_C").tier, Some(Tier::UltraRare));

    let s = set.summary();
    assert_eq!((s.unlocked, s.total, s.percent, s.complete), (2, 3, 66, false));
    assert_eq!(s.by_tier.get("Comum"), Some(&1));
    assert_eq!(s.by_tier.get("Ultrarrara"), Some(&1));
}

#[test]
fn jogo_completo() {
    let tudo = PLAYER.replace(r#""achieved":0"#, r#""achieved":1"#);
    let set = provedor((200, Box::leak(tudo.into_boxed_str()))).fetch(&jogo()).unwrap();
    assert!(set.summary().complete);
}

#[test]
fn perfil_privado_mostra_so_o_catalogo() {
    for resposta in [(200, PRIVATE), (403, "")] {
        let set = provedor(resposta).fetch(&jogo()).unwrap();
        assert!(!set.progress_known);
        assert_eq!(set.achievements.len(), 3);
        assert!(set.achievements.iter().all(|a| !a.unlocked));
        assert!(set.achievements.iter().all(|a| a.tier.is_some()), "a raridade global continua disponível");
    }
}

#[test]
fn erros_claros() {
    let sem_chave = SteamProvider { api_key: " ".into(), steam_id: "1".into(), http: Falso { player: (200, PLAYER), chamadas: RefCell::new(vec![]) } };
    assert_eq!(sem_chave.fetch(&jogo()), Err(Error::NoCredentials));
    assert_eq!(provedor((200, PLAYER)).fetch(&GameRef { steam_app_id: None }), Err(Error::NoAchievements));
    assert_eq!(parse_schema(r#"{"game":{}}"#).unwrap().len(), 0, "jogo sem conquistas");
    assert_eq!(parse_schema("não é json"), Err(Error::BadResponse));
    assert_eq!(parse_player(r#"{"playerstats":{"success":false,"error":"Requested app has no stats"}}"#), Err(Error::BadResponse));
}

#[test]
fn chave_invalida_e_rede_fora_do_ar_nao_vazam_a_chave() {
    struct Recusa(u16);
    impl Http for Recusa {
        fn get(&self, url: &str) -> std::result::Result<(u16, String), String> {
            if self.0 == 0 {
                Err(format!("falha de conexão em {url}"))
            } else {
                Ok((self.0, String::new()))
            }
        }
    }
    let p = SteamProvider { api_key: "CHAVE-SECRETA".into(), steam_id: "1".into(), http: Recusa(403) };
    assert_eq!(p.fetch(&jogo()), Err(Error::BadKey));
    let p = SteamProvider { api_key: "CHAVE-SECRETA".into(), steam_id: "1".into(), http: Recusa(0) };
    let msg = p.fetch(&jogo()).unwrap_err().to_string();
    assert!(!msg.contains("CHAVE-SECRETA"), "a chave não pode aparecer na mensagem: {msg}");
    assert!(msg.contains("Mostro o que ficou guardado"));
}
