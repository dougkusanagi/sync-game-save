import { useEffect, useState, type FormEvent } from "react";
import { Dialog } from "../Dialog";
import { api, escolherArquivoExe, escolherPasta, instalarAtualizacao, textoDoErro, verificarAtualizacao, type Atualizacao, type Estado, type Game, type Temas } from "../api";
import { versaoLegivel } from "../format";
import { som, type ConfigSom } from "../som";
import { acharTema, aplicarTema } from "../temas";

export function PararJogo({ jogo, onClose }: { jogo: Game; onClose: () => void }) {
  const [estado, setEstado] = useState<"esperando" | "forcar" | "fechado" | "ausente" | "erro" | "forcando">("esperando");
  const [erro, setErro] = useState("");
  useEffect(() => {
    let ativo = true;
    api.pararJogo(jogo.id).then((resta) => {
      if (ativo) setEstado(resta === null ? "ausente" : resta ? "forcar" : "fechado");
    }).catch((e) => { if (ativo) { setErro(textoDoErro(e)); setEstado("erro"); } });
    return () => { ativo = false; };
  }, [jogo.id]);
  async function forcar() {
    setEstado("forcando");
    try { await api.forcarParadaJogo(jogo.id); setEstado("fechado"); }
    catch (e) { setErro(textoDoErro(e)); setEstado("erro"); }
  }
  return (
    <Dialog titulo={`Parar ${jogo.name}`} onClose={onClose} focoKey={estado}>
      {estado === "esperando" && <p role="status">Pedindo para o jogo fechar. Aguarde enquanto ele salva e encerra.</p>}
      {estado === "forcar" && <><p>O jogo ainda está aberto. Forçar o fechamento pode corromper ou perder o save. Se possível, salve e feche o jogo normalmente.</p><div className="acoes coluna-movel parada-acoes"><button className="btn fraco" data-f data-autofocus onClick={onClose}>Continuar aguardando</button><button className="btn perigo" data-f onClick={() => void forcar()}>Forçar fechamento</button></div></>}
      {estado === "forcando" && <p role="status">Fechando o jogo. A sincronização começa depois que todos os processos terminarem.</p>}
      {estado === "fechado" && <><p role="status">O pedido foi concluído. A sincronização começa depois que o jogo terminar de fechar.</p><button className="btn" data-f onClick={onClose}>Entendi</button></>}
      {estado === "ausente" && <><p role="status">O jogo não está sendo acompanhado. Se ele ainda estiver aberto, feche-o pelo próprio jogo.</p><button className="btn" data-f onClick={onClose}>Entendi</button></>}
      {estado === "erro" && <><p role="alert">{erro}</p><button className="btn" data-f onClick={onClose}>Fechar</button></>}
    </Dialog>
  );
}

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

/** Ações menos frequentes de um jogo, abertas por "Mais opções" ou pelo botão Opções. */
export function Opcoes({
  jogo,
  onClose,
  onJogar,
  onVersoes,
  onConquistas,
  onRemover,
  onEditar,
}: {
  jogo: Game;
  onClose: () => void;
  onJogar: () => void;
  onVersoes: () => void;
  onConquistas: () => void;
  onRemover: () => void;
  onEditar: () => void;
}) {
  return (
    <Dialog titulo={jogo.name} onClose={onClose}>
      <div className="coluna opcoes-jogo">
        <button className="btn primario" data-f data-autofocus onClick={onJogar}>
          Jogar
        </button>
        <button className="btn fraco" data-f onClick={onVersoes}>
          Versões do save
        </button>
        <button className="btn fraco" data-f onClick={onConquistas}>
          Conquistas
        </button>
        <button className="btn fraco" data-f onClick={onEditar}>
          Editar jogo
        </button>
        <button className="btn fraco" data-f onClick={onRemover}>
          Remover da biblioteca
        </button>
      </div>
    </Dialog>
  );
}

/** Edita os dados da biblioteca sem mover nem alterar arquivos de save. */
export function EditarJogo({ jogo, onClose, onSaved }: { jogo: Game; onClose: () => void; onSaved: () => void }) {
  const [nome, setNome] = useState(jogo.name);
  const [exe, setExe] = useState(jogo.exe);
  const [paths, setPaths] = useState(jogo.save_paths);
  const [ocupado, setOcupado] = useState(false);
  const [erro, setErro] = useState<string | null>(null);

  async function adicionarPasta() {
    const p = await escolherPasta("Escolha uma pasta de save");
    if (!p) return;
    const logical = await api.converterCaminho(p);
    setPaths((atual) => atual.includes(logical) ? atual : [...atual, logical]);
  }

  async function salvar(e: FormEvent) {
    e.preventDefault();
    setOcupado(true);
    setErro(null);
    try {
      await api.editarJogo(jogo.id, { name: nome.trim(), exe, install_dir: jogo.install_dir, catalog_id: jogo.catalog_id, save_paths: paths });
      onSaved();
    } catch (err) {
      setErro(textoDoErro(err));
      setOcupado(false);
    }
  }

  return <Dialog titulo={`Editar ${jogo.name}`} onClose={onClose}>
    <form className="coluna" onSubmit={(e) => void salvar(e)}>
      <label className="campo">Nome do jogo<input data-f data-autofocus value={nome} maxLength={120} onChange={(e) => setNome(e.target.value)} /></label>
      <div className="campo">Executável<div className="linha"><code className="caminho">{exe}</code><button type="button" className="btn fraco" data-f onClick={async () => { const p = await escolherArquivoExe("Escolha o executável do jogo"); if (p) setExe(p); }}>Trocar…</button></div></div>
      <fieldset className="campo"><legend>Pastas de save</legend>
        {paths.map((p) => <label className="opcao" key={p}><input type="checkbox" data-f checked onChange={() => setPaths((atual) => atual.filter((x) => x !== p))} /><code className="caminho">{p}</code></label>)}
        {paths.length === 0 && <p className="dica">Sem caminhos de save; o jogo não será sincronizado.</p>}
        <button type="button" className="btn fraco" data-f onClick={() => void adicionarPasta()}>Adicionar pasta de save…</button>
      </fieldset>
      {erro && <p className="erro" role="alert">{erro}</p>}
      <div className="acoes"><button type="button" className="btn fraco" data-f onClick={onClose}>Cancelar</button><button className="btn primario" data-f disabled={ocupado || !nome.trim() || !exe}>{ocupado ? "Salvando…" : "Salvar alterações"}</button></div>
    </form>
  </Dialog>;
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

function SteamConfig({
  estado,
  rodar,
  ocupado,
}: {
  estado: Estado;
  rodar: (fn: () => Promise<string | void>) => Promise<void>;
  ocupado: boolean;
}) {
  const [steamId, setSteamId] = useState("");
  const valido = /^\d{17}$/.test(steamId.trim());

  return (
    <fieldset className="campo">
      <legend>Conquistas da Steam</legend>
      <p className="dica">
        Não precisa configurar nada: o app usa a conta da Steam deste PC. Para o progresso aparecer, o seu perfil precisa mostrar os detalhes dos
        jogos como públicos (na Steam: Perfil › Editar perfil › Privacidade).
      </p>
      <p>{estado.steam_id ? `Conta da Steam encontrada (SteamID ${estado.steam_id}).` : "Nenhuma conta da Steam encontrada neste PC. Entre na Steam neste PC e abra o app de novo."}</p>
      <details>
        <summary data-f tabIndex={0}>
          Usar outra conta
        </summary>
        <label className="campo">
          SteamID64
          <input
            data-f
            type="text"
            inputMode="numeric"
            autoComplete="off"
            maxLength={17}
            value={steamId}
            placeholder="Ex.: 76561198000000000"
            onChange={(e) => setSteamId(e.target.value.replace(/\D/g, ""))}
          />
        </label>
        {steamId !== "" && !valido && <p className="erro">O SteamID64 tem 17 números.</p>}
        <div className="linha">
          <button
            className="btn fraco"
            data-f
            disabled={!valido || ocupado}
            onClick={() =>
              rodar(async () => {
                await api.definirCredenciaisSteam("", steamId.trim());
                setSteamId("");
                return "Conta da Steam trocada neste PC.";
              })
            }
          >
            Usar esta conta
          </button>
          <button
            className="btn fraco"
            data-f
            disabled={ocupado}
            onClick={() =>
              rodar(async () => {
                await api.definirCredenciaisSteam("", "");
                return "O app voltou a usar a conta da Steam deste PC.";
              })
            }
          >
            Usar a conta deste PC
          </button>
        </div>
      </details>
    </fieldset>
  );
}

function TemaConfig() {
  const [lista, setLista] = useState<Temas | null>(null);
  const [erro, setErro] = useState<string | null>(null);

  useEffect(() => {
    api
      .temas()
      .then((t) => {
        setLista(t);
        aplicarTema(acharTema(t));
      })
      .catch((e) => setErro(textoDoErro(e)));
  }, []);

  async function trocar(id: string) {
    if (!lista || id === lista.atual) return;
    const anterior = acharTema(lista);
    const novo = lista.temas.find((t) => t.id === id) ?? null;
    aplicarTema(novo);
    setLista({ ...lista, atual: id });
    setErro(null);
    try {
      const atualizada = await api.definirTema(id);
      setLista(atualizada);
      aplicarTema(acharTema(atualizada));
    } catch (e) {
      aplicarTema(anterior);
      setLista(lista);
      setErro(textoDoErro(e));
    }
  }

  const atual = acharTema(lista);
  return (
    <fieldset className="campo">
      <legend>Tema</legend>
      <label className="campo">
        Aparência
        <select data-f value={lista?.atual ?? ""} disabled={!lista} onChange={(e) => void trocar(e.target.value)}>
          {!lista && <option value="">Carregando…</option>}
          {lista?.temas.map((t) => (
            <option key={t.id} value={t.id}>
              {t.name}
              {t.builtin ? "" : " (instalado)"}
            </option>
          ))}
        </select>
      </label>
      {atual && (
        <small className="dica">
          {atual.author} · versão {atual.version}
        </small>
      )}
      <small className="dica">Os temas trocam cores e medidas. Nenhum código do tema é executado.</small>
      {erro && (
        <p className="erro" role="alert">
          {erro}
        </p>
      )}
    </fieldset>
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
        <SteamConfig estado={estado} rodar={rodar} ocupado={ocupado} />
        <TemaConfig />
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
  const [info, setInfo] = useState<{ nome: string; versao: string; build: string } | null>(null);
  const [atualizacao, setAtualizacao] = useState<Atualizacao | null>(null);
  const [verificando, setVerificando] = useState(true);
  const [instalando, setInstalando] = useState(false);
  const [erro, setErro] = useState<string | null>(null);

  useEffect(() => {
    api.sobre()
      .then(setInfo)
      .catch(() => setInfo({ nome: "Sync Game Save", versao: "desconhecida", build: "" }));
  }, []);

  async function verificar() {
    setVerificando(true);
    setErro(null);
    try {
      setAtualizacao(await verificarAtualizacao());
    } catch (e) {
      console.error(e);
      setErro("Não foi possível verificar atualizações agora. Confira sua conexão e tente de novo.");
    } finally {
      setVerificando(false);
    }
  }

  useEffect(() => {
    void verificar();
  }, []);

  async function instalar() {
    setInstalando(true);
    setErro(null);
    try {
      await instalarAtualizacao();
    } catch (e) {
      console.error(e);
      setErro("Não foi possível instalar a atualização. Confira sua conexão e tente de novo.");
      setInstalando(false);
    }
  }

  return (
    <Dialog titulo="Sobre" onClose={onClose}>
      <p>
        <strong>{info?.nome ?? "Sync Game Save"}</strong>
        <br />
        Versão {info?.versao ?? "…"}
        {info?.build ? ` · build ${info.build}` : ""}
      </p>
      <p>Guarda os saves dos seus jogos numa pasta de nuvem, sem perder o progresso.</p>
      <div className="campo">
        Atualizações
        {atualizacao ? (
          <>
            <p>
              A versão {atualizacao.versao} está disponível. O app baixa, instala e reabre.
            </p>
            <button className="btn primario" data-f disabled={instalando} onClick={() => void instalar()}>
              {instalando ? "Baixando e instalando…" : `Atualizar para ${atualizacao.versao}`}
            </button>
          </>
        ) : (
          <div className="linha">
            <span>{verificando ? "Procurando atualização…" : "Você está na versão mais recente."}</span>
            <button className="btn fraco" data-f disabled={verificando || instalando} onClick={() => void verificar()}>
              Verificar de novo
            </button>
          </div>
        )}
        {erro && (
          <p className="erro" role="alert">
            {erro}
          </p>
        )}
      </div>
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
