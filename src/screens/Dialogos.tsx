import { useEffect, useState } from "react";
import { Dialog } from "../Dialog";
import { api, escolherPasta, textoDoErro, type Estado, type Game } from "../api";
import { versaoLegivel } from "../format";
import { som, type ConfigSom } from "../som";

/** Versões guardadas de um jogo, com restauração. O save atual vai antes para a pasta de backups. */
export function Versoes({ jogo, onClose }: { jogo: Game; onClose: () => void }) {
  const [lista, setLista] = useState<string[] | null>(null);
  const [confirmar, setConfirmar] = useState<string | null>(null);
  const [resultado, setResultado] = useState<string | null>(null);
  const [erro, setErro] = useState<string | null>(null);
  const [ocupado, setOcupado] = useState(false);

  useEffect(() => {
    api.versoes(jogo.id).then(setLista).catch((e) => setErro(textoDoErro(e)));
  }, [jogo.id]);

  async function restaurar(v: string) {
    setOcupado(true);
    setErro(null);
    try {
      setResultado(await api.restaurarVersao(jogo.id, v));
      setConfirmar(null);
    } catch (e) {
      setErro(textoDoErro(e));
    } finally {
      setOcupado(false);
    }
  }

  return (
    <Dialog titulo={`Versões de ${jogo.name}`} onClose={onClose} focoKey={lista?.length ?? -1}>
      {lista === null && !erro && <p>Carregando versões…</p>}
      {lista?.length === 0 && <p>Ainda não há versões guardadas. A primeira aparece depois que você jogar com a sincronização ligada.</p>}
      {erro && (
        <p className="erro" role="alert">
          {erro}
        </p>
      )}
      {resultado && (
        <p className="caixa ok" role="status">
          Versão restaurada. O save que estava aqui antes foi guardado em <code className="caminho">{resultado}</code>.
        </p>
      )}
      <ul className="lista">
        {lista?.map((v, i) => {
          const { quando, pc } = versaoLegivel(v);
          return (
            <li key={v} className="item-versao">
              <span>
                <strong>{quando}</strong>
                <small>
                  {pc}
                  {i === 0 && " · mais recente"}
                </small>
              </span>
              {confirmar === v ? (
                <span className="linha">
                  <button className="btn perigo" data-f disabled={ocupado} onClick={() => restaurar(v)}>
                    {ocupado ? "Restaurando…" : "Confirmar"}
                  </button>
                  <button className="btn fraco" data-f onClick={() => setConfirmar(null)}>
                    Cancelar
                  </button>
                </span>
              ) : (
                <button className="btn fraco" data-f onClick={() => setConfirmar(v)}>
                  Restaurar
                </button>
              )}
            </li>
          );
        })}
      </ul>
      {confirmar && <p className="dica">O save atual será guardado em backup antes de restaurar. Feche o jogo antes.</p>}
    </Dialog>
  );
}

/** Conflito: os dois lados mudaram. Quem decide é o usuário; o lado descartado fica guardado. */
export function Conflito({ jogo, onEscolher, onClose }: { jogo: Game; onEscolher: (c: "local" | "nuvem") => void; onClose: () => void }) {
  return (
    <Dialog titulo="Dois saves diferentes" onClose={onClose}>
      <p>
        O save de <strong>{jogo.name}</strong> mudou neste PC e também na pasta de sincronização. Escolha qual manter. O outro fica guardado e pode ser recuperado depois, em “Versões”.
      </p>
      <div className="acoes coluna-movel">
        <button className="btn primario" data-f data-autofocus onClick={() => onEscolher("local")}>
          Usar o save deste PC
        </button>
        <button className="btn" data-f onClick={() => onEscolher("nuvem")}>
          Usar o da pasta de sincronização
        </button>
        <button className="btn fraco" data-f onClick={onClose}>
          Cancelar
        </button>
      </div>
    </Dialog>
  );
}

export function Travado({ jogo, pc, onForcar, onClose }: { jogo: Game; pc: string; onForcar: () => void; onClose: () => void }) {
  return (
    <Dialog titulo="Jogo aberto em outro PC" onClose={onClose}>
      <p>
        <strong>{jogo.name}</strong> parece estar aberto em <strong>{pc}</strong>. Abrir aqui ao mesmo tempo pode criar dois saves diferentes.
      </p>
      <div className="acoes coluna-movel">
        <button className="btn primario" data-f data-autofocus onClick={onClose}>
          Cancelar
        </button>
        <button className="btn" data-f onClick={onForcar}>
          Abrir mesmo assim
        </button>
      </div>
    </Dialog>
  );
}

export function SemSync({ mensagem, onSem, onClose }: { mensagem: string; onSem: () => void; onClose: () => void }) {
  return (
    <Dialog titulo="Não foi possível sincronizar" onClose={onClose}>
      <p role="alert">{mensagem}</p>
      <div className="acoes coluna-movel">
        <button className="btn primario" data-f data-autofocus onClick={onClose}>
          Cancelar
        </button>
        <button className="btn" data-f onClick={onSem}>
          Jogar sem sincronizar
        </button>
      </div>
    </Dialog>
  );
}

export function Remover({ jogo, onConfirmar, onClose }: { jogo: Game; onConfirmar: () => void; onClose: () => void }) {
  return (
    <Dialog titulo="Remover da biblioteca" onClose={onClose}>
      <p>
        Remover <strong>{jogo.name}</strong> da biblioteca? Os saves deste PC e os da pasta de sincronização continuam onde estão. Nada é apagado.
      </p>
      <div className="acoes coluna-movel">
        <button className="btn primario" data-f data-autofocus onClick={onClose}>
          Cancelar
        </button>
        <button className="btn perigo" data-f onClick={onConfirmar}>
          Remover
        </button>
      </div>
    </Dialog>
  );
}

function Volume({ nome, valor, desligado, onMudar }: { nome: string; valor: number; desligado: boolean; onMudar: (v: number) => void }) {
  const pct = Math.round(valor * 100);
  return (
    <div className="linha volume" role="group" aria-label={`Volume ${nome}`}>
      <span>Volume</span>
      <button type="button" className="btn fraco" data-f disabled={desligado || pct <= 0} aria-label={`Diminuir volume ${nome}`} onClick={() => onMudar(Math.max(0, valor - 0.05))}>
        −
      </button>
      <output aria-live="polite">{pct}%</output>
      <button type="button" className="btn fraco" data-f disabled={desligado || pct >= 100} aria-label={`Aumentar volume ${nome}`} onClick={() => onMudar(Math.min(1, valor + 0.05))}>
        +
      </button>
    </div>
  );
}

function SomConfig() {
  const [c, setC] = useState<ConfigSom>(() => som.obter());
  const mudar = (p: Partial<ConfigSom>) => {
    som.definir(p);
    setC(som.obter());
  };
  return (
    <fieldset className="campo">
      <legend>Som</legend>
      <label className="opcao">
        <input type="checkbox" data-f checked={c.efeitos} onChange={(e) => mudar({ efeitos: e.target.checked })} />
        <span>Sons da interface (ao mover, confirmar e voltar)</span>
      </label>
      <Volume nome="dos sons da interface" valor={c.volEfeitos} desligado={!c.efeitos} onMudar={(v) => mudar({ volEfeitos: v })} />
      <label className="opcao">
        <input type="checkbox" data-f checked={c.fundo} onChange={(e) => mudar({ fundo: e.target.checked })} />
        <span>Música ambiente</span>
      </label>
      <Volume nome="da música ambiente" valor={c.volFundo} desligado={!c.fundo} onMudar={(v) => mudar({ volFundo: v })} />
      <small className="dica">O som para sozinho enquanto um jogo está aberto ou a janela está escondida.</small>
    </fieldset>
  );
}

export function Configuracoes({ estado, onClose, onMudou, onSobre }: { estado: Estado; onClose: () => void; onMudou: () => void; onSobre: () => void }) {
  const [nome, setNome] = useState(estado.nome_pc);
  const [msg, setMsg] = useState<string | null>(null);
  const [erro, setErro] = useState<string | null>(null);
  const [ocupado, setOcupado] = useState(false);

  async function rodar(fn: () => Promise<string | void>) {
    setOcupado(true);
    setErro(null);
    setMsg(null);
    try {
      const m = await fn();
      if (m) setMsg(m);
      onMudou();
    } catch (e) {
      setErro(textoDoErro(e));
    } finally {
      setOcupado(false);
    }
  }

  return (
    <Dialog titulo="Configurações" onClose={onClose}>
      <div className="coluna">
        <div className="campo">
          Pasta de sincronização
          <div className="linha">
            <code className="caminho" title={estado.pasta_sync ?? ""}>
              {estado.pasta_sync ?? "Nenhuma"}
            </code>
            <button
              className="btn fraco"
              data-f
              data-autofocus
              disabled={ocupado}
              onClick={() =>
                rodar(async () => {
                  const p = await escolherPasta("Escolha a pasta de sincronização dos saves");
                  if (!p) return;
                  await api.definirPastaSync(p);
                  return "Pasta de sincronização atualizada. Os saves já guardados na pasta antiga não foram movidos.";
                })
              }
            >
              Mudar…
            </button>
          </div>
        </div>
        <label className="campo">
          Nome deste PC
          <div className="linha">
            <input data-f value={nome} onChange={(e) => setNome(e.target.value)} maxLength={40} />
            <button className="btn fraco" data-f disabled={ocupado || nome.trim() === "" || nome === estado.nome_pc} onClick={() => rodar(async () => { await api.definirNomePc(nome); return "Nome atualizado."; })}>
              Salvar
            </button>
          </div>
        </label>
        <div className="campo">
          Catálogo de jogos
          <div className="linha">
            <span>{estado.catalogo_pronto ? "Baixado" : "Ainda não baixado"}</span>
            <button className="btn fraco" data-f disabled={ocupado} onClick={() => rodar(async () => `Catálogo atualizado: ${await api.atualizarCatalogo()} jogos.`)}>
              {ocupado ? "Atualizando…" : estado.catalogo_pronto ? "Atualizar" : "Baixar"}
            </button>
          </div>
        </div>
        <SomConfig />
        {msg && (
          <p className="caixa ok" role="status">
            {msg}
          </p>
        )}
        {erro && (
          <p className="erro" role="alert">
            {erro}
          </p>
        )}
        <div className="acoes">
          <button className="btn fraco" data-f onClick={onSobre}>
            Sobre
          </button>
        </div>
      </div>
    </Dialog>
  );
}

export function Sobre({ onClose }: { onClose: () => void }) {
  const [info, setInfo] = useState<{ nome: string; versao: string } | null>(null);
  useEffect(() => {
    api.sobre().then(setInfo).catch(() => setInfo({ nome: "Sync Game Save", versao: "desconhecida" }));
  }, []);
  return (
    <Dialog titulo="Sobre" onClose={onClose}>
      <p>
        <strong>{info?.nome ?? "Sync Game Save"}</strong>
        <br />
        Versão {info?.versao ?? "…"}
      </p>
      <p>Guarda os saves dos seus jogos numa pasta de nuvem, sem perder o progresso.</p>
      <p className="dica">
        O catálogo de caminhos de save vem do manifesto do projeto Ludusavi (licença MIT), de Matthew T. Kennerly.
      </p>
      <div className="acoes">
        <button className="btn primario" data-f data-autofocus onClick={onClose}>
          Fechar
        </button>
      </div>
    </Dialog>
  );
}
