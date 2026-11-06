import { describe, expect, test } from "bun:test";
import { EFEITOS, PADRAO, lerConfig, limitar } from "./som";

describe("efeitos sonoros (limites do plano)", () => {
  for (const [nome, e] of Object.entries(EFEITOS)) {
    test(`${nome}: duração, ataque suave e sem agudos estridentes`, () => {
      const total = e.dur + Math.max(...e.notas.map((n) => n.atraso));
      expect(e.dur).toBeGreaterThanOrEqual(0.04);
      expect(e.dur).toBeLessThanOrEqual(0.25);
      expect(total).toBeLessThanOrEqual(0.45);
      expect(e.ataque).toBeGreaterThanOrEqual(0.005);
      expect(e.vol).toBeLessThanOrEqual(0.8);
      for (const n of e.notas) {
        expect(n.freq).toBeLessThanOrEqual(2000);
        expect(n.freqFim ?? n.freq).toBeLessThanOrEqual(2000);
        expect(n.freq).toBeGreaterThanOrEqual(150);
      }
    });
  }
});

describe("configuração", () => {
  test("padrão tem volume baixo", () => {
    expect(PADRAO.volEfeitos).toBeLessThanOrEqual(0.4);
    expect(PADRAO.volFundo).toBeLessThanOrEqual(0.2);
  });

  test("limita volume entre 0 e 1 e ignora lixo", () => {
    expect(limitar(5)).toBe(1);
    expect(limitar(-2)).toBe(0);
    expect(limitar(Number.NaN)).toBe(0);
    expect(limitar("alto")).toBe(0);
  });

  test("lê o que foi guardado e cai no padrão com dados inválidos", () => {
    expect(lerConfig(null)).toEqual(PADRAO);
    expect(lerConfig("isso não é json")).toEqual(PADRAO);
    expect(lerConfig(JSON.stringify({ efeitos: false, volFundo: 9, volEfeitos: "x" }))).toEqual({
      efeitos: false,
      volEfeitos: 0,
      fundo: PADRAO.fundo,
      volFundo: 1,
    });
  });
});
