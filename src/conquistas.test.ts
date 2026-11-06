import { describe, expect, test } from "bun:test";
import type { Conquista } from "./api";
import { contarPorNivel, dataDoDesbloqueio, filtrarOrdenar, nivelRotulo } from "./conquistas";

const a = (id: string, extra: Partial<Conquista> = {}): Conquista => ({
  id,
  name: id,
  description: "",
  icon: "",
  icon_gray: "",
  hidden: false,
  unlocked: false,
  unlock_time: null,
  percent: null,
  tier: null,
  ...extra,
});

const lista: Conquista[] = [
  a("A", { unlocked: true, unlock_time: 100, percent: 80, tier: "Common" }),
  a("B", { percent: 10, tier: "Rare" }),
  a("C", { unlocked: true, unlock_time: 300, percent: 1, tier: "UltraRare" }),
  a("D", { percent: 30, tier: "Uncommon" }),
  a("E"),
];

describe("conquistas", () => {
  test("conta apenas as desbloqueadas por nível", () => {
    expect(contarPorNivel(lista)).toEqual({ Common: 1, Uncommon: 0, Rare: 0, UltraRare: 1 });
  });

  test("filtra por estado", () => {
    expect(filtrarOrdenar(lista, "desbloqueadas", "todas", "nome").map((x) => x.id)).toEqual(["A", "C"]);
    expect(filtrarOrdenar(lista, "bloqueadas", "todas", "nome").map((x) => x.id)).toEqual(["B", "D", "E"]);
  });

  test("filtra por raridade e deixa sem raridade de fora", () => {
    expect(filtrarOrdenar(lista, "todas", "Rare", "nome").map((x) => x.id)).toEqual(["B"]);
  });

  test("ordena por mais recentes, raridade e nome", () => {
    expect(filtrarOrdenar(lista, "todas", "todas", "recentes").map((x) => x.id)).toEqual(["C", "A", "B", "D", "E"]);
    expect(filtrarOrdenar(lista, "todas", "todas", "raridade").map((x) => x.id)).toEqual(["C", "B", "D", "A", "E"]);
    expect(filtrarOrdenar(lista, "todas", "todas", "nome").map((x) => x.id)).toEqual(["A", "B", "C", "D", "E"]);
  });

  test("rótulos e datas em português", () => {
    expect(nivelRotulo("UltraRare")).toBe("Ultrarrara");
    expect(nivelRotulo(null)).toBe("Sem raridade");
    expect(dataDoDesbloqueio(null)).toBeNull();
    expect(dataDoDesbloqueio(0)).toBeNull();
    expect(dataDoDesbloqueio(1700000000)).toContain("2023");
  });
});
