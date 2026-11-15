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

export type TipoArte = "header" | "hero" | "logo" | "cover";

export type NewGame = Pick<Game, "name" | "exe" | "install_dir" | "catalog_id" | "save_paths">;

export type Estado = {
  pasta_sync: string | null;
  nome_pc: string;
  catalogo_pronto: boolean;
  operacoes_interrompidas: string[];
  steam_configurada: boolean;
  steam_id: string | null;
  steamgriddb_configurada: boolean;
};

export type Nivel = "Common" | "Uncommon" | "Rare" | "UltraRare";

export type Conquista = {
  id: string;
  name: string;
  description: string;
  icon: string;
  icon_gray: string;
  hidden: boolean;
  unlocked: boolean;
  unlock_time: number | null;
  percent: number | null;
  tier: Nivel | null;
};

export type Conquistas = {
  set: { achievements: Conquista[]; progress_known: boolean };
  summary: { unlocked: number; total: number; percent: number; complete: boolean; by_tier: Record<string, number> };
  from_cache: boolean;
  fetched_at: number;
  notice: string | null;
};

export type Tema = {
  id: string;
  name: string;
  author: string;
  version: string;
  builtin: boolean;
  tokens: Record<string, string>;
};

export type Temas = { atual: string; temas: Tema[] };

export type MarcoHub = {
  id: string;
  nome: string;
  descricao: string;
  nivel: "bronze" | "prata" | "ouro";
  progresso: number;
  alvo: number;
  unidade: string;
  desbloqueada: boolean;
};

export type ConquistasHub = {
  conquistas: MarcoHub[];
  desbloqueadas: number;
  total: number;
  horas: number;
  jogos: number;
  sequencia_atual: number;
  melhor_sequencia: number;
  jogos_completos: number;
};

export type VerificacaoPasta = {
  exists: boolean;
  writable: boolean;
  has_data: boolean;
  games: string[];
  message: string | null;
};

export type SaveCandidate = { logical: string; resolved: string; exists: boolean; files: number; selected: boolean };

/** Pasta descoberta pela mudança entre a captura e o fim do jogo. */
export type SaveDescoberto = { logical: string; resolved: string; added: number; removed: number; bytes: number };

export type Reconhecimento = {
  name: string;
  catalog_id: number | null;
  confidence: "High" | "Low" | "None";
  exe: string | null;
  save_candidates: SaveCandidate[];
  alternatives: { id: number; name: string }[];
};

export type ResultadoVarredura = { recognized: [string, Reconhecimento][]; needs_confirmation: [string, Reconhecimento][]; unknown: [string, Reconhecimento][]; errors: [string, string][] };

export type ResultadoJogar =
  | { kind: "conflict" }
  | { kind: "locked"; pc: string }
  | { kind: "finished"; exit_code: number; synced: boolean; notice: string };

export type ResultadoPrimeiroBackup =
  | { kind: "created" | "already_synced" | "nothing_to_backup" | "needs_decision" | "in_use" }
  | { kind: "locked"; pc: string };

export type OpcoesJogar = { forcar?: boolean; conflito?: "local" | "nuvem"; sem_sincronizar?: boolean };

/** Dentro do Tauri existe `__TAURI_INTERNALS__`; no navegador (testes e capturas) usa dados simulados. */
export const noTauri = typeof window !== "undefined" && !("__TAURI_INTERNALS__" in window);

function call<T>(cmd: string, args?: Record<string, unknown>): Promise<T> {
  return noTauri ? mock<T>(cmd, args) : invoke<T>(cmd, args);
}

export const api = {
  sobre: () => call<{ nome: string; versao: string; build: string }>("sobre"),
  estado: () => call<Estado>("estado"),
  verificarPastaSync: (pasta: string) => call<VerificacaoPasta>("verificar_pasta_sync", { pasta }),
  definirPastaSync: (pasta: string) => call<VerificacaoPasta>("definir_pasta_sync", { pasta }),
  definirNomePc: (nome: string) => call<void>("definir_nome_pc", { nome }),
  atualizarCatalogo: () => call<number>("atualizar_catalogo"),
  escolherCandidatoCatalogo: (pasta: string, id: number) => call<Reconhecimento>("escolher_candidato_catalogo", { pasta, id }),
  varrerPasta: (pasta: string, profundidade: number) => call<ResultadoVarredura>("varrer_pasta", { pasta, profundidade }),
  primeiroBackup: (id: number) => call<ResultadoPrimeiroBackup>("primeiro_backup", { id }),
  reconhecerPasta: (pasta: string) => call<Reconhecimento>("reconhecer_pasta", { pasta }),
  converterCaminho: (caminho: string) => call<string>("converter_caminho", { caminho }),
  descobrirPastaSave: (exe: string) => call<number>("descobrir_save", { exe }),
  candidatosSave: (exe: string) => call<SaveDescoberto[]>("candidatos_save", { exe }),
  listarJogos: () => call<Game[]>("listar_jogos"),
  adicionarJogo: (jogo: NewGame) => call<number>("adicionar_jogo", { jogo }),
  editarJogo: (id: number, jogo: NewGame) => call<void>("editar_jogo", { id, jogo }),
  removerJogo: (id: number) => call<void>("remover_jogo", { id }),
  jogar: (id: number, opcoes?: OpcoesJogar) => call<ResultadoJogar>("jogar", { id, opcoes }),
  pararJogo: (id: number) => call<boolean | null>("parar_jogo", { id }),
  forcarParadaJogo: (id: number) => call<boolean>("forcar_parada_jogo", { id }),
  capaDoJogo: (id: number) => call<string | null>("capa_do_jogo", { id }),
  arteDoJogo: (id: number, tipo: TipoArte) => call<string | null>("arte_do_jogo", { id, tipo }),
  versoes: (id: number) => call<string[]>("versoes", { id }),
  restaurarVersao: (id: number, versao: string) => call<string>("restaurar_versao", { id, versao }),
  conquistas: (id: number, forcar = false) => call<Conquistas>("conquistas", { id, forcar }),
  definirCredenciaisSteam: (chave: string, steamId: string) =>
    call<void>("definir_credenciais_steam", { chave, steamId }),
  definirChaveSteamGridDB: (chave: string) => call<void>("definir_chave_steamgriddb", { chave }),
  conquistasHub: () => call<ConquistasHub>("conquistas_hub"),
  temas: () => call<Temas>("temas"),
  definirTema: (id: string) => call<Temas>("definir_tema", { id }),
};

export type Atualizacao = {
  versao: string;
  atual: string;
  notas: string | null;
};

/** Procura uma atualização no canal configurado. No navegador simulado nunca há uma. */
export async function verificarAtualizacao(): Promise<Atualizacao | null> {
  if (noTauri) return null;
  const { check } = await import("@tauri-apps/plugin-updater");
  const u = await check();
  if (!u) return null;
  return { versao: u.version, atual: u.currentVersion, notas: u.body ?? null };
}

/** Baixa, instala e reabre o app. Só termina se algo der errado (o app reinicia). */
export async function instalarAtualizacao(): Promise<void> {
  if (noTauri) return;
  const { check } = await import("@tauri-apps/plugin-updater");
  const { relaunch } = await import("@tauri-apps/plugin-process");
  const u = await check();
  if (!u) throw new Error("Nenhuma atualização disponível.");
  await u.downloadAndInstall();
  await relaunch();
}

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
  if (noTauri) {
    const tratar = (e: Event) => { const { id, estado } = (e as CustomEvent<{ id: number; estado: string }>).detail; fn(id, estado); };
    window.addEventListener("sgs:mock-jogo", tratar);
    return Promise.resolve(() => window.removeEventListener("sgs:mock-jogo", tratar));
  }
  return listen<{ id: number; estado: string }>("jogo-estado", (e) => fn(e.payload.id, e.payload.estado));
}

/** Mensagem de erro em português, venha do Rust (texto) ou do navegador. */
export function textoDoErro(e: unknown): string {
  if (typeof e === "string") return e;
  if (e instanceof Error) return e.message;
  return "Algo deu errado. Tente de novo.";
}

/** Ações do controle lidas no Rust (só chegam com a janela visível e focada). */
export function aoUsarControle(fn: (acao: string, dispositivo: string) => void): Promise<() => void> {
  if (noTauri) return Promise.resolve(() => {});
  return listen<{ acao: string; dispositivo: string }>("controle-acao", (e) => fn(e.payload.acao, e.payload.dispositivo));
}
