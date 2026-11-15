package achievements

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

const schema = `{"game":{"gameName":"Jogo","availableGameStats":{"achievements":[
 {"name":"ACH_A","defaultvalue":0,"displayName":"Primeiros passos","hidden":0,"description":"Comece o jogo","icon":"https://x/a.jpg","icongray":"https://x/ag.jpg"},
 {"name":"ACH_B","defaultvalue":0,"displayName":"Segredo","hidden":1,"icon":"https://x/b.jpg","icongray":"https://x/bg.jpg"},
 {"name":"ACH_C","defaultvalue":0,"displayName":"Lenda","hidden":0,"description":"Termine tudo","icon":"https://x/c.jpg","icongray":"https://x/cg.jpg"}]}}}`
const percent = `{"achievementpercentages":{"achievements":[{"name":"ACH_A","percent":91.5},{"name":"ACH_B","percent":"12.25"},{"name":"ACH_C","percent":0.8}]}}`
const player = `{"playerstats":{"steamID":"1","gameName":"Jogo","success":true,"achievements":[
 {"apiname":"ACH_A","achieved":1,"unlocktime":1700000000},{"apiname":"ACH_B","achieved":0,"unlocktime":0},{"apiname":"ACH_C","achieved":1,"unlocktime":0}]}}`
const private = `{"playerstats":{"error":"Profile is not public","success":false}}`

type falso struct {
	playerStatus int
	playerBody   string
}

func (f falso) Get(url string) (int, string, error) {
	switch {
	case strings.Contains(url, "GetSchemaForGame"):
		return 200, schema, nil
	case strings.Contains(url, "GlobalAchievementPercentages"):
		return 200, percent, nil
	}
	return f.playerStatus, f.playerBody, nil
}

func provedor(status int, body string) SteamProvider {
	return SteamProvider{APIKey: "CHAVE-SECRETA", SteamID: "7656", HTTP: falso{status, body}}
}

var app = uint64(241600)
var jogo = GameRef{SteamAppID: &app}

func achado(set Set, id string) Achievement {
	for _, a := range set.Achievements {
		if a.ID == id {
			return a
		}
	}
	panic("sem " + id)
}

func TestNiveisPorRaridadeNosLimites(t *testing.T) {
	for p, want := range map[float32]Tier{50.1: Common, 50: Uncommon, 20: Uncommon, 19.9: Rare, 5: Rare, 4.9: UltraRare} {
		if got := TierFromPercent(p); got != want {
			t.Fatal(p, got)
		}
	}
	if UltraRare.Label() != "Ultrarrara" {
		t.Fatal()
	}
}

func TestJuntaCatalogoRaridadeEProgresso(t *testing.T) {
	set, err := provedor(200, player).Fetch(jogo)
	if err != nil || !set.ProgressKnown {
		t.Fatal(set, err)
	}
	a, b, c := achado(set, "ACH_A"), achado(set, "ACH_B"), achado(set, "ACH_C")
	if !a.Unlocked || a.UnlockTime == nil || *a.UnlockTime != 1700000000 || *a.Tier != Common {
		t.Fatal(a)
	}
	if b.Unlocked || !b.Hidden || *b.Tier != Rare {
		t.Fatal("porcentagem em texto também vale", b)
	}
	if !c.Unlocked || c.UnlockTime != nil || *c.Tier != UltraRare {
		t.Fatal(c)
	}
	s := set.Summary()
	if s.Unlocked != 2 || s.Total != 3 || s.Percent != 66 || s.Complete || s.ByTier["Comum"] != 1 || s.ByTier["Ultrarrara"] != 1 {
		t.Fatal(s)
	}
}

func TestJogoCompleto(t *testing.T) {
	set, err := provedor(200, strings.ReplaceAll(player, `"achieved":0`, `"achieved":1`)).Fetch(jogo)
	if err != nil || !set.Summary().Complete {
		t.Fatal(set, err)
	}
}

func TestPerfilPrivadoMostraSoOCatalogo(t *testing.T) {
	for _, r := range []struct {
		status int
		body   string
	}{{200, private}, {403, ""}} {
		set, err := provedor(r.status, r.body).Fetch(jogo)
		if err != nil || set.ProgressKnown || len(set.Achievements) != 3 {
			t.Fatal(set, err)
		}
		for _, a := range set.Achievements {
			if a.Unlocked || a.Tier == nil {
				t.Fatal("raridade global deve continuar disponível", a)
			}
		}
	}
}

func TestErrosClaros(t *testing.T) {
	semChave := SteamProvider{APIKey: " ", SteamID: "1", HTTP: falso{200, player}}
	if _, err := semChave.Fetch(jogo); !errors.Is(err, ErrNoCredentials) {
		t.Fatal(err)
	}
	if _, err := provedor(200, player).Fetch(GameRef{}); !errors.Is(err, ErrNoAchievements) {
		t.Fatal(err)
	}
	if l, err := ParseSchema(`{"game":{}}`); err != nil || len(l) != 0 {
		t.Fatal(l, err)
	}
	if _, err := ParseSchema("não é json"); !errors.Is(err, ErrBadResponse) {
		t.Fatal(err)
	}
	if _, err := ParsePlayer(`{"playerstats":{"success":false,"error":"Requested app has no stats"}}`); !errors.Is(err, ErrBadResponse) {
		t.Fatal(err)
	}
}

type recusa int

func (r recusa) Get(url string) (int, string, error) {
	if r == 0 {
		return 0, "", errors.New("falha de conexão em " + url)
	}
	return int(r), "", nil
}

func TestChaveInvalidaERedeForaDoArNaoVazamAChave(t *testing.T) {
	p := SteamProvider{APIKey: "CHAVE-SECRETA", SteamID: "1", HTTP: recusa(403)}
	if _, err := p.Fetch(jogo); !errors.Is(err, ErrBadKey) {
		t.Fatal(err)
	}
	p.HTTP = recusa(0)
	_, err := p.Fetch(jogo)
	if err == nil || strings.Contains(err.Error(), "CHAVE-SECRETA") || !strings.Contains(err.Error(), "Mostro o que ficou guardado") {
		t.Fatal(err)
	}
}

func TestLeOXMLPublicoDoPerfil(t *testing.T) {
	xml := `<?xml version="1.0"?><playerstats><privacyState>public</privacyState><achievements>
		<achievement closed="1"><iconClosed><![CDATA[https://a/cor.jpg]]></iconClosed><iconOpen><![CDATA[https://a/cinza.jpg]]></iconOpen>
		<name><![CDATA[Biofobia]]></name><apiname><![CDATA[fear_of_life]]></apiname><description><![CDATA[Morra 20 vezes.]]></description>
		<unlockTimestamp>1700000000</unlockTimestamp></achievement>
		<achievement closed="0"><name><![CDATA[Outra]]></name><apiname><![CDATA[outra]]></apiname><description></description></achievement>
	</achievements></playerstats>`
	s, err := ParseCommunityXML(xml)
	if err != nil || len(s.Achievements) != 2 {
		t.Fatal(s, err)
	}
	a := s.Achievements[0]
	if !a.Unlocked || a.UnlockTime == nil || *a.UnlockTime != 1700000000 || a.Icon != "https://a/cor.jpg" || a.IconGray != "https://a/cinza.jpg" || s.Achievements[1].Unlocked {
		t.Fatal(s)
	}
	if _, err := ParseCommunityXML("<playerstats><privacyState>private</privacyState></playerstats>"); !errors.Is(err, ErrPrivateProfile) {
		t.Fatal(err)
	}
	if _, err := ParseCommunityXML("<response><error>The specified profile is private.</error></response>"); !errors.Is(err, ErrPrivateProfile) {
		t.Fatal(err)
	}
}

func TestAchaAContaMaisRecenteNoLoginusers(t *testing.T) {
	vdf := "\"users\"\n{\n\t\"76561198000000001\"\n\t{\n\t\t\"MostRecent\"\t\t\"0\"\n\t}\n\t\"76561198263490373\"\n\t{\n\t\t\"AccountName\"\t\t\"x\"\n\t\t\"MostRecent\"\t\t\"1\"\n\t}\n}"
	if got := SteamIDFromLoginUsers(vdf); got != "76561198263490373" {
		t.Fatal(got)
	}
	if got := SteamIDFromLoginUsers("\"users\"\n{\n\t\"76561198000000001\"\n}"); got != "76561198000000001" {
		t.Fatal(got)
	}
	if SteamIDFromLoginUsers("") != "" {
		t.Fatal()
	}
}

func TestJSONDoCacheUsaOsNomesDaVersaoRust(t *testing.T) {
	set, _ := provedor(200, player).Fetch(jogo)
	b, err := jsonMarshal(set)
	if err != nil || !strings.Contains(b, `"icon_gray":`) || !strings.Contains(b, `"progress_known":true`) || !strings.Contains(b, `"tier":"Common"`) || !strings.Contains(b, `"unlock_time":1700000000`) {
		t.Fatal(b, err)
	}
}

func jsonMarshal(v any) (string, error) {
	b, err := json.Marshal(v)
	return string(b), err
}
