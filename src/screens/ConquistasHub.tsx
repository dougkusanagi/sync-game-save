import { useEffect, useState } from "react";
import { Dialog } from "../Dialog";
import { api, textoDoErro, type ConquistasHub as Dados, type MarcoHub } from "../api";

function quantidade(n: number, unidade: string): string {
  if (unidade === "h") return `${n} h`;
  if (unidade === "jogo" || unidade === "jogos") return `${n} ${n === 1 ? "jogo" : "jogos"}`;
  if (unidade === "syncs") return `${n} ${n === 1 ? "sincronização" : "sincronizações"}`;
  return `${n} ${unidade}`;
}

function Marco({ marco }: { marco: MarcoHub }) {
  const pct = marco.alvo === 0 ? 0 : Math.round((marco.progresso * 100) / marco.alvo);
  return (
    <li className="marco" data-nivel={marco.nivel} data-bloqueada={!marco.desbloqueada || undefined}>
      <div className="marco-texto">
        <strong>{marco.nome}</strong>
        <small>{marco.descricao}</small>
        <span className="marco-progresso">
          <span className="barra" aria-hidden="true">
            <span style={{ width: `${marco.desbloqueada ? 100 : pct}%` }} />
          </span>
          <span>{marco.desbloqueada ? "Concluído" : `${quantidade(marco.progresso, marco.unidade)} de ${quantidade(marco.alvo, marco.unidade)}`}</span>
        </span>
      </div>
      <span className={`selo${marco.desbloqueada ? " ok" : ""}`} data-nivel={marco.nivel}>
        {marco.desbloqueada ? "Desbloqueada" : marco.nivel}
      </span>
    </li>
  );
}

/** Marcos de uso do HUB, iguais para qualquer jogo. */
export function ConquistasHub({ onClose }: { onClose: () => void }) {
  const [dados, setDados] = useState<Dados | null>(null);
  const [erro, setErro] = useState<string | null>(null);
  const [ocupado, setOcupado] = useState(false);

  async function carregar() {
    setOcupado(true);
    setErro(null);
    try {
      setDados(await api.conquistasHub());
    } catch (e) {
      setErro(textoDoErro(e));
    } finally {
      setOcupado(false);
    }
  }

  useEffect(() => {
    void carregar();
  }, []);

  return (
    <Dialog titulo="Conquistas do HUB" onClose={onClose} focoKey={dados?.desbloqueadas ?? -1} largo>
      {!dados && !erro && <p role="status">Somando o seu progresso…</p>}
      {erro && (
        <div className="coluna">
          <p className="erro" role="alert">
            {erro}
          </p>
          <div className="acoes">
            <button className="btn" data-f onClick={() => void carregar()}>
              Tentar de novo
            </button>
          </div>
        </div>
      )}
      {dados && (
        <>
          <div className="resumo-conquistas">
            <div className="resumo-hub-numero">
              <strong>{dados.desbloqueadas}</strong>
              <span>de {dados.total}</span>
            </div>
            <div>
              <h3>
                {dados.desbloqueadas === dados.total
                  ? "Todos os marcos do HUB foram desbloqueados."
                  : dados.desbloqueadas === 0
                    ? "Seu progresso começa no primeiro jogo."
                    : `Faltam ${dados.total - dados.desbloqueadas} marcos para completar.`}
              </h3>
              <p className="dica">Os marcos valem para todos os jogos da biblioteca e não podem ser perdidos.</p>
            </div>
            <button className="btn fraco" data-f disabled={ocupado} onClick={() => void carregar()}>
              {ocupado ? "Atualizando…" : "Atualizar"}
            </button>
          </div>

          <dl className="stats-hub">
            <div>
              <dt>Horas jogadas</dt>
              <dd>{dados.horas} h</dd>
            </div>
            <div>
              <dt>Jogos na biblioteca</dt>
              <dd>{dados.jogos}</dd>
            </div>
            <div>
              <dt>Sequência atual</dt>
              <dd>
                {dados.sequencia_atual} <small>melhor: {dados.melhor_sequencia}</small>
              </dd>
            </div>
            <div>
              <dt>Jogos completos</dt>
              <dd>{dados.jogos_completos}</dd>
            </div>
          </dl>

          <ul className="conquistas-lista hub-lista" aria-label="Marcos do HUB">
            {dados.conquistas.map((m) => (
              <Marco key={m.id} marco={m} />
            ))}
          </ul>
        </>
      )}
    </Dialog>
  );
}
