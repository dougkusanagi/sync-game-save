import { invoke } from "@tauri-apps/api/core";
import { listen } from "@tauri-apps/api/event";
import { open } from "@tauri-apps/plugin-dialog";
import { mock } from "./mock";

export type Game = {
  id: number;
  name: string;
  exe: string;
  install_dir: string | null;
  catalog_id: number | null;
  save_paths: string[];
  base_digest: string | null;
  last_sync_at: number | null;
  playtime_secs: number;
  last_played_at: number | null;
};

export type NewGame = Pick<Game, "name" | "exe" | "install_dir" | "catalog_id" | "save_paths">;

export type Estado = {
  pasta_sync: string | null;
  nome_pc: string;
  catalogo_pronto: boolean;
  operacoes_interrompidas: string[];
};

export type VerificacaoPasta = {
  exists: boolean;
  writable: boolean;
  has_data: boolean;
  games: string[];
  message: string | null;
};

export type SaveCandidate = { logical: string; resolved: string; exists: boolean; files: number; selected: boolean };

export type Reconhecimento = {
  name: string;
  catalog_id: number | null;
  confidence: "High" | "Low" | "None";
  exe: string | null;
  save_candidates: SaveCandidate[];
  alternatives: { id: number; name: string }[];
};

export type ResultadoJogar =
  | { kind: "conflict" }
  | { kind: "locked"; pc: string }
  | { kind: "finished"; exit_code: number; synced: boolean; notice: string };

export type OpcoesJogar = { forcar?: boolean; conflito?: "local" | "nuvem"; sem_sincronizar?: boolean };

/** Dentro do Tauri existe `__TAURI_INTERNALS__`; no navegador (testes e capturas) usa dados simulados. */
export const noTauri = typeof window !== "undefined" && !("__TAURI_INTERNALS__" in window);

function call<T>(cmd: string, args?: Record<string, unknown>): Promise<T> {
  return noTauri ? mock<T>(cmd, args) : invoke<T>(cmd, args);
}

export const api = {
  sobre: () => call<{ nome: string; versao: string }>("sobre"),
  estado: () => call<Estado>("estado"),
  verificarPastaSync: (pasta: string) => call<VerificacaoPasta>("verificar_pasta_sync", { pasta }),
  definirPastaSync: (pasta: string) => call<VerificacaoPasta>("definir_pasta_sync", { pasta }),
  definirNomePc: (nome: string) => call<void>("definir_nome_pc", { nome }),
  atualizarCatalogo: () => call<number>("atualizar_catalogo"),
  reconhecerPasta: (pasta: string) => call<Reconhecimento>("reconhecer_pasta", { pasta }),
  converterCaminho: (caminho: string) => call<string>("converter_caminho", { caminho }),
  listarJogos: () => call<Game[]>("listar_jogos"),
  adicionarJogo: (jogo: NewGame) => call<number>("adicionar_jogo", { jogo }),
  removerJogo: (id: number) => call<void>("remover_jogo", { id }),
  jogar: (id: number, opcoes?: OpcoesJogar) => call<ResultadoJogar>("jogar", { id, opcoes }),
  capaDoJogo: (id: number) => call<string | null>("capa_do_jogo", { id }),
  versoes: (id: number) => call<string[]>("versoes", { id }),
  restaurarVersao: (id: number, versao: string) => call<string>("restaurar_versao", { id, versao }),
};

/** Abre o seletor nativo de pasta. No navegador simulado devolve um caminho de exemplo. */
export async function escolherPasta(titulo: string): Promise<string | null> {
  if (noTauri) return "C:/Exemplo/Pasta";
  const r = await open({ directory: true, multiple: false, title: titulo });
  return typeof r === "string" ? r : null;
}

export async function escolherArquivoExe(titulo: string): Promise<string | null> {
  if (noTauri) return "C:/Exemplo/jogo.exe";
  const r = await open({ multiple: false, title: titulo, filters: [{ name: "Programas", extensions: ["exe"] }] });
  return typeof r === "string" ? r : null;
}

export function aoMudarEstadoDoJogo(fn: (id: number, estado: string) => void): Promise<() => void> {
  if (noTauri) return Promise.resolve(() => {});
  return listen<{ id: number; estado: string }>("jogo-estado", (e) => fn(e.payload.id, e.payload.estado));
}

/** Mensagem de erro em português, venha do Rust (texto) ou do navegador. */
export function textoDoErro(e: unknown): string {
  if (typeof e === "string") return e;
  if (e instanceof Error) return e.message;
  return "Algo deu errado. Tente de novo.";
}
