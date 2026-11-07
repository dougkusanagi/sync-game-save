// Som da interface, sintetizado com WebAudio (criado do zero; nenhum sample de terceiros).
// - Efeitos: curtos (40–250 ms), ataque suave, sem agudos estridentes.
// - Fundo: pad ambiente contínuo, sem melodia nem batida, que muda devagar para não cansar.
// - Volume baixo por padrão, controles separados, e silêncio com o jogo aberto ou a janela escondida.

export type ConfigSom = { efeitos: boolean; volEfeitos: number; fundo: boolean; volFundo: number };

export const PADRAO: ConfigSom = { efeitos: true, volEfeitos: 0.3, fundo: true, volFundo: 0.12 };

export function limitar(v: unknown): number {
  const n = typeof v === "number" && Number.isFinite(v) ? v : 0;
  return Math.min(1, Math.max(0, n));
}

/** Lê a configuração guardada. Qualquer dado estranho cai nos valores padrão. */
export function lerConfig(bruto: string | null): ConfigSom {
  if (!bruto) return { ...PADRAO };
  try {
    const o = JSON.parse(bruto) as Partial<ConfigSom>;
    return {
      efeitos: typeof o.efeitos === "boolean" ? o.efeitos : PADRAO.efeitos,
      volEfeitos: o.volEfeitos === undefined ? PADRAO.volEfeitos : limitar(o.volEfeitos),
      fundo: typeof o.fundo === "boolean" ? o.fundo : PADRAO.fundo,
      volFundo: o.volFundo === undefined ? PADRAO.volFundo : limitar(o.volFundo),
    };
  } catch {
    return { ...PADRAO };
  }
}

export type NomeEfeito = "foco" | "confirmar" | "voltar" | "painel" | "erro" | "sucesso";

type Nota = { freq: number; freqFim?: number; atraso: number };
type Efeito = { notas: Nota[]; dur: number; ataque: number; vol: number; tipo: OscillatorType };

/** Cada efeito dura entre 40 e 250 ms, com ataque suave e frequências abaixo de 2 kHz. */
export const EFEITOS: Record<NomeEfeito, Efeito> = {
  foco: { notas: [{ freq: 520, freqFim: 560, atraso: 0 }], dur: 0.06, ataque: 0.008, vol: 0.5, tipo: "sine" },
  confirmar: { notas: [{ freq: 440, atraso: 0 }, { freq: 660, atraso: 0.05 }], dur: 0.16, ataque: 0.01, vol: 0.7, tipo: "sine" },
  voltar: { notas: [{ freq: 520, freqFim: 380, atraso: 0 }], dur: 0.12, ataque: 0.01, vol: 0.55, tipo: "sine" },
  painel: { notas: [{ freq: 330, freqFim: 495, atraso: 0 }], dur: 0.2, ataque: 0.02, vol: 0.5, tipo: "triangle" },
  erro: { notas: [{ freq: 300, atraso: 0 }, { freq: 240, atraso: 0.08 }], dur: 0.22, ataque: 0.01, vol: 0.55, tipo: "triangle" },
  sucesso: { notas: [{ freq: 392, atraso: 0 }, { freq: 523, atraso: 0.07 }, { freq: 659, atraso: 0.14 }], dur: 0.24, ataque: 0.012, vol: 0.6, tipo: "sine" },
};

const CHAVE = "sgs.som";

class Som {
  private config: ConfigSom = { ...PADRAO };
  private ctx: AudioContext | null = null;
  private gEfeitos: GainNode | null = null;
  private gFundo: GainNode | null = null;
  private fundoIniciado = false;
  private jogando = false;
  private escondida = false;
  private ultimoFoco = 0;
  private oscilacoes: { osc: OscillatorNode; base: number }[] = [];

  iniciar(): void {
    try {
      this.config = lerConfig(localStorage.getItem(CHAVE));
    } catch {
      this.config = { ...PADRAO };
    }
    // O navegador só libera áudio depois de um gesto do usuário.
    const liberar = () => {
      this.garantirContexto();
      this.aplicar();
    };
    addEventListener("pointerdown", liberar, { once: true });
    addEventListener("keydown", liberar, { once: true });
    document.addEventListener("visibilitychange", () => {
      this.escondida = document.hidden;
      this.aplicar();
    });
  }

  obter(): ConfigSom {
    return { ...this.config };
  }

  definir(parcial: Partial<ConfigSom>): void {
    this.config = {
      ...this.config,
      ...parcial,
      volEfeitos: limitar(parcial.volEfeitos ?? this.config.volEfeitos),
      volFundo: limitar(parcial.volFundo ?? this.config.volFundo),
    };
    try {
      localStorage.setItem(CHAVE, JSON.stringify(this.config));
    } catch {
      /* sem armazenamento: vale só nesta sessão */
    }
    this.garantirContexto();
    this.aplicar();
  }

  /** Com um jogo aberto, o fundo e os efeitos ficam em silêncio. */
  definirJogando(v: boolean): void {
    this.jogando = v;
    this.aplicar();
  }

  tocar(nome: NomeEfeito): void {
    if (!this.config.efeitos || this.jogando || this.escondida || !this.ctx || !this.gEfeitos) return;
    if (this.ctx.state !== "running") return;
    // O "mover foco" dispara muito rápido ao segurar a seta; limita para não virar zumbido.
    if (nome === "foco") {
      const agora = performance.now();
      if (agora - this.ultimoFoco < 70) return;
      this.ultimoFoco = agora;
    }
    const e = EFEITOS[nome];
    const t0 = this.ctx.currentTime;
    // Variação leve de altura no foco, para não ficar repetitivo.
    const variacao = nome === "foco" ? 1 + (Math.random() - 0.5) * 0.06 : 1;
    for (const n of e.notas) {
      const osc = this.ctx.createOscillator();
      const g = this.ctx.createGain();
      osc.type = e.tipo;
      const ini = t0 + n.atraso;
      osc.frequency.setValueAtTime(n.freq * variacao, ini);
      if (n.freqFim) osc.frequency.linearRampToValueAtTime(n.freqFim * variacao, ini + e.dur);
      g.gain.setValueAtTime(0.0001, ini);
      g.gain.linearRampToValueAtTime(e.vol, ini + e.ataque);
      g.gain.exponentialRampToValueAtTime(0.0001, ini + e.dur);
      osc.connect(g).connect(this.gEfeitos);
      osc.start(ini);
      osc.stop(ini + e.dur + 0.02);
    }
  }

  private garantirContexto(): void {
    if (this.ctx || typeof AudioContext === "undefined") return;
    try {
      this.ctx = new AudioContext();
    } catch {
      return;
    }
    this.gEfeitos = this.ctx.createGain();
    this.gEfeitos.connect(this.ctx.destination);
    this.gFundo = this.ctx.createGain();
    this.gFundo.gain.value = 0;
    this.gFundo.connect(this.ctx.destination);
  }

  private aplicar(): void {
    const ctx = this.ctx;
    if (!ctx || !this.gEfeitos || !this.gFundo) return;
    const silencio = this.jogando || this.escondida;
    this.gEfeitos.gain.value = this.config.efeitos && !silencio ? this.config.volEfeitos : 0;
    const querFundo = this.config.fundo && !silencio;
    // Sobe e desce devagar para não dar estalo.
    this.gFundo.gain.cancelScheduledValues(ctx.currentTime);
    this.gFundo.gain.linearRampToValueAtTime(querFundo ? this.config.volFundo : 0, ctx.currentTime + 1.5);
    if (querFundo) {
      void ctx.resume();
      this.iniciarFundo();
    } else if (silencio || !this.config.fundo) {
      // Suspende só quando nada deve tocar, para não gastar CPU.
      window.setTimeout(() => {
        if (this.ctx && !this.config.efeitos && !this.config.fundo) void this.ctx.suspend();
        if (this.ctx && silencio) void this.ctx.suspend();
      }, 1800);
    }
  }

  /** Pad de 4 notas (lá menor com nona), com volume e afinação que mudam devagar. Sem melodia. */
  private iniciarFundo(): void {
    if (this.fundoIniciado || !this.ctx || !this.gFundo) return;
    this.fundoIniciado = true;
    const ctx = this.ctx;
    const filtro = ctx.createBiquadFilter();
    filtro.type = "lowpass";
    filtro.frequency.value = 900;
    filtro.connect(this.gFundo);
    for (const base of [110, 164.81, 220, 329.63]) {
      for (const desafino of [-4, 4]) {
        const osc = ctx.createOscillator();
        osc.type = "sine";
        osc.frequency.value = base;
        osc.detune.value = desafino;
        const g = ctx.createGain();
        g.gain.value = 0.05;
        // Respiração lenta do volume (um ciclo a cada 12–30 s).
        const lfo = ctx.createOscillator();
        lfo.frequency.value = 1 / (12 + Math.random() * 18);
        const prof = ctx.createGain();
        prof.gain.value = 0.035;
        lfo.connect(prof).connect(g.gain);
        osc.connect(g).connect(filtro);
        osc.start();
        lfo.start();
        this.oscilacoes.push({ osc, base });
      }
    }
    // A cada 20–40 s a afinação anda alguns centésimos, sem ruído nem salto.
    const andar = () => {
      for (const o of this.oscilacoes) {
        o.osc.detune.linearRampToValueAtTime((Math.random() - 0.5) * 16, ctx.currentTime + 18);
      }
      window.setTimeout(andar, 20000 + Math.random() * 20000);
    };
    andar();
  }
}

export const som = new Som();
