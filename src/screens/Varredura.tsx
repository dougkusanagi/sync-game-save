import { useState } from "react";
import { Dialog } from "../Dialog";
import { api, escolherArquivoExe, escolherPasta, textoDoErro, type Reconhecimento } from "../api";

type Item = { pasta: string; rec: Reconhecimento; candidatos: { id: number; name: string }[]; grupo: string; marcado: boolean; saves: string[]; adicionado: boolean; erro?: string };
type Props = { onClose: () => void; onVoltar: () => void; onAdicionado: () => void };

export function Varredura({ onClose, onVoltar, onAdicionado }: Props) {
  const [itens, setItens] = useState<Item[] | null>(null);
  const [erros, setErros] = useState<[string, string][]>([]);
  const [ocupado, setOcupado] = useState(false);
  const [erro, setErro] = useState<string | null>(null);
  const [profundidade, setProfundidade] = useState(2);
  const [houveAdicao, setHouveAdicao] = useState(false);
  const fechar = () => ocupado ? undefined : houveAdicao ? onAdicionado() : onClose();
  const mudar = (pasta: string, fn: (item: Item) => Item) => setItens((atual) => atual?.map((i) => i.pasta === pasta ? fn(i) : i) ?? null);

  async function procurar() {
    setErro(null);
    try {
      const pasta = await escolherPasta("Escolha a pasta que contém seus jogos");
      if (!pasta) return;
      setOcupado(true);
      const resultado = await api.varrerPasta(pasta, profundidade);
      const grupo = (lista: [string, Reconhecimento][], nome: string): Item[] => lista.map(([p, rec]) => ({ pasta: p, rec, candidatos: [...(rec.catalog_id === null ? [] : [{ id: rec.catalog_id, name: rec.name }]), ...rec.alternatives], grupo: nome, marcado: false, saves: rec.save_candidates.filter((c) => c.selected).map((c) => c.logical), adicionado: false }));
      setItens([...grupo(resultado.recognized, "Reconhecidos"), ...grupo(resultado.needs_confirmation, "Precisam de confirmação"), ...grupo(resultado.unknown, "Não reconhecidos")]);
      setErros(resultado.errors);
    } catch (e) { setErro(textoDoErro(e)); } finally { setOcupado(false); }
  }

  async function adicionarPasta(item: Item) {
    try {
      const pasta = await escolherPasta("Escolha a pasta onde o jogo guarda o save");
      if (!pasta) return;
      const logical = await api.converterCaminho(pasta);
      mudar(item.pasta, (i) => ({ ...i, saves: i.saves.includes(logical) ? i.saves : [...i.saves, logical], rec: { ...i.rec, save_candidates: i.rec.save_candidates.some((c) => c.logical === logical) ? i.rec.save_candidates : [...i.rec.save_candidates, { logical, resolved: pasta, exists: true, files: 0, selected: true }] } }));
    } catch (e) { mudar(item.pasta, (i) => ({ ...i, erro: textoDoErro(e) })); }
  }

  async function trocarExecutavel(item: Item) {
    try {
      const exe = await escolherArquivoExe("Escolha o executável do jogo");
      if (exe) mudar(item.pasta, (i) => ({ ...i, rec: { ...i.rec, exe } }));
    } catch (e) { mudar(item.pasta, (i) => ({ ...i, erro: textoDoErro(e) })); }
  }

  async function escolherCandidato(item: Item, id: number) {
    setOcupado(true);
    try {
      const rec = await api.escolherCandidatoCatalogo(item.pasta, id);
      mudar(item.pasta, (i) => ({ ...i, rec, saves: rec.save_candidates.filter((c) => c.selected).map((c) => c.logical), marcado: false, erro: undefined }));
    } catch (e) { mudar(item.pasta, (i) => ({ ...i, erro: textoDoErro(e) })); }
    finally { setOcupado(false); }
  }

  async function adicionar() {
    if (!itens || ocupado) return;
    setOcupado(true);
    for (const item of itens.filter((i) => i.marcado && !i.adicionado)) {
      if (!item.rec.exe || !item.saves.length || !item.rec.name.trim()) continue;
      try {
        await api.adicionarJogo({ name: item.rec.name.trim(), exe: item.rec.exe, install_dir: item.pasta, catalog_id: item.rec.catalog_id, save_paths: item.saves });
        mudar(item.pasta, (i) => ({ ...i, adicionado: true, marcado: false, erro: undefined }));
        setHouveAdicao(true);
      } catch (e) { mudar(item.pasta, (i) => ({ ...i, erro: textoDoErro(e) })); }
    }
    setOcupado(false);
  }

  return <Dialog titulo="Procurar jogos" onClose={fechar} largo focoKey={itens === null ? "inicio" : "resultado"}>
    <div className="coluna varredura">
      {!itens ? <>
        <p>Escolha uma pasta que contém jogos instalados. Confira cada resultado antes de adicionar.</p>
        <label className="campo">Níveis de pastas a procurar <select data-f value={profundidade} onChange={(e) => setProfundidade(Number(e.target.value))}><option value={1}>1</option><option value={2}>2</option><option value={3}>3</option></select></label>
        <button className="btn primario" data-f data-autofocus disabled={ocupado} onClick={procurar}>{ocupado ? "Procurando…" : "Escolher pasta e procurar…"}</button>
        <button className="btn fraco" data-f disabled={ocupado} onClick={onVoltar}>Voltar</button>
      </> : <>
        <p>Marque os jogos que quer adicionar e confira os caminhos de save. Jogos sem executável ou caminho de save não podem ser adicionados aqui. A primeira sincronização será feita ao abrir cada jogo.</p>
        {itens.length === 0 && <p>Nenhum jogo encontrado nesta pasta.</p>}
        {["Reconhecidos", "Precisam de confirmação", "Não reconhecidos"].map((grupo) => itens.some((i) => i.grupo === grupo) && <section key={grupo} className="coluna"><h3>{grupo}</h3>{itens.filter((i) => i.grupo === grupo).map((i) => <fieldset className="campo" key={i.pasta} disabled={ocupado || i.adicionado}>
          <legend>{i.rec.name}</legend>
          {i.candidatos.length > 1 && <div className="campo"><span>Outros jogos possíveis</span><small>Ao trocar, o nome, o executável e os caminhos de save serão substituídos pelas sugestões do catálogo.</small>{i.candidatos.filter((c) => c.id !== i.rec.catalog_id).map((c) => <button type="button" className="btn fraco" data-f key={c.id} onClick={() => void escolherCandidato(i, c.id)}>Usar {c.name}</button>)}</div>}
          <label className="campo">Nome do jogo<input data-f value={i.rec.name} maxLength={120} onChange={(e) => { const nome = e.target.value; mudar(i.pasta, (x) => ({ ...x, rec: { ...x.rec, name: nome }, marcado: nome.trim() ? x.marcado : false })); }} /></label>
          <label className="opcao"><input type="checkbox" data-f checked={i.marcado} disabled={!i.rec.exe || i.saves.length === 0 || !i.rec.name.trim()} onChange={(e) => { const marcado = e.target.checked; mudar(i.pasta, (x) => ({ ...x, marcado })); }} /> Adicionar {i.rec.name}</label>
          <small><code className="caminho">{i.pasta}</code></small>
          <small>Executável: {i.rec.exe ?? "não encontrado"}</small><button type="button" className="btn fraco" data-f onClick={() => void trocarExecutavel(i)}>Trocar executável…</button>
          {i.rec.save_candidates.length === 0 && <small>Nenhum caminho de save encontrado. Adicione a pasta de save abaixo.</small>}
          {i.rec.save_candidates.map((c) => <label className="opcao" key={c.logical}><input type="checkbox" data-f checked={i.saves.includes(c.logical)} onChange={(e) => { const marcado = e.target.checked; mudar(i.pasta, (x) => { const saves = marcado ? [...x.saves, c.logical] : x.saves.filter((s) => s !== c.logical); return { ...x, saves, marcado: saves.length ? x.marcado : false }; }); }} /><span><code className="caminho">{c.logical}</code><small>{c.exists ? `Encontrado: ${c.files} arquivo(s)` : "Ainda não existe neste PC"}</small></span></label>)}
          <button type="button" className="btn fraco" data-f onClick={() => void adicionarPasta(i)}>Adicionar pasta de save…</button>
          {i.adicionado && <p>Adicionado à biblioteca.</p>}{i.erro && <p className="erro" role="alert">{i.erro}</p>}
        </fieldset>)}</section>)}
        {erros.length > 0 && <section><h3>Pastas que não puderam ser examinadas</h3><ul>{erros.map(([p, e], idx) => <li key={idx}><code className="caminho">{p}</code>: {e}</li>)}</ul></section>}
        <div className="acoes"><button className="btn fraco" data-f disabled={ocupado} onClick={fechar}>Concluir</button><button className="btn primario" data-f disabled={ocupado || !itens.some((i) => i.marcado && !i.adicionado)} onClick={adicionar}>{ocupado ? "Adicionando…" : "Adicionar selecionados"}</button></div>
      </>}
      {erro && <p className="erro" role="alert">{erro}</p>}
    </div>
  </Dialog>;
}
