/** URL da arte em cache servida pelo Wails, sem expor o caminho local. */
export function urlArteWails(caminho: string): string | null {
  const nome = caminho.split(/[\\/]/).pop();
  if (!nome || nome === "." || nome === ".." || nome.includes("\0")) return null;
  return `/capa/${encodeURIComponent(nome)}`;
}
