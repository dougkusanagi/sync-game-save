import { useEffect, useState } from "react";
import { Dialog } from "../Dialog";
import { api, textoDoErro, type Game } from "../api";
import { haQuanto } from "../format";
import { SomConfig } from "./Dialogos";
import type { EstadoCard } from "./Biblioteca";

type Props = { jogos: Game[]; estados: Record<number, EstadoCard>; onClose: () => void; onMudou: () => void };

const ROTULO_ESTADO: Record<string, string> = { jogando: "Jogando agora", sincronizando: "Sincronizando", salvando: "Enviando o save" };

const RESULTADO: Record<string, string> = {
  created: "Save enviado.",
  already_synced: "Já estava em dia.",
  nothing_to_backup: "Sem arquivos de save neste PC.",
  needs_decision: "A pasta de sincronização tem outra versão. Abra o jogo para escolher qual manter.",
  in_use: "Jogo aberto ou sincronizando. Tente depois que fechar.",
};

/** Central rápida: estado real de cada jogo, envio dos saves deste PC, volume e saída do app. */
export function Central({ jogos, estados, onClose, onMudou }: Props) {
  const [ocupado, setOcupado] = useState(false);
  const [resultados, setResultados] = useState<Record<number, string>>({});
  const [erro, setErro] = useState<string | null>(null);
  const [avisos, setAvisos] = useState<{ quando: number; texto: string }[]>([]);
  const [aviso, setAviso] = useState<string | null>(null);
  useEffect(() => { api.avisos().then(setAvisos).catch(() => setAvisos([])); }, []);
  const jogando = Object.values(estados).some(Boolean);
  const comSave = jogos.filter((j) => j.save_paths.length > 0);

  async function sincronizarTudo() {
    setOcupado(true);
    setResultados({});
    for (const j of comSave) {
      if (estados[j.id]) { setResultados((r) => ({ ...r, [j.id]: RESULTADO.in_use })); continue; }
      setResultados((r) => ({ ...r, [j.id]: "Enviando…" }));
      try {
        const r = await api.primeiroBackup(j.id);
        setResultados((m) => ({ ...m, [j.id]: r.kind === "locked" ? `${r.pc} está jogando agora. Tente depois que fechar.` : RESULTADO[r.kind] }));
      } catch (e) { setResultados((m) => ({ ...m, [j.id]: textoDoErro(e) })); }
    }
    setOcupado(false);
    onMudou();
  }

  async function voltar(id: number) {
    setAviso(null);
    try {
      const ok = await api.voltarAoJogo(id);
      if (ok) onClose();
      else setAviso("Não achei a janela do jogo. Se ele está em tela cheia ou minimizado, use Alt+Tab.");
    } catch (e) { setAviso(textoDoErro(e)); }
  }

  async function sair() {
    setErro(null);
    try { await api.sairDoApp(); } catch (e) { setErro(textoDoErro(e)); }
  }

  return <Dialog titulo="Central rápida" onClose={onClose} largo>
    <div className="coluna">
      <section className="coluna" aria-label="Estado dos jogos">
        <h3>Jogos</h3>
        {comSave.length === 0 && <p>Nenhum jogo com caminho de save ainda.</p>}
        <ul className="lista-central">
          {comSave.map((j) => <li key={j.id}>
            <strong>{j.name}</strong>
            <small>{estados[j.id] ? ROTULO_ESTADO[estados[j.id] as string] ?? "Ocupado" : j.last_sync_at === null ? "Ainda não sincronizado" : `Sincronizado ${haQuanto(j.last_sync_at)}`}</small>
            {estados[j.id] === "jogando" && <button className="btn fraco" data-f onClick={() => void voltar(j.id)}>Voltar ao jogo</button>}
            {resultados[j.id] && <small role="status">{resultados[j.id]}</small>}
          </li>)}
        </ul>
        <button className="btn primario" data-f data-autofocus disabled={ocupado || comSave.length === 0} onClick={() => void sincronizarTudo()}>{ocupado ? "Sincronizando…" : "Sincronizar tudo agora"}</button>
        <small>Envia para a pasta de sincronização os saves que mudaram neste PC. Versões de outros PCs são recebidas ao abrir cada jogo. Nunca troca um save sozinho: se os dois lados mudaram, a escolha é sua.</small>
      </section>
      {aviso && <p className="erro" role="status">{aviso}</p>}
      <section className="coluna" aria-label="Avisos recentes">
        <h3>Avisos recentes</h3>
        {avisos.length === 0 ? <p>Nenhum aviso desde que o app abriu.</p> : <ul className="lista-central">
          {avisos.map((a, i) => <li key={i}><small>{haQuanto(a.quando)}</small><span>{a.texto}</span></li>)}
        </ul>}
      </section>
      <SomConfig />
      <section className="coluna">
        <h3>Aplicativo</h3>
        <button className="btn fraco" data-f disabled={jogando || ocupado} onClick={() => void sair()}>Sair do app</button>
        {jogando && <small>Feche o jogo e espere o save ser enviado antes de sair.</small>}
        {erro && <p className="erro" role="alert">{erro}</p>}
      </section>
    </div>
  </Dialog>;
}
