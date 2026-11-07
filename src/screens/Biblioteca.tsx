import type { Game } from "../api";
import { Icone } from "../Icone";
import { Arte, infoSync } from "./Inicio";

export type EstadoCard = "sincronizando" | "jogando" | "salvando" | "ocupado" | undefined;

type Props = {
  jogos: Game[];
  estados: Record<number, EstadoCard>;
  onAbrir: (j: Game) => void;
  onDestacar: (j: Game) => void;
  onAdicionar: () => void;
};

export function Biblioteca({ jogos, estados, onAbrir, onDestacar, onAdicionar }: Props) {
  const ordem = [...jogos].sort((a, b) => a.name.localeCompare(b.name, "pt-BR"));
  return (
    <div className="lib">
      <div className="lib-head">
        <h1>Biblioteca</h1>
        <span className="lib-count">{jogos.length === 0 ? "Nenhum jogo" : `${jogos.length} ${jogos.length === 1 ? "jogo" : "jogos"}`}</span>
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
        <ul className="lib-grid" data-group="lib-grid" aria-label="Seus jogos">
          {ordem.map((j, i) => {
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
    </div>
  );
}
