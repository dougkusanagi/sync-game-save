export function duracao(secs: number): string {
  if (secs < 60) return secs === 0 ? "Ainda não jogado" : "Menos de 1 min";
  const h = Math.floor(secs / 3600);
  const m = Math.floor((secs % 3600) / 60);
  if (h === 0) return `${m} min`;
  return m === 0 ? `${h} h` : `${h} h ${m} min`;
}

export function haQuanto(unix: number | null, agora = Date.now() / 1000): string {
  if (unix === null) return "nunca";
  const d = Math.max(0, agora - unix);
  if (d < 60) return "agora há pouco";
  if (d < 3600) return `há ${Math.floor(d / 60)} min`;
  if (d < 86400) return `há ${Math.floor(d / 3600)} h`;
  return `há ${Math.floor(d / 86400)} dia(s)`;
}

/** "1793920203_PC-SALA" vira data e nome do PC. */
export function versaoLegivel(id: string): { quando: string; pc: string } {
  const [secs, ...resto] = id.split("_");
  const n = Number(secs);
  const quando = Number.isFinite(n) && n > 0 ? new Date(n * 1000).toLocaleString("pt-BR", { dateStyle: "short", timeStyle: "short" }) : id;
  return { quando, pc: resto.join("_").replace(/~sgs-\d+$/, "") || "PC desconhecido" };
}
