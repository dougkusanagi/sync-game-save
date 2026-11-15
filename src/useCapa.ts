import { useEffect, useState } from "react";
import { api, emWails, type Game, type TipoArte } from "./api";
import { urlArteWails } from "./arteWails";

const cache = new Map<string, string | null>();
const aguardando = new Map<string, Promise<string | null>>();

function buscar(id: number, tipo: TipoArte): Promise<string | null> {
  const chave = `${id}:${tipo}`;
  if (cache.has(chave)) return Promise.resolve(cache.get(chave) ?? null);
  const emAndamento = aguardando.get(chave);
  if (emAndamento) return emAndamento;
  const pedido = api
    .arteDoJogo(id, tipo)
    .then((p) => {
      // No navegador (dados simulados), a arte já vem como URL.
      const src = !p ? null : emWails ? urlArteWails(p) : (p.startsWith("https://") ? p : null);
      cache.set(chave, src);
      return src;
    })
    .catch(() => {
      cache.set(chave, null);
      return null;
    })
    .finally(() => aguardando.delete(chave));
  aguardando.set(chave, pedido);
  return pedido;
}

/** Endereço local da arte do jogo (capa, fundo, logo ou capa vertical), reaproveitado entre as telas. */
export function useArte(jogo: Game | null, tipo: TipoArte): string | null {
  const chave = jogo ? `${jogo.id}:${tipo}` : "";
  const [src, setSrc] = useState<string | null>(() => (jogo ? (cache.get(chave) ?? null) : null));

  useEffect(() => {
    if (!jogo) {
      setSrc(null);
      return;
    }
    let vivo = true;
    setSrc(cache.get(chave) ?? null);
    void buscar(jogo.id, tipo).then((s) => {
      if (vivo) setSrc(s);
    });
    return () => {
      vivo = false;
    };
  }, [chave]);

  return src;
}

/** Capa horizontal do jogo. */
export const useCapa = (jogo: Game | null) => useArte(jogo, "header");
