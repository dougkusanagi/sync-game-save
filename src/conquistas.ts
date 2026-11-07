import type { Conquista, Nivel } from "./api";

export type Filtro = "todas" | "desbloqueadas" | "bloqueadas";
export type Raridade = Nivel | "todas";
export type Ordem = "recentes" | "raridade" | "nome";

export const NIVEL: Record<Nivel, string> = {
  Common: "Comum",
  Uncommon: "Incomum",
  Rare: "Rara",
  UltraRare: "Ultrarrara",
};

export const NIVEIS: Nivel[] = ["Common", "Uncommon", "Rare", "UltraRare"];

export function nivelRotulo(n: Nivel | null): string {
  return n ? NIVEL[n] : "Sem raridade";
}

/** Conta as conquistas desbloqueadas por nível. */
export function contarPorNivel(lista: Conquista[]): Record<Nivel, number> {
  const r: Record<Nivel, number> = { Common: 0, Uncommon: 0, Rare: 0, UltraRare: 0 };
  for (const a of lista) {
    if (a.unlocked && a.tier) r[a.tier]++;
  }
  return r;
}

const porNome = new Intl.Collator("pt-BR", { sensitivity: "base" });

/** Filtra por estado e raridade e ordena. Conquistas sem raridade ficam por último. */
export function filtrarOrdenar(lista: Conquista[], filtro: Filtro, raridade: Raridade, ordem: Ordem): Conquista[] {
  const r = lista.filter((a) => {
    if (filtro === "desbloqueadas" && !a.unlocked) return false;
    if (filtro === "bloqueadas" && a.unlocked) return false;
    if (raridade !== "todas" && a.tier !== raridade) return false;
    return true;
  });
  return r.sort((a, b) => {
    if (ordem === "nome") return porNome.compare(a.name, b.name);
    if (ordem === "raridade") {
      const pa = a.percent ?? Number.POSITIVE_INFINITY;
      const pb = b.percent ?? Number.POSITIVE_INFINITY;
      if (pa !== pb) return pa - pb;
      return porNome.compare(a.name, b.name);
    }
    const ta = a.unlock_time ?? 0;
    const tb = b.unlock_time ?? 0;
    if (a.unlocked !== b.unlocked) return a.unlocked ? -1 : 1;
    if (ta !== tb) return tb - ta;
    return porNome.compare(a.name, b.name);
  });
}

export function dataDoDesbloqueio(unix: number | null): string | null {
  if (!unix) return null;
  return new Date(unix * 1000).toLocaleString("pt-BR", { dateStyle: "short", timeStyle: "short" });
}
