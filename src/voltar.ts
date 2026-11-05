// Pilha de ações do "voltar" (Esc, B no controle): a camada de cima responde primeiro.
const pilha: Array<() => void> = [];

export function empilharVoltar(fn: () => void): () => void {
  pilha.push(fn);
  return () => {
    const i = pilha.lastIndexOf(fn);
    if (i >= 0) pilha.splice(i, 1);
  };
}

export function voltar(): boolean {
  const fn = pilha.at(-1);
  if (!fn) return false;
  fn();
  return true;
}
