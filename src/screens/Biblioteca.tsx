import type { Game } from "../api";
import { Icone } from "../Icone";
import { Arte, infoSync } from "./Inicio";

export type EstadoCard = "sincronizando" | "jogando" | "salvando" | "ocupado" | undefined;

export type FiltroLib = "todos" | "sincronizados" | "sem-sync" | "nunca-jogados";
export type OrdemLib = "nome" | "recentes" | "tempo";

type Props = {
  jogos: Game[];
  estados: Record<number, EstadoCard>;
  busca: string;
  filtro: FiltroLib;
  ordem: OrdemLib;
  onBusca: (v: string) => void;
  onFiltro: (v: FiltroLib) => void;
  onOrdem: (v: OrdemLib) => void;
  onAbrir: (j: Game) => void;
  onDestacar: (j: Game) => void;
  onAdicionar: () => void;
};

/** Tira acentos e caixa, para a busca achar "celeste" e "Céleste" do mesmo jeito. */
const limpa = (s: string) => s.normalize("NFD").replace(/[\u0300-\u036f]/g, "").toLowerCase();

const FILTROS: [FiltroLib, string][] = [
  ["todos", "Todos"],
  ["sincronizados", "Sincronizados"],
  ["sem-sync", "Sem sincronização"],
  ["nunca-jogados", "Nunca jogados"],
];

const nomeRecente = (j: Game) => j.last_played_at ?? Number.NEGATIVE_INFINITY;

function listaFiltrada(jogos: Game[], busca: string, filtro: FiltroLib, ordem: OrdemLib): Game[] {
  let lista = jogos;
  if (busca.trim() !== "") lista = lista.filter((j) => limpa(j.name).includes(limpa(busca.trim())));
  if (filtro === "sincronizados") lista = lista.filter((j) => j.last_sync_at !== null);
  else if (filtro === "sem-sync") lista = lista.filter((j) => j.last_sync_at === null);
  else if (filtro === "nunca-jogados") lista = lista.filter((j) => j.last_played_at === null);
  switch (ordem) {
    case "recentes":
      lista = [...lista].sort((a, b) => nomeRecente(b) - nomeRecente(a) || a.name.localeCompare(b.name, "pt-BR"));
      break;
    case "tempo":
      lista = [...lista].sort((a, b) => b.playtime_secs - a.playtime_secs || a.name.localeCompare(b.name, "pt-BR"));
      break;
    default:
      lista = [...lista].sort((a, b) => a.name.localeCompare(b.name, "pt-BR"));
  }
  return lista;
}

export function Biblioteca({ jogos, estados, busca, filtro, ordem, onBusca, onFiltro, onOrdem, onAbrir, onDestacar, onAdicionar }: Props) {
  const filtrando = busca.trim() !== "" || filtro !== "todos";
  const visiveis = listaFiltrada(jogos, busca, filtro, ordem);
  return (
    <div className="lib">
      <div className="lib-head">
        <h1>Biblioteca</h1>
        <span className="lib-count">
          {jogos.length === 0
            ? "Nenhum jogo"
            : filtrando
              ? `${visiveis.length} de ${jogos.length} ${jogos.length === 1 ? "jogo" : "jogos"}`
              : `${jogos.length} ${jogos.length === 1 ? "jogo" : "jogos"}`}
        </span>
      </div>
      {jogos.length === 0 ? (
        <section className="empty" aria-labelledby="vazio-titulo">
          <Icone nome="library" className="ico empty-ico" />
          <h2 id="vazio-titulo">Sua biblioteca está vazia</h2>
          <p>Adicione um jogo para o app guardar o save dele na pasta de sincronização.</p>
          <div className="empty-actions">
            <button className="btn btn-play" data-f data-autofocus onClick={onAdicionar}>
              <Icone nome="plus" />
              <span>Adicionar jogo</span>
            </button>
          </div>
        </section>
      ) : (
        <>
          <div className="lib-bar" data-group="lib-tools" aria-label="Busca, filtros e ordenação">
            <label className="lib-busca">
              <Icone nome="search" />
              <input
                data-f
                type="text"
                autoComplete="off"
                placeholder="Buscar jogo pelo nome"
                aria-label="Buscar jogo pelo nome"
                value={busca}
                onChange={(e) => onBusca(e.target.value)}
              />
            </label>
            <div className="lib-filtros" role="group" aria-label="Filtros">
              {FILTROS.map(([v, rotulo]) => (
                <button key={v} className={`btn-pequeno btn${filtro === v ? " primario" : " fraco"}`} data-f data-libtab={v} aria-pressed={filtro === v} onClick={() => onFiltro(v)}>
                  {rotulo}
                </button>
              ))}
            </div>
            <label className="lib-ordem">
              Ordenar por
              <select data-f value={ordem} onChange={(e) => onOrdem(e.target.value as OrdemLib)}>
                <option value="nome">Nome</option>
                <option value="recentes">Jogado recentemente</option>
                <option value="tempo">Mais tempo de jogo</option>
              </select>
            </label>
          </div>
          {visiveis.length === 0 ? (
            <section className="empty" aria-labelledby="sem-resultado-titulo">
              <Icone nome="library" className="ico empty-ico" />
              <h2 id="sem-resultado-titulo">Nenhum jogo com estes filtros</h2>
              <p>Nenhum jogo combina com a busca e os filtros escolhidos.</p>
              <div className="empty-actions">
                <button
                  className="btn fraco"
                  data-f
                  data-autofocus
                  onClick={() => {
                    onBusca("");
                    onFiltro("todos");
                  }}
                >
                  <Icone nome="x" />
                  <span>Limpar filtros</span>
                </button>
              </div>
            </section>
          ) : (
            <ul className="lib-grid" data-group="lib-grid" aria-label="Seus jogos">
              {visiveis.map((j, i) => {
                const s = infoSync(j, estados[j.id]);
                return (
                  <li key={j.id}>
                    <button
                      className="card"
                      data-f
                      data-game={j.id}
                      data-autofocus={i === 0 || undefined}
                      title={j.name}
                      aria-label={`${j.name}. ${s.rotulo}`}
                      onFocus={() => onDestacar(j)}
                      onMouseEnter={() => onDestacar(j)}
                      onClick={() => onAbrir(j)}
                    >
                      <span className="card-art">
                        <Arte jogo={j} tipo="cover" />
                      </span>
                      <span className="card-name">{j.name}</span>
                      <span className={`card-sub tone-${s.tom}`}>
                        <Icone nome={s.icone} />
                        <span>{s.rotulo}</span>
                      </span>
                    </button>
                  </li>
                );
              })}
            </ul>
          )}
        </>
      )}
    </div>
  );
}
