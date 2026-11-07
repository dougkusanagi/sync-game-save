import { useState } from "react";
import { api, escolherPasta, textoDoErro, type VerificacaoPasta } from "../api";

type Props = { nomePc: string; onPronto: () => void };

/** Primeiro uso: explica a pasta na nuvem, valida a pasta escolhida e registra este PC. */
export function Wizard({ nomePc, onPronto }: Props) {
  const [passo, setPasso] = useState<1 | 2 | 3>(1);
  const [pasta, setPasta] = useState<string | null>(null);
  const [check, setCheck] = useState<VerificacaoPasta | null>(null);
  const [nome, setNome] = useState(nomePc);
  const [ocupado, setOcupado] = useState(false);
  const [erro, setErro] = useState<string | null>(null);

  async function escolher() {
    setErro(null);
    const p = await escolherPasta("Escolha a pasta de sincronização dos saves");
    if (!p) return;
    setOcupado(true);
    try {
      const c = await api.verificarPastaSync(p);
      setPasta(p);
      setCheck(c);
      if (c.message) setErro(c.message);
    } catch (e) {
      setPasta(null);
      setCheck(null);
      setErro(textoDoErro(e));
    } finally {
      setOcupado(false);
    }
  }

  async function concluir() {
    if (!pasta) return;
    setOcupado(true);
    setErro(null);
    try {
      await api.definirPastaSync(pasta);
      await api.definirNomePc(nome);
      onPronto();
    } catch (e) {
      setErro(textoDoErro(e));
      setOcupado(false);
    }
  }

  return (
    <main className="wizard" aria-labelledby="wiz-titulo">
      <p className="passo">Passo {passo} de 3</p>
      {passo === 1 && (
        <section>
          <h1 id="wiz-titulo">Vamos guardar seus saves com segurança</h1>
          <p>
            O Sync Game Save copia o progresso dos seus jogos para uma pasta que o OneDrive, o Google Drive, o Dropbox ou o Syncthing sincroniza. Assim, se este PC quebrar ou for formatado, seus saves continuam salvos e
            podem ir para outros PCs.
          </p>
          <p>Antes de qualquer cópia que troque um save, o app guarda o save anterior. Nada é apagado sem backup.</p>
          <p className="dica">Dica: no OneDrive, marque a pasta como “Sempre manter neste dispositivo”.</p>
          <button className="btn primario" data-f data-autofocus onClick={() => setPasso(2)}>
            Escolher a pasta
          </button>
        </section>
      )}
      {passo === 2 && (
        <section>
          <h1 id="wiz-titulo">Qual pasta guarda os saves?</h1>
          <p>Escolha uma pasta que já esteja dentro do seu serviço de nuvem. Se você já usa o Sync Game Save em outro PC, escolha a mesma pasta.</p>
          <button className="btn primario" data-f data-autofocus onClick={escolher} disabled={ocupado}>
            {ocupado ? "Verificando…" : pasta ? "Escolher outra pasta" : "Escolher pasta…"}
          </button>
          {pasta && check && !check.message && (
            <div className="caixa ok" role="status">
              <strong>{pasta}</strong>
              {check.has_data ? (
                <p>
                  Encontrei dados do Sync Game Save aqui{check.games.length > 0 && `: ${check.games.join(", ")}`}. Nada será sobrescrito; os jogos serão ligados quando você adicioná-los neste PC.
                </p>
              ) : (
                <p>Pasta pronta. Consegui gravar e ler nela.</p>
              )}
            </div>
          )}
          {erro && (
            <p className="erro" role="alert">
              {erro}
            </p>
          )}
          <div className="acoes">
            <button className="btn fraco" data-f onClick={() => setPasso(1)}>
              Voltar
            </button>
            <button className="btn primario" data-f disabled={!pasta || !!check?.message} onClick={() => setPasso(3)}>
              Continuar
            </button>
          </div>
        </section>
      )}
      {passo === 3 && (
        <section>
          <h1 id="wiz-titulo">Como chamar este PC?</h1>
          <p>O nome aparece nos avisos, por exemplo “o jogo está aberto em PC-QUARTO”.</p>
          <label className="campo">
            Nome deste PC
            <input data-f data-autofocus value={nome} onChange={(e) => setNome(e.target.value)} maxLength={40} />
          </label>
          {erro && (
            <p className="erro" role="alert">
              {erro}
            </p>
          )}
          <div className="acoes">
            <button className="btn fraco" data-f onClick={() => setPasso(2)}>
              Voltar
            </button>
            <button className="btn primario" data-f disabled={ocupado || nome.trim() === ""} onClick={concluir}>
              {ocupado ? "Salvando…" : "Concluir"}
            </button>
          </div>
        </section>
      )}
    </main>
  );
}
