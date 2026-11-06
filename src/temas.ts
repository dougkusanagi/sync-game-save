import type { Tema, Temas } from "./api";

const aplicadas = new Set<string>();

/** Troca os tokens do tema atual no elemento raiz. Tokens de fora de `--sgs-` são ignorados. */
export function aplicarTema(tema: Tema | null): void {
  const raiz = document.documentElement;
  for (const chave of aplicadas) raiz.style.removeProperty(chave);
  aplicadas.clear();
  raiz.dataset.tema = tema?.id ?? "";
  if (!tema) return;
  for (const [chave, valor] of Object.entries(tema.tokens)) {
    if (!chave.startsWith("--sgs-")) continue;
    raiz.style.setProperty(chave, valor);
    aplicadas.add(chave);
  }
}

export function acharTema(lista: Temas | null): Tema | null {
  return lista?.temas.find((t) => t.id === lista.atual) ?? null;
}
