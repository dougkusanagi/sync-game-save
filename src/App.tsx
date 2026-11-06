import { useCallback, useEffect, useRef, useState } from "react";
import { api, aoMudarEstadoDoJogo, aoUsarControle, noTauri, textoDoErro, type Estado, type Game, type OpcoesJogar } from "./api";
import * as focus from "./nav/focus.js";
import { initInput } from "./nav/input.js";
import { voltar } from "./voltar";
import { som } from "./som";
import { AddGame } from "./screens/AddGame";
import { Biblioteca, type EstadoCard } from "./screens/Biblioteca";
import { Conflito, Configuracoes, Remover, SemSync, Sobre, Travado, Versoes } from "./screens/Dialogos";
import { Wizard } from "./screens/Wizard";

// Os ouvintes de teclado e controle são globais e só podem ser registrados uma vez
// (o React em desenvolvimento executa os efeitos duas vezes).
let entradaIniciada = false;

type Camada =
  | { tipo: "adicionar" }
  | { tipo: "config" }
  | { tipo: "sobre" }
  | { tipo: "versoes"; jogo: Game }
  | { tipo: "remover"; jogo: Game }
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
      }
    };
    // No app, o controle é lido no Rust (funciona mesmo sem foco no WebView); no navegador, pela Gamepad API.
    initInput({ onDevice: () => {}, onAction: tratar, gamepad: noTauri });
    void aoUsarControle((acao) => tratar(acao));
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
    setCards((c) => ({ ...c, [jogo.id]: "sincronizando" }));
    try {
      const r = await api.jogar(jogo.id, opcoes);
      if (r.kind === "conflict") abrir({ tipo: "conflito", jogo });
      else if (r.kind === "locked") abrir({ tipo: "travado", jogo, pc: r.pc });
      else mostrar(`${jogo.name}: ${r.notice}`);
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

  if (falha) {
    return (
      <div ref={raiz} className="app">
        <main className="vazio">
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
      <div ref={raiz} className="app">
        <p className="carregando" role="status">
          Abrindo…
        </p>
      </div>
    );
  }

  if (semPasta) {
    return (
      <div ref={raiz} className="app">
        <Wizard nomePc={estado.nome_pc} onPronto={() => void carregar()} />
      </div>
    );
  }

  return (
    <div ref={raiz} className="app">
      <header className="topo">
        <h1>Sync Game Save</h1>
        <nav className="topo-acoes" aria-label="Ações">
          <button className="btn primario" data-f onClick={() => abrir({ tipo: "adicionar" })}>
            Adicionar jogo
          </button>
          <button className="btn fraco" data-f onClick={() => abrir({ tipo: "config" })}>
            Configurações
          </button>
        </nav>
      </header>

      {estado.operacoes_interrompidas.length > 0 && (
        <p className="faixa aviso" role="alert">
          Uma operação não terminou da última vez: {estado.operacoes_interrompidas.join(", ")}. Confira o save desse jogo em “Versões” antes de jogar.
        </p>
      )}

      <main>
        <Biblioteca
          jogos={jogos}
          estados={cards}
          onJogar={(j) => void jogar(j)}
          onVersoes={(j) => abrir({ tipo: "versoes", jogo: j })}
          onRemover={(j) => abrir({ tipo: "remover", jogo: j })}
          onAdicionar={() => abrir({ tipo: "adicionar" })}
        />
      </main>

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
