import { useState } from "react";
import { Dialog } from "../Dialog";
import { api, escolherArquivoExe, escolherPasta, textoDoErro, type Reconhecimento, type SaveCandidate } from "../api";

type Props = { catalogoPronto: boolean; onClose: () => void; onAdicionado: () => void };

type Item = SaveCandidate & { marcado: boolean };

const certeza = {
  High: "Reconheci este jogo. Confira os dados e adicione.",
  Low: "Encontrei um jogo parecido, mas não tenho certeza. Confira o nome e o save.",
  None: "Não conheço este jogo. Indique o executável e a pasta onde ele guarda o save.",
} as const;

/** Escolhe a pasta do jogo, mostra o que foi reconhecido e adiciona à biblioteca. */
export function AddGame({ catalogoPronto, onClose, onAdicionado }: Props) {
  const [pastaJogo, setPastaJogo] = useState<string | null>(null);
  const [rec, setRec] = useState<Reconhecimento | null>(null);
  const [nome, setNome] = useState("");
  const [exe, setExe] = useState<string | null>(null);
  const [itens, setItens] = useState<Item[]>([]);
  const [ocupado, setOcupado] = useState(false);
  const [erro, setErro] = useState<string | null>(null);
  const [catalogo, setCatalogo] = useState(catalogoPronto);

  async function escolherJogo() {
    setErro(null);
    const p = await escolherPasta("Escolha a pasta onde o jogo está instalado");
    if (!p) return;
    setOcupado(true);
    try {
      const r = await api.reconhecerPasta(p);
      setPastaJogo(p);
      setRec(r);
      setNome(r.name);
      setExe(r.exe);
      setItens(r.save_candidates.map((c) => ({ ...c, marcado: c.selected })));
    } catch (e) {
      setErro(textoDoErro(e));
    } finally {
      setOcupado(false);
    }
  }

  async function escolherExecutavel() {
    setErro(null);
    const escolhido = await escolherArquivoExe("Escolha o executável do jogo");
    if (!escolhido) return;
    const separador = Math.max(escolhido.lastIndexOf("/"), escolhido.lastIndexOf("\\"));
    if (separador < 0) {
      setErro("Não consegui identificar a pasta deste executável. Escolha a pasta do jogo.");
      return;
    }
    const pasta = escolhido.slice(0, separador);
    setOcupado(true);
    try {
      const r = await api.reconhecerPasta(pasta);
      setPastaJogo(pasta);
      setRec(r);
      setNome(r.name);
      setExe(escolhido);
      setItens(r.save_candidates.map((c) => ({ ...c, marcado: c.selected })));
    } catch (e) {
      setErro(textoDoErro(e));
    } finally {
      setOcupado(false);
    }
  }

  async function baixarCatalogo() {
    setOcupado(true);
    setErro(null);
    try {
      await api.atualizarCatalogo();
      setCatalogo(true);
      if (pastaJogo) await reconhecerDeNovo(pastaJogo);
    } catch (e) {
      setErro(textoDoErro(e));
    } finally {
      setOcupado(false);
    }
  }

  async function reconhecerDeNovo(p: string) {
    const r = await api.reconhecerPasta(p);
    setRec(r);
    setNome(r.name);
    setExe(r.exe ?? exe);
    setItens(r.save_candidates.map((c) => ({ ...c, marcado: c.selected })));
  }

  async function outraPasta() {
    const p = await escolherPasta("Escolha a pasta onde o jogo guarda o save");
    if (!p) return;
    const logical = await api.converterCaminho(p);
    setItens((atual) => (atual.some((i) => i.logical === logical) ? atual : [...atual, { logical, resolved: p, exists: true, files: 0, selected: true, marcado: true }]));
  }

  async function trocarExe() {
    const e = await escolherArquivoExe("Escolha o executável do jogo");
    if (e) setExe(e);
  }

  async function adicionar() {
    if (!exe) return;
    setOcupado(true);
    setErro(null);
    try {
      await api.adicionarJogo({
        name: nome.trim(),
        exe,
        install_dir: pastaJogo,
        catalog_id: rec?.catalog_id ?? null,
        save_paths: itens.filter((i) => i.marcado).map((i) => i.logical),
      });
      onAdicionado();
    } catch (e) {
      setErro(textoDoErro(e));
      setOcupado(false);
    }
  }

  const semSave = itens.every((i) => !i.marcado);

  return (
    <Dialog titulo="Adicionar jogo" onClose={onClose} largo focoKey={rec ? "form" : "inicio"}>
      {!rec ? (
        <div className="coluna">
          <p>Escolha a pasta onde o jogo está instalado. O app tenta descobrir o nome, o executável e onde ele guarda o save.</p>
          <button className="btn primario" data-f data-autofocus onClick={escolherJogo} disabled={ocupado}>
            {ocupado ? "Procurando…" : "Escolher pasta do jogo…"}
          </button>
          <button className="btn fraco" data-f onClick={escolherExecutavel} disabled={ocupado}>
            Escolher executável…
          </button>
        </div>
      ) : (
        <form
          className="coluna"
          onSubmit={(e) => {
            e.preventDefault();
            void adicionar();
          }}
        >
          <p className={`aviso-certeza ${rec.confidence.toLowerCase()}`}>{certeza[rec.confidence]}</p>
          {!catalogo && (
            <p className="dica">
              O catálogo de jogos ainda não foi baixado, por isso não sei onde este jogo guarda o save.{" "}
              <button type="button" className="btn fraco" data-f onClick={baixarCatalogo} disabled={ocupado}>
                Baixar catálogo
              </button>
            </p>
          )}
          <label className="campo">
            Nome do jogo
            <input data-f data-autofocus value={nome} onChange={(e) => setNome(e.target.value)} maxLength={120} />
          </label>
          <div className="campo">
            Executável
            <div className="linha">
              <code className="caminho" title={exe ?? ""}>
                {exe ?? "Nenhum executável encontrado"}
              </code>
              <button type="button" className="btn fraco" data-f onClick={trocarExe}>
                Trocar…
              </button>
            </div>
          </div>
          <fieldset className="campo">
            <legend>Pastas de save</legend>
            {itens.length === 0 && <p className="dica">Nenhuma pasta de save conhecida. Use “Adicionar outra pasta de save”.</p>}
            {itens.map((i) => (
              <label key={i.logical} className="opcao">
                <input type="checkbox" data-f checked={i.marcado} onChange={(e) => setItens((l) => l.map((x) => (x.logical === i.logical ? { ...x, marcado: e.target.checked } : x)))} />
                <span>
                  <code className="caminho">{i.logical}</code>
                  <small>{i.exists ? (i.files > 0 ? `Encontrada: ${i.files} arquivo(s)` : "Encontrada") : "Ainda não existe neste PC"}</small>
                </span>
              </label>
            ))}
            <button type="button" className="btn fraco" data-f onClick={outraPasta}>
              Adicionar outra pasta de save…
            </button>
          </fieldset>
          {semSave && <p className="dica">Sem uma pasta de save marcada, o jogo abre, mas o save não é sincronizado. Você pode indicar a pasta depois.</p>}
          {erro && (
            <p className="erro" role="alert">
              {erro}
            </p>
          )}
          <div className="acoes">
            <button type="button" className="btn fraco" data-f onClick={onClose}>
              Cancelar
            </button>
            <button type="submit" className="btn primario" data-f disabled={ocupado || nome.trim() === "" || !exe}>
              {ocupado ? "Adicionando…" : "Adicionar jogo"}
            </button>
          </div>
        </form>
      )}
      {!rec && erro && (
        <p className="erro" role="alert">
          {erro}
        </p>
      )}
    </Dialog>
  );
}
