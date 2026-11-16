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

/** O WebView nativo pode estar disponível antes da configuração do runtime Wails. */
const janelaNativa = typeof window === "undefined" ? undefined : window as Window & {
  _wails?: { environment?: unknown };
  chrome?: { webview?: unknown };
  webkit?: { messageHandlers?: { external?: unknown } };
};
export const emWails = !!(janelaNativa?._wails?.environment || janelaNativa?.chrome?.webview || janelaNativa?.webkit?.messageHandlers?.external);
/** Fora do app (navegador, testes), a interface usa dados simulados. */
export const emMock = !emWails;

function call<T>(cmd: string, args?: Record<string, unknown>): Promise<T> {
  if (emWails) {
    const comandos: Record<string, { metodo: string; argumentos: unknown[] }> = {
      estado: { metodo: "Estado", argumentos: [] },
      verificar_pasta_sync: { metodo: "VerificarPastaSync", argumentos: [args?.pasta] },
      definir_pasta_sync: { metodo: "DefinirPastaSync", argumentos: [args?.pasta] },
      definir_nome_pc: { metodo: "DefinirNomePc", argumentos: [args?.nome] },
      atualizar_catalogo: { metodo: "AtualizarCatalogo", argumentos: [] },
      reconhecer_pasta: { metodo: "ReconhecerPasta", argumentos: [args?.pasta] },
      converter_caminho: { metodo: "ConverterCaminho", argumentos: [args?.caminho] },
      listar_jogos: { metodo: "ListarJogos", argumentos: [] },
      versoes: { metodo: "Versoes", argumentos: [args?.id] },
      restaurar_versao: { metodo: "RestaurarVersao", argumentos: [args?.id, args?.versao] },
      temas: { metodo: "Temas", argumentos: [] },
      definir_tema: { metodo: "DefinirTema", argumentos: [args?.id] },
      descobrir_save: { metodo: "DescobrirSave", argumentos: [args?.exe] },
      candidatos_save: { metodo: "CandidatosSave", argumentos: [args?.exe] },
      capa_do_jogo: { metodo: "CapaDoJogo", argumentos: [args?.id] },
      arte_do_jogo: { metodo: "ArteDoJogo", argumentos: [args?.id, args?.tipo] },
      conquistas: { metodo: "Conquistas", argumentos: [args?.id, args?.forcar ?? null] },
      conquistas_hub: { metodo: "ConquistasHub", argumentos: [] },
      definir_credenciais_steam: { metodo: "DefinirCredenciaisSteam", argumentos: [args?.chave, args?.steamId] },
      definir_chave_steamgriddb: { metodo: "DefinirChaveSteamGridDB", argumentos: [args?.chave] },
      escolher_candidato_catalogo: { metodo: "EscolherCandidatoCatalogo", argumentos: [args?.pasta, args?.id] },
      varrer_pasta: { metodo: "VarrerPasta", argumentos: [args?.pasta, args?.profundidade] },
      primeiro_backup: { metodo: "PrimeiroBackup", argumentos: [args?.id] },
      sair_app: { metodo: "SairApp", argumentos: [] },
      abrir_pasta_de_logs: { metodo: "AbrirPastaDeLogs", argumentos: [] },
      jogar: { metodo: "Jogar", argumentos: [args?.id, args?.opcoes ?? null] },
      parar_jogo: { metodo: "PararJogo", argumentos: [args?.id] },
      forcar_parada_jogo: { metodo: "ForcarParadaJogo", argumentos: [args?.id] },
      adicionar_jogo: { metodo: "AdicionarJogo", argumentos: [args?.jogo] },
      editar_jogo: { metodo: "EditarJogo", argumentos: [args?.id, args?.jogo] },
      remover_jogo: { metodo: "RemoverJogo", argumentos: [args?.id] },
    };
    const comando = comandos[cmd];
    if (comando) {
      return import("@wailsio/runtime").then(({ Call }) => Call.ByName(`main.App.${comando.metodo}`, ...comando.argumentos)) as Promise<T>;
    }
    return Promise.reject(new Error("Esta versão ainda não consegue abrir sua biblioteca."));
  }
  return mock<T>(cmd, args);
}
export const api = {
  sobre: async () => {
    if (emWails) {
      const { Call } = await import("@wailsio/runtime");
      return await Call.ByName("main.App.Sobre") as { nome: string; versao: string; build: string };
    }
    return call<{ nome: string; versao: string; build: string }>("sobre");
  },
  estado: () => call<Estado>("estado"),
  verificarPastaSync: (pasta: string) => call<VerificacaoPasta>("verificar_pasta_sync", { pasta }),
  definirPastaSync: (pasta: string) => call<VerificacaoPasta>("definir_pasta_sync", { pasta }),
  definirNomePc: (nome: string) => call<void>("definir_nome_pc", { nome }),
  atualizarCatalogo: () => call<number>("atualizar_catalogo"),
  escolherCandidatoCatalogo: (pasta: string, id: number) => call<Reconhecimento>("escolher_candidato_catalogo", { pasta, id }),
  varrerPasta: (pasta: string, profundidade: number) => call<ResultadoVarredura>("varrer_pasta", { pasta, profundidade }),
  sairDoApp: () => call<void>("sair_app"),
  abrirPastaDeLogs: () => call<void>("abrir_pasta_de_logs"),
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
  if (emWails) {
    const { Call } = await import("@wailsio/runtime");
    return (await Call.ByName("main.App.VerificarAtualizacao")) as Atualizacao | null;
  }
  return null;
}

/** Baixa, instala e reabre o app. Só termina se algo der errado (o app reinicia). */
export async function instalarAtualizacao(): Promise<void> {
  if (emWails) {
    const { Call } = await import("@wailsio/runtime");
    await Call.ByName("main.App.InstalarAtualizacao");
    return;
  }
}

/** Abre o seletor nativo de pasta. No navegador simulado devolve um caminho de exemplo. */
export async function escolherPasta(titulo: string): Promise<string | null> {
  if (emWails) {
    const { Dialogs } = await import("@wailsio/runtime");
    return (await Dialogs.OpenFile({ Title: titulo, CanChooseDirectories: true, CanChooseFiles: false })) || null;
  }
  return "C:/Exemplo/Pasta";
}

export async function escolherArquivoExe(titulo: string): Promise<string | null> {
  if (emWails) {
    const { Dialogs } = await import("@wailsio/runtime");
    return (await Dialogs.OpenFile({ Title: titulo, CanChooseFiles: true, Filters: [{ DisplayName: "Programas", Pattern: "*.exe" }] })) || null;
  }
  return "C:/Exemplo/jogo.exe";
}

export function aoMudarEstadoDoJogo(fn: (id: number, estado: string) => void): Promise<() => void> {
  if (emWails) {
    return import("@wailsio/runtime").then(({ Events }) => Events.On("jogo-estado", (e) => {
      const dados = e.data as { id: number; estado: string };
      fn(dados.id, dados.estado);
    }));
  }
  const tratar = (e: Event) => { const { id, estado } = (e as CustomEvent<{ id: number; estado: string }>).detail; fn(id, estado); };
  window.addEventListener("sgs:mock-jogo", tratar);
  return Promise.resolve(() => window.removeEventListener("sgs:mock-jogo", tratar));
}

/** Mensagem de erro em português, venha do Go (texto) ou do navegador. */
export function textoDoErro(e: unknown): string {
  if (typeof e === "string") return e;
  if (e instanceof Error) return e.message;
  return "Algo deu errado. Tente de novo.";
}

/** Ações do controle lidas no Go (só chegam com a janela visível e focada). */
export function aoUsarControle(fn: (acao: string, dispositivo: string) => void): Promise<() => void> {
  if (emWails) {
    return import("@wailsio/runtime").then(({ Events }) => Events.On("controle-acao", (e) => {
      const dados = e.data as { acao: string; dispositivo: string };
      fn(dados.acao, dados.dispositivo);
    }));
  }
  return Promise.resolve(() => {});
}
