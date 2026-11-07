import { useEffect, useRef, useState, type CSSProperties } from "react";
import type { Game } from "../api";
import { duracao, haQuanto } from "../format";
import { Icone } from "../Icone";
import { useArte } from "../useCapa";
import type { EstadoCard } from "./Biblioteca";

export type Tom = "ok" | "info" | "warn" | "danger" | "dim";
export type InfoSync = { tom: Tom; icone: string; rotulo: string; detalhe: string };

const andamento: Record<string, string> = {
  sincronizando: "Sincronizando…",
  jogando: "Jogando",
  salvando: "Enviando o save…",
  ocupado: "Aguarde…",
};

/** Estado do save do jogo, em texto curto e em detalhe. */
export function infoSync(j: Game, st?: EstadoCard): InfoSync {
  if (st) return { tom: "info", icone: "cloudSync", rotulo: andamento[st], detalhe: "Não feche o app enquanto isso acontece." };
  if (j.save_paths.length === 0)
    return { tom: "dim", icone: "cloudOff", rotulo: "Sem sincronização", detalhe: "O jogo abre, mas o app não sabe onde fica o save." };
  if (j.last_sync_at === null)
    return { tom: "warn", icone: "cloudSync", rotulo: "Ainda não sincronizado", detalhe: "O save vai para a nuvem quando você fechar o jogo." };
  return { tom: "ok", icone: "cloudCheck", rotulo: "Sincronizado", detalhe: `Última versão salva na nuvem ${haQuanto(j.last_sync_at)}.` };
}

export function corDoNome(nome: string): string {
  let h = 0;
  for (const c of nome) h = (h * 31 + c.charCodeAt(0)) % 360;
  return `hsl(${h} 55% 42%)`;
}

export function iniciais(nome: string): string {
  const palavras = nome.split(/\s+/).filter((w) => /^[\p{L}\d]/u.test(w) && w.length > 2);
  const r = palavras
    .slice(0, 2)
    .map((w) => w[0])
    .join("")
    .toUpperCase();
  return r || nome.trim().charAt(0).toUpperCase() || "?";
}

/** Arte do jogo com espaço reservado (cor e iniciais) quando não há imagem. */
export function Arte({ jogo, tipo }: { jogo: Game; tipo: "header" | "cover" }) {
  const src = useArte(jogo, tipo);
  const [falhou, setFalhou] = useState(false);
  useEffect(() => setFalhou(false), [src]);
  if (src && !falhou) return <img src={src} alt="" decoding="async" onError={() => setFalhou(true)} />;
  return (
    <span className="ph" style={{ "--ph": corDoNome(jogo.name) } as CSSProperties}>
      <span>{iniciais(jogo.name)}</span>
    </span>
  );
}

export const SECOES = [
  { id: "recentes", titulo: "Jogados recentemente", vazio: "Nenhum jogo foi aberto pelo app ainda." },
  { id: "adicionados", titulo: "Adicionados", vazio: "" },
  { id: "atencao", titulo: "Precisam de atenção", vazio: "Tudo certo: todos os jogos têm o save sincronizado." },
] as const;

export function jogosDaSecao(secao: number, jogos: Game[]): Game[] {
  const id = SECOES[secao].id;
  if (id === "adicionados") return [...jogos].sort((a, b) => b.id - a.id);
  if (id === "atencao") return jogos.filter((j) => j.save_paths.length === 0 || j.last_sync_at === null);
  return [...jogos].sort((a, b) => (b.last_played_at ?? -1) - (a.last_played_at ?? -1) || b.id - a.id);
}

function Heroi({
  jogo,
  st,
  onJogar,
  onParar,
  onVersoes,
  onConquistas,
  onOpcoes,
  onAdicionar,
}: {
  jogo: Game | null;
  st?: EstadoCard;
  onJogar: () => void;
  onParar: () => void;
  onVersoes: () => void;
  onConquistas: () => void;
  onOpcoes: () => void;
  onAdicionar: () => void;
}) {
  const logo = useArte(jogo, "logo");
  const [logoFalhou, setLogoFalhou] = useState(false);
  useEffect(() => setLogoFalhou(false), [logo]);

  if (!jogo) {
    return (
      <div className="hero" key="vazio">
        <div className="hero-logo">
          <h1 className="hero-title">Sua biblioteca está vazia</h1>
        </div>
        <p className="hero-desc">Adicione um jogo para abrir pelo app e manter o save sincronizado com a sua pasta da nuvem.</p>
        <div className="hero-actions" data-group="hero-actions">
          <button className="btn btn-play" data-f data-autofocus onClick={onAdicionar}>
            <Icone nome="plus" />
            <span>Adicionar jogo</span>
          </button>
        </div>
      </div>
    );
  }

  const s = infoSync(jogo, st);
  const comLogo = !!logo && !logoFalhou;
  return (
    <div className="hero" key={jogo.id} data-game-ctx={jogo.id}>
      <div className="hero-logo">
        {comLogo && <img src={logo} alt="" onError={() => setLogoFalhou(true)} />}
        <h1 className={`hero-title${comLogo ? " sr-only" : ""}`}>{jogo.name}</h1>
      </div>
      <div className="hero-meta">
        <span className="meta-ico" title="Tempo de jogo">
          <Icone nome="clock" />
        </span>
        <span>{duracao(jogo.playtime_secs)}</span>
        {jogo.last_played_at !== null && (
          <>
            <span className="dot">•</span>
            <span>Jogado {haQuanto(jogo.last_played_at)}</span>
          </>
        )}
      </div>
      <div className="hero-actions" data-group="hero-actions">
        {st === "jogando" ? (
          <button className="btn btn-play" data-f onClick={onParar} title="Fecha o jogo e sincroniza o save">
            <Icone nome="stop" />
            <span>Parar jogo</span>
          </button>
        ) : (
          <button className="btn btn-play" data-f disabled={!!st} onClick={onJogar}>
            <Icone nome="play" />
            <span>Jogar</span>
          </button>
        )}
        <button className="btn btn-glass" data-f disabled={!!st} onClick={onVersoes}>
          <Icone nome="history" />
          <span>Versões</span>
        </button>
        <button className="btn btn-glass" data-f disabled={!!st} onClick={onConquistas}>
          <Icone nome="trophy" />
          <span>Conquistas</span>
        </button>
        <button className="btn btn-glass btn-icon" data-f disabled={!!st} onClick={onOpcoes} aria-label="Mais opções" title="Mais opções">
          <Icone nome="more" />
        </button>
      </div>
      <div className="hero-status" role="status">
        <span className={`tone-${s.tom}`}>
          <Icone nome={s.icone} />
        </span>
        <strong className={`tone-${s.tom}`}>{s.rotulo}</strong>
        <span className="hero-status-detail">{s.detalhe}</span>
      </div>
    </div>
  );
}

function Capsula({ jogo, st, autofoco, onFoco, onAbrir }: { jogo: Game; st?: EstadoCard; autofoco?: boolean; onFoco: () => void; onAbrir: () => void }) {
  const s = infoSync(jogo, st);
  const alerta = s.tom === "warn" || s.tom === "danger";
  return (
    <button
      className="cap"
      data-f
      data-game={jogo.id}
      data-autofocus={autofoco || undefined}
      aria-label={`${jogo.name}. ${s.rotulo}`}
      title={jogo.name}
      onFocus={onFoco}
      onMouseEnter={onFoco}
      onClick={onAbrir}
    >
      <span className="cap-art">
        <Arte jogo={jogo} tipo="header" />
      </span>
      {alerta && (
        <span className={`cap-flag tone-${s.tom}`}>
          <Icone nome={s.icone} />
        </span>
      )}
      <span className="cap-bar">
        <span className="cap-name">{jogo.name}</span>
        <span className={`cap-sync tone-${s.tom}`}>
          <Icone nome={s.icone} />
        </span>
      </span>
    </button>
  );
}

type Props = {
  jogos: Game[];
  estados: Record<number, EstadoCard>;
  destaque: Game | null;
  secao: number;
  onSecao: (i: number) => void;
  onDestacar: (j: Game) => void;
  onJogar: (j: Game) => void;
  onParar: (j: Game) => void;
  onVersoes: (j: Game) => void;
  onConquistas: (j: Game) => void;
  onOpcoes: (j: Game) => void;
  onAdicionar: () => void;
  onBiblioteca: () => void;
};

export function Inicio({ jogos, estados, destaque, secao, onSecao, onDestacar, onJogar, onParar, onVersoes, onConquistas, onOpcoes, onAdicionar, onBiblioteca }: Props) {
  const faixa = useRef<HTMLDivElement>(null);
  const lista = jogosDaSecao(secao, jogos);
  const sec = SECOES[secao];

  // Rolagem vertical do mouse anda pela faixa.
  useEffect(() => {
    const el = faixa.current;
    if (!el) return;
    const rolar = (e: WheelEvent) => {
      if (Math.abs(e.deltaY) > Math.abs(e.deltaX)) {
        el.scrollLeft += e.deltaY;
        e.preventDefault();
      }
    };
    el.addEventListener("wheel", rolar, { passive: false });
    return () => el.removeEventListener("wheel", rolar);
  }, []);

  return (
    <div className="home">
      <Heroi
        jogo={destaque}
        st={destaque ? estados[destaque.id] : undefined}
        onJogar={() => destaque && onJogar(destaque)}
        onParar={() => destaque && onParar(destaque)}
        onVersoes={() => destaque && onVersoes(destaque)}
        onConquistas={() => destaque && onConquistas(destaque)}
        onOpcoes={() => destaque && onOpcoes(destaque)}
        onAdicionar={onAdicionar}
      />
      {jogos.length > 0 && (
        <>
          <div className="shelf-head">
            <h2 className="shelf-title">{sec.titulo}</h2>
            <div className="shelf-tabs" role="tablist" aria-label="Seções do início">
              {SECOES.map((s, i) => (
                <button key={s.id} className="shelf-tab" role="tab" aria-selected={i === secao} tabIndex={-1} onClick={() => onSecao(i)}>
                  {s.titulo}
                </button>
              ))}
            </div>
          </div>
          <div className="shelf" ref={faixa} data-scroll="x" data-group="shelf">
            {lista.length === 0 && (
              <div className="shelf-empty" data-f tabIndex={0}>
                <Icone nome="check" />
                <span>{sec.vazio}</span>
              </div>
            )}
            {lista.map((j, i) => (
              <Capsula
                key={j.id}
                jogo={j}
                st={estados[j.id]}
                autofoco={i === 0}
                onFoco={() => onDestacar(j)}
                onAbrir={() => onJogar(j)}
              />
            ))}
            <button className="cap cap-tile" data-f onClick={onBiblioteca}>
              <span className="cap-art">
                <Icone nome="library" />
              </span>
              <span className="cap-bar">
                <span className="cap-name">Ver biblioteca</span>
              </span>
            </button>
            <button className="cap cap-tile" data-f onClick={onAdicionar}>
              <span className="cap-art">
                <Icone nome="plus" />
              </span>
              <span className="cap-bar">
                <span className="cap-name">Adicionar jogo</span>
              </span>
            </button>
          </div>
        </>
      )}
    </div>
  );
}
