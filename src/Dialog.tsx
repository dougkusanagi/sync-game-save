import { useEffect, useRef, type ReactNode } from "react";
import * as focus from "./nav/focus.js";
import { empilharVoltar } from "./voltar";

type Props = {
  titulo: string;
  onClose: () => void;
  children: ReactNode;
  /** Muda quando o conteúdo carrega, para o foco ir ao primeiro item. */
  focoKey?: unknown;
  largo?: boolean;
};

/** Camada modal: o foco fica dentro dela e volta ao botão de origem ao fechar. */
export function Dialog({ titulo, onClose, children, focoKey, largo }: Props) {
  const ref = useRef<HTMLDivElement>(null);
  const fechar = useRef(onClose);
  fechar.current = onClose;

  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    focus.pushLayer(el, {});
    // O foco inicial vai para a primeira ação do conteúdo, não para o botão Fechar do topo.
    const corpo = el.querySelector(".dialogo-corpo");
    if (corpo) focus.focusFirst(corpo);
    const tirar = empilharVoltar(() => fechar.current());
    return () => {
      tirar();
      focus.popLayer();
    };
  }, []);

  useEffect(() => {
    const el = ref.current;
    if (el) focus.focusFirst(el.querySelector(".dialogo-corpo") ?? el);
  }, [focoKey]);

  return (
    <div className="veu" onMouseDown={(e) => e.target === e.currentTarget && onClose()}>
      <div ref={ref} className={`dialogo${largo ? " largo" : ""}`} role="dialog" aria-modal="true" aria-label={titulo}>
        <header className="dialogo-topo">
          <h2>{titulo}</h2>
          <button className="btn fraco" data-f onClick={onClose} aria-label="Fechar">
            Fechar
          </button>
        </header>
        <div className="dialogo-corpo">{children}</div>
      </div>
    </div>
  );
}
