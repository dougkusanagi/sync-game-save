import { useEffect, useState } from "react";
import { convertFileSrc } from "@tauri-apps/api/core";
import { api, noTauri, type Game } from "../api";
import { duracao, haQuanto } from "../format";

export type EstadoCard = "sincronizando" | "jogando" | "salvando" | "ocupado" | undefined;

const rotulo: Record<string, string> = {
  sincronizando: "Sincronizando…",
  jogando: "Jogando",
  salvando: "Enviando o save…",
  ocupado: "Aguarde…",
};

function Capa({ jogo }: { jogo: Game }) {
  const [src, setSrc] = useState<string | null>(null);
  useEffect(() => {
    let vivo = true;
    api
      .capaDoJogo(jogo.id)
      .then((p) => {
        if (!vivo || !p || noTauri) return;
        setSrc(convertFileSrc(p));
      })
      .catch(() => {});
    return () => {
      vivo = false;
    };
  }, [jogo.id]);
  const inicial = jogo.name.trim().charAt(0).toUpperCase() || "?";
  return (
    <div className="capa" aria-hidden="true">
      {src ? <img src={src} alt="" onError={() => setSrc(null)} /> : <span>{inicial}</span>}
    </div>
  );
}

type Props = {
  jogos: Game[];
  estados: Record<number, EstadoCard>;
  onJogar: (j: Game) => void;
  onConquistas: (j: Game) => void;
  onVersoes: (j: Game) => void;
  onRemover: (j: Game) => void;
  onAdicionar: () => void;
};

export function Biblioteca({ jogos, estados, onJogar, onConquistas, onVersoes, onRemover, onAdicionar }: Props) {
  if (jogos.length === 0) {
    return (
      <section className="vazio" aria-labelledby="vazio-titulo">
        <h2 id="vazio-titulo">Sua biblioteca está vazia</h2>
        <p>Adicione um jogo para o app guardar o save dele na pasta de sincronização.</p>
        <button className="btn primario" data-f data-autofocus onClick={onAdicionar}>
          Adicionar jogo
        </button>
      </section>
    );
  }
  return (
    <ul className="grade" aria-label="Seus jogos">
      {jogos.map((j, i) => {
        const st = estados[j.id];
        const semSave = j.save_paths.length === 0;
        return (
          <li key={j.id} className="cartao" data-group>
            <Capa jogo={j} />
            <div className="cartao-info">
              <h2 title={j.name}>{j.name}</h2>
              <p className="meta">
                {duracao(j.playtime_secs)}
                {j.last_sync_at !== null && <> · salvo na nuvem {haQuanto(j.last_sync_at)}</>}
                {j.last_sync_at === null && !semSave && <> · ainda não sincronizado</>}
              </p>
              {semSave && <p className="aviso">Sem pasta de save: o jogo abre, mas o save não é sincronizado.</p>}
              {st && (
                <p className="estado" role="status">
                  {rotulo[st]}
                </p>
              )}
            </div>
            <div className="cartao-acoes">
              <button className="btn primario" data-f data-autofocus={i === 0 || undefined} disabled={!!st} onClick={() => onJogar(j)}>
                {st === "jogando" ? "Jogando" : "Jogar"}
              </button>
              <button className="btn fraco" data-f disabled={!!st} onClick={() => onConquistas(j)}>
                Conquistas
              </button>
              <button className="btn fraco" data-f disabled={!!st} onClick={() => onVersoes(j)}>
                Versões
              </button>
              <button className="btn fraco" data-f disabled={!!st} onClick={() => onRemover(j)}>
                Remover
              </button>
            </div>
          </li>
        );
      })}
    </ul>
  );
}
