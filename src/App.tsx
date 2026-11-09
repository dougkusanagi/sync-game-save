import { useCallback, useEffect, useRef, useState, type CSSProperties } from "react";
import { api, aoMudarEstadoDoJogo, aoUsarControle, noTauri, textoDoErro, verificarAtualizacao, type Atualizacao, type Estado, type Game, type OpcoesJogar, type Temas } from "./api";
import { Icone } from "./Icone";
import { useArte } from "./useCapa";
import { GLYPHS } from "./nav/input.js";
import * as focus from "./nav/focus.js";
import { initInput } from "./nav/input.js";
import { voltar } from "./voltar";
import { som } from "./som";
import { acharTema, aplicarTema } from "./temas";
import { AddGame } from "./screens/AddGame";
import { Biblioteca, type EstadoCard } from "./screens/Biblioteca";
import { Conquistas } from "./screens/Conquistas";
import { ConquistasHub } from "./screens/ConquistasHub";
import { Conflito, Configuracoes, EditarJogo, Opcoes, PararJogo, Remover, SemSync, Sobre, Travado, Versoes } from "./screens/Dialogos";
import { corDoNome, Inicio, jogosDaSecao, SECOES } from "./screens/Inicio";
import { Wizard } from "./screens/Wizard";

// Os ouvintes de teclado e controle são globais e só podem ser registrados uma vez
// (o React em desenvolvimento executa os efeitos duas vezes).
let entradaIniciada = false;


type Camada =
  | { tipo: "adicionar" }
  | { tipo: "config" }
  | { tipo: "sobre" }
  | { tipo: "conquistas"; jogo: Game }
  | { tipo: "conquistas-hub" }
  | { tipo: "versoes"; jogo: Game }
  | { tipo: "opcoes"; jogo: Game }
  | { tipo: "editar"; jogo: Game }
  | { tipo: "remover"; jogo: Game }
  | { tipo: "parar"; jogo: Game }
  | { tipo: "conflito"; jogo: Game }
  | { tipo: "travado"; jogo: Game; pc: string }
  | { tipo: "semsync"; jogo: Game; mensagem: string };

export default function App() {
  const raiz = useRef<HTMLDivElement>(null);
  const [estado, setEstado] = useState<Estado | null>(null);
  const [jogos, setJogos] = useState<Game[]>([]);
  const [cards, setCards] = useState<Record<number, EstadoCard>>({});
  const [camadas, setCamadas] = useState<Camada[]>([]);
  const [aviso, setAviso] = useState<{ texto: string; erro: boolean } | null>(null);
  const [falha, setFalha] = useState<string | null>(null);
  const [atualizacao, setAtualizacao] = useState<Atualizacao | null>(null);
  const [temas, setTemas] = useState<Temas | null>(null);
  const [escolhido, setDestaque] = useState<Game | null>(null);
  const [agora, setAgora] = useState(() => new Date());
  const [tela, setTela] = useState<"inicio" | "biblioteca">("inicio");
  const [secao, setSecao] = useState(0);
  const [dispositivo, setDispositivo] = useState<"keyboard" | "xbox" | "ps">("keyboard");
  // Sem escolha (ou com a escolhida removida), o destaque é o primeiro jogo da seção do Início.
  const destaque = escolhido ?? jogosDaSecao(secao, jogos)[0] ?? null;
  const heroFundo = useArte(destaque, "hero");
  const capaFundo = useArte(destaque, "header");
  const arteFundo = heroFundo ?? capaFundo;
  const acoes = useRef<(acao: string) => void>(() => {});

  const carregar = useCallback(async () => {
    try {
      const [e, j] = await Promise.all([api.estado(), api.listarJogos()]);
      setEstado(e);
      setJogos(j);
      setFalha(null);
    } catch (err) {
      setFalha(textoDoErro(err));
    }
  }, []);

  useEffect(() => {
    void carregar();
  }, [carregar]);

  useEffect(() => {
    setDestaque((d) => (d && jogos.some((j) => j.id === d.id) ? (jogos.find((j) => j.id === d.id) ?? null) : null));
  }, [jogos]);

  useEffect(() => {
    const t = window.setInterval(() => setAgora(new Date()), 15000);
    return () => window.clearInterval(t);
  }, []);

  useEffect(() => {
    api.temas()
      .then((t) => { setTemas(t); aplicarTema(acharTema(t)); })
      .catch(() => {});
  }, []);

  const trocarTema = async (id: string) => {
    if (!temas) return;
    const anterior = acharTema(temas);
    aplicarTema(temas.temas.find((t) => t.id === id) ?? null);
    try {
      const atualizado = await api.definirTema(id);
      setTemas(atualizado);
      aplicarTema(acharTema(atualizado));
    } catch (erro) {
      aplicarTema(anterior);
      setAviso({ texto: textoDoErro(erro), erro: true });
    }
  };

  // Procura atualização ao abrir, sem atrapalhar o uso. A instalação fica em "Sobre".
  useEffect(() => {
    verificarAtualizacao()
      .then(setAtualizacao)
      .catch(() => {});
  }, []);

  const abrir = (c: Camada) => {
    som.tocar("painel");
    setCamadas((l) => [...l, c]);
  };
  const fechar = useCallback(() => setCamadas((l) => l.slice(0, -1)), []);
  const fecharTodas = () => setCamadas([]);

  // Foco, teclado e controle.
  useEffect(() => {
    if (!raiz.current || entradaIniciada) return;
    entradaIniciada = true;
    focus.init(raiz.current);
    const tratar = (acao: string) => {
      if (acao === "up" || acao === "down" || acao === "left" || acao === "right") focus.move(acao);
      else if (acao === "confirm") (focus.current() as HTMLElement | null)?.click();
      else if (acao === "back") {
        if (voltar()) som.tocar("voltar");
        else acoes.current("back");
      } else acoes.current(acao);
    };
    // No app, o controle é lido no Rust (funciona mesmo sem foco no WebView); no navegador, pela Gamepad API.
    initInput({ onDevice: (d: "keyboard" | "xbox" | "ps") => setDispositivo(d), onAction: tratar, gamepad: noTauri });
    void aoUsarControle((acao, d) => {
      setDispositivo(/ps|playstation|dualsense|dualshock/i.test(d) ? "ps" : "xbox");
      tratar(acao);
    });
    document.addEventListener("sgs:focus", () => som.tocar("foco"));
    document.addEventListener("click", (e) => {
      if ((e.target as HTMLElement).closest?.(".btn:not(:disabled)")) som.tocar("confirmar");
    });
  }, []);

  // Foco inicial depois que a tela muda.
  const pronto = estado !== null;
  const semPasta = pronto && estado.pasta_sync === null;
  useEffect(() => {
    if (!pronto || camadas.length > 0 || !raiz.current) return;
    // Só escolhe um foco inicial se nada válido estiver focado; ao fechar um diálogo, o motor de foco
    // já devolveu o foco ao botão de origem e isso não pode ser desfeito.
    const atual = focus.current() as HTMLElement | null;
    if (!atual || !atual.isConnected || !raiz.current.contains(atual)) focus.focusFirst(raiz.current);
  }, [pronto, semPasta, jogos.length, camadas.length]);

  // Com algum jogo aberto, o som da interface fica em silêncio.
  useEffect(() => {
    som.definirJogando(Object.values(cards).some((c) => c === "jogando"));
  }, [cards]);

  // Andamento do jogo, vindo do Rust.
  useEffect(() => {
    let tirar = () => {};
    void aoMudarEstadoDoJogo((id, st) => {
      setCards((c) => ({ ...c, [id]: st === "pronto" || st === "erro" ? undefined : (st as EstadoCard) }));
    }).then((f) => (tirar = f));
    return () => tirar();
  }, []);

  function mostrar(texto: string, erro = false) {
    som.tocar(erro ? "erro" : "sucesso");
    setAviso({ texto, erro });
    window.setTimeout(() => setAviso((a) => (a?.texto === texto ? null : a)), 9000);
  }

  async function jogar(jogo: Game, opcoes?: OpcoesJogar) {
    // Jogo já aberto (ou abrindo): Jogar de novo não faz nada.
    if (cards[jogo.id] && !opcoes) return;
    setCards((c) => ({ ...c, [jogo.id]: "sincronizando" }));
    try {
      const r = await api.jogar(jogo.id, opcoes);
      if (r.kind === "conflict") abrir({ tipo: "conflito", jogo });
      else if (r.kind === "locked") abrir({ tipo: "travado", jogo, pc: r.pc });
      // Deu tudo certo: nenhum aviso. Só avisa quando o save não foi sincronizado.
      else if (!r.synced) mostrar(`${jogo.name}: ${r.notice}`);
    } catch (err) {
      const msg = textoDoErro(err);
      // Pasta de sync fora do ar: oferece jogar mesmo assim, sem tocar nos saves.
      if (msg.includes("não está disponível")) abrir({ tipo: "semsync", jogo, mensagem: msg });
      else mostrar(`${jogo.name}: ${msg}`, true);
    } finally {
      setCards((c) => ({ ...c, [jogo.id]: undefined }));
      void carregar();
    }
  }

  function parar(jogo: Game) {
    abrir({ tipo: "parar", jogo });
  }

  // Atalhos de tela (teclado, controle e cliques na legenda). Só valem sem diálogo aberto.
  acoes.current = (acao: string) => {
    if (camadas.length > 0 || !estado || estado.pasta_sync === null) return;
    const atual = focus.current() as HTMLElement | null;
    const idFocado = Number(atual?.closest<HTMLElement>("[data-game], [data-game-ctx]")?.dataset.game ?? atual?.closest<HTMLElement>("[data-game-ctx]")?.dataset.gameCtx);
    const jogo = jogos.find((j) => j.id === idFocado) ?? destaque;
    const irPara = (t: "inicio" | "biblioteca") => {
      setTela(t);
      window.requestAnimationFrame(() => {
        const alvo = raiz.current?.querySelector("main");
        if (alvo) focus.focusFirst(alvo);
      });
    };
    switch (acao) {
      case "confirmar":
        atual?.click();
        break;
      case "home":
        irPara("inicio");
        break;
      case "library":
        irPara("biblioteca");
        break;
      case "back":
        if (tela === "biblioteca") irPara("inicio");
        break;
      case "addGame":
        abrir({ tipo: "adicionar" });
        break;
      case "settings":
        abrir({ tipo: "config" });
        break;
      case "options":
        if (jogo) abrir({ tipo: "opcoes", jogo });
        break;
      case "details":
        if (jogo) abrir({ tipo: "versoes", jogo });
        break;
      case "prevTab":
      case "nextTab":
        if (tela === "inicio" && jogos.length > 0) {
          const n = (secao + (acao === "nextTab" ? 1 : -1) + SECOES.length) % SECOES.length;
          setSecao(n);
          const primeiro = jogosDaSecao(n, jogos)[0];
          if (primeiro) setDestaque(primeiro);
          window.requestAnimationFrame(() => {
            const faixa = raiz.current?.querySelector<HTMLElement>(".shelf");
            if (faixa) {
              faixa.scrollLeft = 0;
              focus.focusFirst(faixa);
            }
          });
        }
        break;
    }
  };


  if (falha) {
    return (
      <div ref={raiz} className="app tela">
        <main className="vazio">
          <Icone nome="alert" className="vazio-ico" />
          <h1>Não foi possível abrir o app</h1>
          <p role="alert">{falha}</p>
          <button className="btn primario" data-f data-autofocus onClick={() => void carregar()}>
            Tentar de novo
          </button>
        </main>
      </div>
    );
  }

  if (!estado) {
    return (
      <div ref={raiz} className="app tela">
        <p className="carregando" role="status">
          Abrindo…
        </p>
      </div>
    );
  }

  if (semPasta) {
    return (
      <div ref={raiz} className="app tela">
        <Wizard nomePc={estado.nome_pc} onPronto={() => void carregar()} />
      </div>
    );
  }

  const hora = agora.toLocaleTimeString("pt-BR", { hour: "2-digit", minute: "2-digit" });
  const nomeFundo = destaque?.name ?? "Sync Game Save";
  const estiloFundo = { "--ph": corDoNome(nomeFundo) } as CSSProperties;
  const semSave = jogos.filter((j) => j.save_paths.length > 0 && j.last_sync_at === null).length;
  const ocupados = Object.values(cards).filter(Boolean).length;
  const [icoNuvem, tomNuvem, textoNuvem] = !estado.pasta_sync
    ? ["cloudOff", "warn", "Sem pasta de sincronização"]
    : ocupados
      ? ["cloudSync", "info", "Sincronizando saves"]
      : semSave
        ? ["cloudSync", "warn", semSave === 1 ? "1 jogo ainda não sincronizado" : `${semSave} jogos ainda não sincronizados`]
        : ["cloudCheck", "ok", "Nuvem: tudo em dia"];
  const g = GLYPHS[dispositivo] as Record<string, string>;
  const legenda: [string, string][] =
    jogos.length === 0
      ? [["addGame", "Adicionar jogo"]]
      : tela === "inicio"
      ? [["confirm", "Jogar"], ["options", "Opções"], ["details", "Versões"], ["nextTab", "Seção"], ["library", "Biblioteca"]]
      : [["confirm", "Abrir"], ["options", "Opções"], ["details", "Versões"], ["back", "Início"]];
  const irPara = (t: "inicio" | "biblioteca") => acoes.current(t === "inicio" ? "home" : "library");

  return (
    <div ref={raiz} className="app" data-tela={tela} data-input={dispositivo}>
      <div className="fundo" aria-hidden="true" style={estiloFundo}>
        <div
          key={arteFundo ?? "sem-arte"}
          className="fundo-arte"
          data-foto={arteFundo ? "" : undefined}
          style={arteFundo ? { backgroundImage: `url("${arteFundo}")` } : undefined}
        />
        <div className="fundo-veu" />
      </div>

      <aside className="rail" aria-label="Navegação principal">
        <nav className="rail-mid" data-group="rail" aria-label="Telas">
          <button className="rail-btn" data-f aria-label="Início" aria-current={tela === "inicio" ? "page" : undefined} onClick={() => irPara("inicio")}>
            <Icone nome="home" />
            <span className="rail-rotulo">Início</span>
          </button>
          <button className="rail-btn" data-f aria-label="Biblioteca" aria-current={tela === "biblioteca" ? "page" : undefined} onClick={() => irPara("biblioteca")}>
            <Icone nome="library" />
            <span className="rail-rotulo">Biblioteca</span>
          </button>
          <button className="rail-btn" data-f aria-label="Meu progresso" onClick={() => abrir({ tipo: "conquistas-hub" })}>
            <Icone nome="trophy" />
            <span className="rail-rotulo">Meu progresso</span>
          </button>
          <button className="rail-btn" data-f aria-label="Adicionar jogo" onClick={() => abrir({ tipo: "adicionar" })}>
            <Icone nome="plus" />
            <span className="rail-rotulo">Adicionar jogo</span>
          </button>
        </nav>
        <div className="rail-bottom" data-group="rail-rodape" aria-label="Ajuda e configurações">
          <button className="rail-btn" data-f aria-label="Sobre" onClick={() => abrir({ tipo: "sobre" })}>
            <Icone nome="info" />
            <span className="rail-rotulo">Sobre</span>
          </button>
          <button className="rail-btn" data-f aria-label="Configurações" onClick={() => abrir({ tipo: "config" })}>
            <Icone nome="settings" />
            <span className="rail-rotulo">Configurações</span>
          </button>
        </div>
      </aside>

      <header className="topbar">
        <div className="topbar-esq">
          <span className="relogio" aria-hidden="true">
            {hora}
          </span>
          <button className="nuvem" data-f aria-label={textoNuvem} title="Abrir as configurações" onClick={() => abrir({ tipo: "config" })}>
            <span className={`tone-${tomNuvem}`}>
              <Icone nome={icoNuvem} />
            </span>
            <span>{textoNuvem}</span>
          </button>
        </div>
        <div className="topbar-dir" data-group="topbar">
          {temas && <label className="tema-rapido">
            <span className="sr-only">Tema visual</span>
            <select data-f aria-label="Tema visual" value={temas.atual} onChange={(e) => void trocarTema(e.target.value)}>
              {temas.temas.map((tema) => <option key={tema.id} value={tema.id}>{tema.name}</option>)}
            </select>
          </label>}
          {atualizacao && (
            <button
              className="btn fraco btn-pequeno"
              data-f
              aria-label={`Atualização ${atualizacao.versao} disponível`}
              onClick={() => abrir({ tipo: "sobre" })}
              title={`A versão ${atualizacao.versao} do app está disponível.`}
            >
              <Icone nome="cloudDown" />
              <span>Atualização {atualizacao.versao}</span>
            </button>
          )}
          <span className="pc-nome" title="Este computador">
            <Icone nome="monitor" />
            <span>{estado.nome_pc}</span>
          </span>
        </div>
      </header>

      <main className="conteudo">
        {estado.operacoes_interrompidas.length > 0 && (
          <p className="faixa aviso" role="alert">
            Uma operação não terminou da última vez: {estado.operacoes_interrompidas.join(", ")}. Confira o save desse jogo em “Versões” antes de jogar.
          </p>
        )}
        {tela === "inicio" ? (
          <Inicio
            key={String(estado.steamgriddb_configurada)}
            jogos={jogos}
            estados={cards}
            destaque={destaque}
            secao={secao}
            onSecao={setSecao}
            onDestacar={setDestaque}
            onJogar={(j) => void jogar(j)}
            onParar={(j) => void parar(j)}
            onVersoes={(j) => abrir({ tipo: "versoes", jogo: j })}
            onConquistas={(j) => abrir({ tipo: "conquistas", jogo: j })}
            onOpcoes={(j) => abrir({ tipo: "opcoes", jogo: j })}
            onAdicionar={() => abrir({ tipo: "adicionar" })}
            onBiblioteca={() => irPara("biblioteca")}
          />
        ) : (
          <Biblioteca
            key={String(estado.steamgriddb_configurada)}
            jogos={jogos}
            estados={cards}
            onAbrir={(j) => abrir({ tipo: "opcoes", jogo: j })}
            onDestacar={setDestaque}
            onAdicionar={() => abrir({ tipo: "adicionar" })}
          />
        )}
      </main>

      {camadas.length === 0 && (
        <footer className="legend" aria-label="Comandos disponíveis">
          {legenda
            .filter(([a]) => g[a])
            .map(([a, rotulo]) => (
              <button key={a} className="legend-item" tabIndex={-1} onClick={() => acoes.current(a === "confirm" ? "confirmar" : a)}>
                {a === "nextTab" && g.prevTab && <kbd className="glyph g-prevTab">{g.prevTab}</kbd>}
                <kbd className={`glyph g-${a}${["confirm", "back", "options"].includes(a) && dispositivo !== "keyboard" ? " face" : ""}`}>{g[a]}</kbd>
                <span>{rotulo}</span>
              </button>
            ))}
        </footer>
      )}

      {aviso && (
        <p className={`toast${aviso.erro ? " erro" : ""}`} role={aviso.erro ? "alert" : "status"}>
          {aviso.texto}
        </p>
      )}

      {camadas.map((c, i) => {
        const ultima = i === camadas.length - 1;
        if (!ultima) return null;
        switch (c.tipo) {
          case "adicionar":
            return (
              <AddGame
                key={i}
                catalogoPronto={estado.catalogo_pronto}
                onClose={fechar}
                onAdicionado={() => {
                  fecharTodas();
                  void carregar();
                }}
              />
            );
          case "config":
            return <Configuracoes key={i} estado={estado} onClose={fechar} onMudou={() => void carregar()} onSobre={() => abrir({ tipo: "sobre" })} />;
          case "sobre":
            return <Sobre key={i} onClose={fechar} />;
          case "conquistas":
            return <Conquistas key={i} jogo={c.jogo} onClose={fechar} />;
          case "conquistas-hub":
            return <ConquistasHub key={i} onClose={fechar} />;
          case "opcoes":
            return (
              <Opcoes
                key={i}
                jogo={c.jogo}
                onClose={fechar}
                onJogar={() => {
                  fecharTodas();
                  void jogar(c.jogo);
                }}
                onVersoes={() => abrir({ tipo: "versoes", jogo: c.jogo })}
                onConquistas={() => abrir({ tipo: "conquistas", jogo: c.jogo })}
                onRemover={() => abrir({ tipo: "remover", jogo: c.jogo })}
                onEditar={() => abrir({ tipo: "editar", jogo: c.jogo })}
              />
            );
          case "editar":
            return <EditarJogo key={i} jogo={c.jogo} onClose={fechar} onSaved={() => { fecharTodas(); void carregar(); mostrar(`${c.jogo.name}: alterações salvas.`); }} />;
          case "versoes":
            return <Versoes key={i} jogo={c.jogo} onClose={fechar} />;
          case "remover":
            return (
              <Remover
                key={i}
                jogo={c.jogo}
                onClose={fechar}
                onConfirmar={async () => {
                  fecharTodas();
                  try {
                    await api.removerJogo(c.jogo.id);
                    mostrar(`${c.jogo.name} saiu da biblioteca. Os saves continuam onde estavam.`);
                  } catch (err) {
                    mostrar(textoDoErro(err), true);
                  }
                  void carregar();
                }}
              />
            );
          case "parar":
            return <PararJogo key={i} jogo={c.jogo} onClose={fechar} />;
          case "conflito":
            return (
              <Conflito
                key={i}
                jogo={c.jogo}
                onClose={fechar}
                onEscolher={(escolha) => {
                  fecharTodas();
                  void jogar(c.jogo, { conflito: escolha });
                }}
              />
            );
          case "travado":
            return (
              <Travado
                key={i}
                jogo={c.jogo}
                pc={c.pc}
                onClose={fechar}
                onForcar={() => {
                  fecharTodas();
                  void jogar(c.jogo, { forcar: true });
                }}
              />
            );
          case "semsync":
            return (
              <SemSync
                key={i}
                mensagem={c.mensagem}
                onClose={fechar}
                onSem={() => {
                  fecharTodas();
                  void jogar(c.jogo, { sem_sincronizar: true });
                }}
              />
            );
        }
      })}
    </div>
  );
}
