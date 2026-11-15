package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"sync-game-save/internal/achievements"
	"sync-game-save/internal/library"
)

func TestHubCalculaProgressoEDesbloqueiaMarcos(t *testing.T) {
	h := calcularHub(usoHub{playtimeSecs: 10*3600 + 1800, jogos: 5, sequencia: 25, melhorSequencia: 30, jogosCompletos: 1})
	if h.Horas != 10 || h.Total != 12 || h.Desbloqueadas != 7 || h.SequenciaAtual != 25 || h.MelhorSequencia != 30 || h.JogosCompletos != 1 {
		t.Fatal(h)
	}
	for _, c := range h.Conquistas {
		if c.ID == "primeira-hora" && (!c.Desbloqueada || c.Progresso != 1) {
			t.Fatal(c)
		}
		if c.ID == "sequencia-100" && (c.Desbloqueada || c.Progresso != 25) {
			t.Fatal(c)
		}
	}
	vazio := calcularHub(usoHub{})
	for _, c := range vazio.Conquistas {
		if c.Desbloqueada || c.Progresso != 0 {
			t.Fatal(c)
		}
	}
	if um := calcularHub(usoHub{playtimeSecs: 3600, jogos: 1}); !um.Conquistas[0].Desbloqueada || um.Desbloqueadas != 2 {
		t.Fatal("o que falta desbloquear deve vir depois", um)
	}
}

func TestArteTiposNomesEUrls(t *testing.T) {
	if t1, ok := analisarTipoArte("hero"); !ok || t1 != arteHero {
		t.Fatal()
	}
	if _, ok := analisarTipoArte("fundo"); ok {
		t.Fatal("tipo inválido aceito")
	}
	if urlArteSteam(367520, arteLogo) != "https://cdn.cloudflare.steamstatic.com/steam/apps/367520/logo.png" {
		t.Fatal(urlArteSteam(367520, arteLogo))
	}
	if arteHeader.nomeNoCache(7) != "7.jpg" || arteCover.nomeNoCache(7) != "7-cover.jpg" || arteHero.caminhoSGDB(42) != "heroes/game/42" {
		t.Fatal("nomes")
	}
	if _, err := (App{}).ArteDoJogo(1, "fundo"); err == nil {
		t.Fatal("tipo desconhecido aceito")
	}
}

func TestBuscaDaLojaAceitaSoONomeIgual(t *testing.T) {
	corpo := []byte(`{"items":[{"type":"app","name":"Rogue Legacy 2","id":1253920},{"type":"app","name":"Rogue Legacy","id":241600}]}`)
	for nome, esperado := range map[string]uint64{"RogueLegacy": 241600, "rogue legacy": 241600, "Rogue": 0} {
		if id, ok := idDaBuscaDaLoja(corpo, nome); id != esperado || ok != (esperado != 0) {
			t.Fatal(nome, id, ok)
		}
	}
}

func TestRespostasDoSteamGridDBECodificacao(t *testing.T) {
	if id, ok := primeiroIDSGDB([]byte(`{"success":true,"data":[{"id":1234,"name":"Hollow Knight"},{"id":99}]}`)); !ok || id != 1234 {
		t.Fatal(id, ok)
	}
	if u, ok := primeiraURLSGDB([]byte(`{"data":[{"url":"https://cdn2.steamgriddb.com/abc.png"}]}`)); !ok || u != "https://cdn2.steamgriddb.com/abc.png" {
		t.Fatal(u, ok)
	}
	if _, ok := primeiraURLSGDB([]byte(`{"data":[]}`)); ok {
		t.Fatal("lista vazia aceita")
	}
	if _, ok := primeiraURLSGDB([]byte(`{"data":[{"url":"http://inseguro"}]}`)); ok {
		t.Fatal("url sem https aceita")
	}
	for in, want := range map[string]string{"Hollow Knight": "Hollow%20Knight", "Ação & Aventura": "A%C3%A7%C3%A3o%20%26%20Aventura", "  jogo-1.2_ok~  ": "jogo-1.2_ok~"} {
		if got := codificarBusca(in); got != want {
			t.Fatal(in, got)
		}
	}
}

func TestBaixarArteSoAceitaImagem(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ok.jpg":
			w.Write([]byte{0xFF, 0xD8, 0xFF, 0xE0, 1, 2, 3})
		case "/pagina.html":
			w.Write([]byte("<html>não é imagem</html>"))
		default:
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()
	dest := filepath.Join(t.TempDir(), "art", "1.jpg")
	if p, ok := baixarArte(srv.URL+"/ok.jpg", dest); !ok || p != dest {
		t.Fatal(p, ok)
	}
	if _, err := os.Stat(dest + ".tmp"); !os.IsNotExist(err) {
		t.Fatal("temporário ficou")
	}
	outro := filepath.Join(t.TempDir(), "2.jpg")
	for _, caminho := range []string{"/pagina.html", "/inexistente"} {
		if _, ok := baixarArte(srv.URL+caminho, outro); ok {
			t.Fatal("aceitou", caminho)
		}
	}
	if _, err := os.Stat(outro); !os.IsNotExist(err) {
		t.Fatal("arquivo inválido gravado")
	}
}

func TestArteEmCacheNaoUsaARede(t *testing.T) {
	t.Setenv("SGS_DATA_DIR", t.TempDir())
	id, err := (App{}).AdicionarJogo(library.NewGame{Name: "Jogo", Exe: "x.exe"})
	if err != nil {
		t.Fatal(err)
	}
	cache := filepath.Join(pastaDados(), "art", arteHeader.nomeNoCache(id))
	os.MkdirAll(filepath.Dir(cache), 0755)
	os.WriteFile(cache, []byte("img"), 0644)
	got, err := (App{}).CapaDoJogo(id)
	if err != nil || got == nil || *got != cache {
		t.Fatal(got, err)
	}
}

type steamFalsa struct {
	chamadas int
	falha    bool
	perfil   string
}

func (s *steamFalsa) Get(url string) (int, string, error) {
	s.chamadas++
	if s.falha {
		return 0, "", errors.New("sem internet")
	}
	if strings.Contains(url, "GlobalAchievementPercentages") {
		return 200, `{"achievementpercentages":{"achievements":[{"name":"a","percent":10}]}}`, nil
	}
	return 200, s.perfil, nil
}

const perfilXML = `<playerstats><privacyState>public</privacyState><achievements>
<achievement closed="1"><name>Um</name><apiname>a</apiname><description>d</description><unlockTimestamp>5</unlockTimestamp></achievement>
</achievements></playerstats>`

func TestConquistasUsamCacheComErroEPerfilPrivado(t *testing.T) {
	t.Setenv("SGS_DATA_DIR", t.TempDir())
	t.Setenv("SGS_STEAM_DIR", t.TempDir())
	app := App{}
	id, _ := app.AdicionarJogo(library.NewGame{Name: "Jogo", Exe: "x.exe"})
	lib, _ := abrirBiblioteca()
	lib.SetSetting("steam_appid:"+itoa(id), "241600")
	lib.SetSetting("steam_id", "76561198000000001")
	lib.Close()

	rede := &steamFalsa{perfil: perfilXML}
	v, err := conquistasDoJogo(id, false, rede, 1000)
	if err != nil || v.FromCache || v.Summary.Unlocked != 1 || !v.Summary.Complete || v.Notice != nil || *v.Set.Achievements[0].Tier != achievements.Rare {
		t.Fatal(v, err)
	}
	antes := rede.chamadas
	// Dentro da validade, usa o cache sem rede.
	if v, err = conquistasDoJogo(id, false, rede, 1100); err != nil || !v.FromCache || v.FetchedAt != 1000 || rede.chamadas != antes {
		t.Fatal(v, err, rede.chamadas)
	}
	// Sem internet depois da validade: devolve o guardado com o motivo.
	rede.falha = true
	if v, err = conquistasDoJogo(id, false, rede, 5000); err != nil || !v.FromCache || v.Notice == nil || !strings.Contains(*v.Notice, "Mostro o que ficou guardado") {
		t.Fatal(v, err)
	}
	// Sem cache e sem internet: erro.
	id2, _ := app.AdicionarJogo(library.NewGame{Name: "Outro", Exe: "y.exe"})
	lib, _ = abrirBiblioteca()
	lib.SetSetting("steam_appid:"+itoa(id2), "1")
	lib.Close()
	if _, err := conquistasDoJogo(id2, true, rede, 5000); err == nil {
		t.Fatal("erro sem cache engolido")
	}
	// A conta do jogo completo conta nos marcos do HUB.
	if h, _ := app.ConquistasHub(); h.JogosCompletos != 1 {
		t.Fatal(h)
	}
}

func TestCredenciaisSteamValidamOSteamID(t *testing.T) {
	t.Setenv("SGS_DATA_DIR", t.TempDir())
	app := App{}
	if err := app.DefinirCredenciaisSteam("chave", "abc"); err == nil {
		t.Fatal("steam id inválido aceito")
	}
	if err := app.DefinirCredenciaisSteam(" chave ", " 76561198000000001 "); err != nil {
		t.Fatal(err)
	}
	if err := app.DefinirChaveSteamGridDB(" k "); err != nil {
		t.Fatal(err)
	}
	est, err := app.Estado()
	if err != nil || !est.SteamConfigurada || !est.SteamGridDBConfigurada || est.SteamID == nil || *est.SteamID != "76561198000000001" {
		t.Fatal(est, err)
	}
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }
