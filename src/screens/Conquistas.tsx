import { useEffect, useMemo, useState } from "react";
import { Dialog } from "../Dialog";
import { api, textoDoErro, type Conquista, type Conquistas as Dados, type Game, type Nivel } from "../api";
import { contarPorNivel, dataDoDesbloqueio, filtrarOrdenar, nivelRotulo, NIVEIS, type Filtro, type Ordem, type Raridade } from "../conquistas";
import { haQuanto } from "../format";

function Anel({ pct, completo }: { pct: number; completo: boolean }) {
  return (
    <div className={`anel${completo ? " completo" : ""}`} role="img" aria-label={`${pct}% das conquistas desbloqueadas`}>
      <svg viewBox="0 0 42 42" aria-hidden="true">
        <circle className="anel-fundo" cx="21" cy="21" r="18" pathLength={100} />
        <circle className="anel-valor" cx="21" cy="21" r="18" pathLength={100} strokeDasharray={`${pct} ${100 - pct}`} />
      </svg>
      <strong aria-hidden="true">{pct}%</strong>
    </div>
  );
}

function Icone({ a }: { a: Conquista }) {
  const [falhou, setFalhou] = useState(false);
  const src = a.unlocked ? a.icon : a.icon_gray || a.icon;
  const inicial = a.name.trim().charAt(0).toUpperCase() || "?";
  return (
    <span className="conquista-icone" aria-hidden="true">
      {src && !falhou ? <img src={src} alt="" onError={() => setFalhou(true)} /> : <span>{inicial}</span>}
    </span>
  );
}

function Item({ a, revelada, onRevelar }: { a: Conquista; revelada: boolean; onRevelar: () => void }) {
  const oculta = a.hidden && !a.unlocked && !revelada;
  const data = dataDoDesbloqueio(a.unlock_time);
  return (
    <li className="conquista" data-nivel={a.tier ?? "sem"} data-bloqueada={!a.unlocked || undefined}>
      <Icone a={a} />
      <div className="conquista-texto">
        <strong>{oculta ? "Conquista oculta" : a.name}</strong>
        <small>{oculta ? "Jogue para descobrir o que fazer." : a.description || "Sem descrição."}</small>
        <span className="conquista-meta">
          {a.unlocked && <span className="selo ok">Desbloqueada{data ? ` em ${data}` : ""}</span>}
          {!a.unlocked && <span className="selo">Bloqueada</span>}
          <span className="selo" data-nivel={a.tier ?? "sem"}>
            {nivelRotulo(a.tier)}
          </span>
          {a.percent !== null && <span>{a.percent.toLocaleString("pt-BR", { maximumFractionDigits: 1 })}% dos jogadores</span>}
        </span>
      </div>
      {oculta && (
        <button className="btn fraco" data-f onClick={onRevelar}>
          Mostrar
        </button>
      )}
    </li>
  );
}

function BarraNiveis({ lista }: { lista: Conquista[] }) {
  const contagem = contarPorNivel(lista);
  return (
    <ul className="niveis" aria-label="Conquistas desbloqueadas por raridade">
      {NIVEIS.map((n: Nivel) => (
        <li key={n} data-nivel={n}>
          <span>{nivelRotulo(n)}</span>
          <strong>{contagem[n]}</strong>
        </li>
      ))}
    </ul>
  );
}

/** Conquistas da Steam do jogo, com cache local e leitura simples do progresso. */
export function Conquistas({ jogo, onClose }: { jogo: Game; onClose: () => void }) {
  const [dados, setDados] = useState<Dados | null>(null);
  const [erro, setErro] = useState<string | null>(null);
  const [ocupado, setOcupado] = useState(false);
  const [filtro, setFiltro] = useState<Filtro>("todas");
  const [raridade, setRaridade] = useState<Raridade>("todas");
  const [ordem, setOrdem] = useState<Ordem>("recentes");
  const [reveladas, setReveladas] = useState<Set<string>>(new Set());

  async function carregar(forcar: boolean) {
    setOcupado(true);
    setErro(null);
    try {
      setDados(await api.conquistas(jogo.id, forcar));
    } catch (e) {
      setErro(textoDoErro(e));
    } finally {
      setOcupado(false);
    }
  }

  useEffect(() => {
    void carregar(false);
  }, [jogo.id]);

  const lista = useMemo(
    () => (dados ? filtrarOrdenar(dados.set.achievements, filtro, raridade, ordem) : []),
    [dados, filtro, raridade, ordem],
  );
  const filtrando = filtro !== "todas" || raridade !== "todas";
  const pct = dados ? Math.round(dados.summary.percent) : 0;

  function revelar(id: string) {
    setReveladas((r) => new Set(r).add(id));
  }

  return (
    <Dialog titulo={`Conquistas de ${jogo.name}`} onClose={onClose} focoKey={dados?.fetched_at ?? -1} largo>
      {!dados && !erro && <p role="status">Buscando as conquistas na Steam…</p>}
      {erro && (
        <div className="coluna">
          <p className="erro" role="alert">
            {erro}
          </p>
          <div className="acoes">
            <button className="btn" data-f onClick={() => void carregar(true)}>
              Tentar de novo
            </button>
          </div>
        </div>
      )}
      {dados && (
        <>
          <div className="resumo-conquistas">
            <Anel pct={pct} completo={dados.summary.complete} />
            <div>
              <h3>
                {dados.summary.unlocked} de {dados.summary.total} conquistas · {pct}%
              </h3>
              <p className="dica">
                {dados.summary.complete
                  ? "Você desbloqueou todas. Jogo completo!"
                  : dados.summary.unlocked === 0
                    ? "Nenhuma desbloqueada ainda. O progresso aparece aqui."
                    : `Faltam ${dados.summary.total - dados.summary.unlocked}. O progresso é atualizado ao jogar.`}
              </p>
              {!dados.set.progress_known && <p className="aviso">A Steam não mostrou o que você desbloqueou. Confira a privacidade do seu perfil.</p>}
              {dados.notice && <p className="aviso">{dados.notice}</p>}
              {dados.from_cache && <p className="dica">Mostrando o que ficou guardado {haQuanto(dados.fetched_at)}.</p>}
            </div>
            <button className="btn fraco" data-f disabled={ocupado} onClick={() => void carregar(true)}>
              {ocupado ? "Atualizando…" : "Atualizar"}
            </button>
          </div>

          <BarraNiveis lista={dados.set.achievements} />

          <div className="filtros" role="group" aria-label="Filtrar conquistas">
            {(
              [
                ["todas", "Todas"],
                ["desbloqueadas", "Desbloqueadas"],
                ["bloqueadas", "Bloqueadas"],
              ] as [Filtro, string][]
            ).map(([v, rotulo]) => (
              <button key={v} className={`btn${filtro === v ? " primario" : " fraco"}`} data-f aria-pressed={filtro === v} onClick={() => setFiltro(v)}>
                {rotulo}
              </button>
            ))}
          </div>
          <div className="linha filtros-select">
            <label className="campo">
              Raridade
              <select data-f value={raridade} onChange={(e) => setRaridade(e.target.value as Raridade)}>
                <option value="todas">Todas</option>
                {NIVEIS.map((n) => (
                  <option key={n} value={n}>
                    {nivelRotulo(n)}
                  </option>
                ))}
              </select>
            </label>
            <label className="campo">
              Ordenar por
              <select data-f value={ordem} onChange={(e) => setOrdem(e.target.value as Ordem)}>
                <option value="recentes">Mais recentes</option>
                <option value="raridade">Raridade</option>
                <option value="nome">Nome</option>
              </select>
            </label>
          </div>

          {lista.length === 0 ? (
            <div className="vazio-curto">
              <p>{filtrando ? "Nenhuma conquista com este filtro." : "Este jogo não tem conquistas na Steam."}</p>
              {filtrando && (
                <button
                  className="btn fraco"
                  data-f
                  onClick={() => {
                    setFiltro("todas");
                    setRaridade("todas");
                  }}
                >
                  Limpar filtros
                </button>
              )}
            </div>
          ) : (
            <ul className="conquistas-lista">
              {lista.map((a) => (
                <Item key={a.id} a={a} revelada={reveladas.has(a.id)} onRevelar={() => revelar(a.id)} />
              ))}
            </ul>
          )}
        </>
      )}
    </Dialog>
  );
}
