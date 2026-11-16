// Dados simulados para rodar a interface no navegador (testes E2E e capturas de tela).
// Cenário pela URL: ?cenario=primeiro-uso | vazio | jogos (padrão) | conflito
import type { Conquista, Conquistas, ConquistasHub, Game, Estado, MarcoHub, Reconhecimento, Tema } from "./api";

const cenario = new URLSearchParams(location.search).get("cenario") ?? "jogos";
let terminarJogo: (() => void) | undefined;
let backupLiberado = false;

const STEAM: Record<string, number> = { "Rogue Legacy": 241600, "Hollow Knight": 367520, Celeste: 504230, Hades: 1145360, "Animal Well": 813230, "Sea of Stars": 1244090 };

const jogo = (id: number, name: string, extra: Partial<Game> = {}): Game => ({
  id,
  name,
  exe: `C:/Games/${name}/${name}.exe`,
  install_dir: `C:/Games/${name}`,
  catalog_id: null,
  save_paths: ["<winDocuments>/SavedGames/" + name],
  base_digest: "abc",
  last_sync_at: Math.floor(Date.now() / 1000) - 3600,
  playtime_secs: 7200,
  last_played_at: Math.floor(Date.now() / 1000) - 3600,
  ...extra,
});

let estado: Estado = {
  pasta_sync: cenario === "primeiro-uso" ? null : "D:/OneDrive/Saves",
  nome_pc: "PC-SALA",
  catalogo_pronto: true,
  operacoes_interrompidas: [],
  steam_configurada: false,
  steam_id: "76561198000000000",
  steamgriddb_configurada: false,
};

const temas: Tema[] = [
  { id: "escuro", name: "Escuro (padrão)", author: "Sync Game Save", version: "0.1.0", builtin: true, tokens: {} },
  {
    id: "claro",
    name: "Claro",
    author: "Sync Game Save",
    version: "0.1.0",
    builtin: true,
    tokens: {
      "--sgs-bg": "#f2f5fb",
      "--sgs-bg-rgb": "242 245 251",
      "--sgs-surface": "rgb(255 255 255 / 0.72)",
      "--sgs-surface-strong": "rgb(255 255 255 / 0.92)",
      "--sgs-surface-solid": "#ffffff",
      "--sgs-surface-hover": "rgb(16 22 37 / 0.06)",
      "--sgs-border": "rgb(16 22 37 / 0.14)",
      "--sgs-border-strong": "rgb(16 22 37 / 0.32)",
      "--sgs-text": "#101625",
      "--sgs-text-dim": "rgb(16 22 37 / 0.72)",
      "--sgs-text-faint": "rgb(16 22 37 / 0.55)",
      "--sgs-accent-default": "#1d6fe0",
      "--sgs-ok": "#0f8a60",
      "--sgs-info": "#1d6fe0",
      "--sgs-warn": "#9a6500",
      "--sgs-danger": "#c62d45",
      "--sgs-tier-common": "#5b6b82",
      "--sgs-tier-uncommon": "#0f8a60",
      "--sgs-tier-rare": "#1d6fe0",
      "--sgs-tier-ultra": "#9a6500",
      "--sgs-focus-ring": "0 0 0 0.1875rem #1d6fe0, 0 0.75rem 2.5rem rgb(16 22 37 / 0.2)",
      "--sgs-capa-bg": "linear-gradient(135deg, #dfe7f5, #eef2f9)",
      "--sgs-color-scheme": "light",
    },
  },
];
const CHAVE_TEMA = "sgs.mock.tema";
const temaGuardado = localStorage.getItem(CHAVE_TEMA);
let temaAtual = temas.some((t) => t.id === temaGuardado) ? (temaGuardado as string) : "escuro";

const conquista = (extra: Partial<Conquista> & Pick<Conquista, "id" | "name">): Conquista => ({
  description: "",
  icon: "",
  icon_gray: "",
  hidden: false,
  unlocked: false,
  unlock_time: null,
  percent: null,
  tier: null,
  ...extra,
});

const conquistasDoJogo = (): Conquistas => {
  const achievements: Conquista[] = [
    conquista({
      id: "ACH_ROGUE_1",
      name: "Primeira morte",
      description: "Morra pela primeira vez na mansão.",
      unlocked: true,
      unlock_time: Math.floor(Date.now() / 1000) - 86400 * 3,
      percent: 91.5,
      tier: "Common",
    }),
    conquista({
      id: "ACH_ROGUE_2",
      name: "Herdeiro persistente",
      description: "Chegue ao segundo chefe sem usar o elevador.",
      unlocked: true,
      unlock_time: Math.floor(Date.now() / 1000) - 7200,
      percent: 12.25,
      tier: "Rare",
    }),
    conquista({
      id: "ACH_ROGUE_3",
      name: "Segredo da família",
      description: "Encontre o diário escondido na biblioteca.",
      hidden: true,
      percent: 0.8,
      tier: "UltraRare",
    }),
  ];
  const unlocked = achievements.filter((a) => a.unlocked).length;
  return {
    set: { achievements, progress_known: true },
    summary: {
      unlocked,
      total: achievements.length,
      percent: Math.floor((unlocked * 100) / achievements.length),
      complete: unlocked === achievements.length,
      by_tier: { Comum: 1, Rara: 1 },
    },
    from_cache: false,
    fetched_at: Math.floor(Date.now() / 1000),
    notice: null,
  };
};

const conquistasHub = (): ConquistasHub => {
  const horas = Math.floor(jogos.reduce((s, j) => s + j.playtime_secs, 0) / 3600);
  const sequencia = 12;
  const melhor = 18;
  const completos = 1;
  const marcos: Array<[string, string, string, MarcoHub["nivel"], number, string, number]> = [
    ["primeira-hora", "Primeira hora", "Jogue por 1 hora no total.", "bronze", 1, "h", horas],
    ["dez-horas", "Dez horas de jogo", "Jogue por 10 horas no total.", "prata", 10, "h", horas],
    ["cinquenta-horas", "Cinquenta horas", "Jogue por 50 horas no total.", "ouro", 50, "h", horas],
    ["duzentas-horas", "Duzentas horas", "Jogue por 200 horas no total.", "ouro", 200, "h", horas],
    ["primeiro-jogo", "Primeiro jogo", "Adicione 1 jogo à biblioteca.", "bronze", 1, "jogo", jogos.length],
    ["cinco-jogos", "Coleção começando", "Adicione 5 jogos à biblioteca.", "prata", 5, "jogos", jogos.length],
    ["vinte-jogos", "Biblioteca cheia", "Adicione 20 jogos à biblioteca.", "ouro", 20, "jogos", jogos.length],
    ["sequencia-5", "Ritmo constante", "Conclua 5 sincronizações seguidas sem erro.", "bronze", 5, "syncs", sequencia],
    ["sequencia-25", "Tudo em dia", "Conclua 25 sincronizações seguidas sem erro.", "prata", 25, "syncs", sequencia],
    ["sequencia-100", "Sem falhar", "Conclua 100 sincronizações seguidas sem erro.", "ouro", 100, "syncs", sequencia],
    ["primeiro-completo", "Jogo completo", "Desbloqueie todas as conquistas da Steam de um jogo.", "prata", 1, "jogo", completos],
    ["cinco-completos", "Caçador de conquistas", "Complete as conquistas da Steam de 5 jogos.", "ouro", 5, "jogos", completos],
  ];
  const conquistas: MarcoHub[] = marcos
    .map(([id, nome, descricao, nivel, alvo, unidade, bruto]) => ({
      id,
      nome,
      descricao,
      nivel,
      progresso: Math.min(bruto, alvo),
      alvo,
      unidade,
      desbloqueada: bruto >= alvo,
    }))
    .sort((a, b) => Number(b.desbloqueada) - Number(a.desbloqueada) || a.alvo - b.alvo);
  return {
    conquistas,
    desbloqueadas: conquistas.filter((c) => c.desbloqueada).length,
    total: conquistas.length,
    horas,
    jogos: jogos.length,
    sequencia_atual: sequencia,
    melhor_sequencia: melhor,
    jogos_completos: completos,
  };
};

let jogos: Game[] =
  cenario === "vazio" || cenario === "primeiro-uso"
    ? []
    : cenario === "preview"
      ? [
          jogo(1, "Rogue Legacy", { playtime_secs: 41000 }),
          jogo(2, "Hollow Knight", { playtime_secs: 93000 }),
          jogo(3, "Celeste", { playtime_secs: 25200, last_sync_at: Math.floor(Date.now() / 1000) - 86400 }),
          jogo(4, "Hades", { playtime_secs: 61200 }),
          jogo(5, "Animal Well", { playtime_secs: 78700, last_played_at: Math.floor(Date.now() / 1000) - 600 }),
          // Sea of Stars foi adicionado e nunca aberto pelo app: fica sem sync
          // e nunca jogado, para exercitar os filtros da biblioteca.
          jogo(6, "Sea of Stars", { playtime_secs: 0, last_sync_at: null, last_played_at: null }),
        ]
      : [
        jogo(1, "Rogue Legacy", { playtime_secs: 41000 }),
        jogo(2, "Hollow Knight: Edição Especial com um Nome Muito Longo para Testar a Quebra de Linha"),
        jogo(3, "Celeste", { last_sync_at: null, last_played_at: null, playtime_secs: 0 }),
        jogo(4, "SupercalifragilisticexpialidocioustextWithoutSpacesAtAllJustToBreakTheLayout", { playtime_secs: 59 }),
      ];

const espera = (ms = 120) => new Promise((r) => setTimeout(r, ms));

export async function mock<T>(cmd: string, args: Record<string, unknown> = {}): Promise<T> {
  await espera();
  const r = (v: unknown) => v as T;
  switch (cmd) {
    case "sobre":
      return r({ nome: "Sync Game Save", versao: "2.0.1", build: "local (teste)" });
    case "estado":
      return r(estado);
    case "verificar_pasta_sync":
    case "definir_pasta_sync": {
      const pasta = String(args.pasta);
      if (pasta.includes("invalida")) throw "Esta pasta não existe.";
      if (cmd === "definir_pasta_sync") estado = { ...estado, pasta_sync: pasta };
      return r({ exists: true, writable: true, has_data: pasta.includes("outro-pc"), games: pasta.includes("outro-pc") ? ["Celeste", "Hades"] : [], message: null });
    }
    case "definir_nome_pc":
      estado = { ...estado, nome_pc: String(args.nome) };
      return r(undefined);
    case "atualizar_catalogo":
      estado = { ...estado, catalogo_pronto: true };
      return r(45612);
    case "reconhecer_pasta": {
      const rec: Reconhecimento = {
        name: "Rogue Legacy",
        catalog_id: 13676,
        confidence: "High",
        exe: String(args.pasta) + "/RogueLegacy.exe",
        save_candidates: [
          { logical: "<winDocuments>/SavedGames/RogueLegacy", resolved: "C:/Users/voce/Documents/SavedGames/RogueLegacy", exists: true, files: 10, selected: true },
          { logical: "<winAppData>/Rogue Legacy", resolved: "C:/Users/voce/AppData/Roaming/Rogue Legacy", exists: false, files: 0, selected: false },
        ],
        alternatives: [],
      };
      return r(rec);
    }
    case "varrer_pasta": {
      const rec: Reconhecimento = { name: "Jogo encontrado", catalog_id: 12, confidence: "High", exe: "C:/Games/Encontrado/game.exe", save_candidates: [{ logical: "<winDocuments>/Save", resolved: "C:/Save", exists: true, files: 3, selected: true }], alternatives: [{ id: 13, name: "Outro candidato" }] };
      return r({ recognized: [["C:/Games/Encontrado", rec]], needs_confirmation: [], unknown: [["C:/Games/Outro", { ...rec, name: "Outro jogo", catalog_id: null, confidence: "None", save_candidates: [], alternatives: [] }]], errors: [["C:/Games/Protegido", "Acesso negado"]] });
    }
    case "escolher_candidato_catalogo":
      return r({ name: args.id === 13 ? "Outro candidato" : "Jogo encontrado", catalog_id: args.id, confidence: "High", exe: "C:/Games/Encontrado/game.exe", save_candidates: [{ logical: "<base>/Saves", resolved: "C:/Games/Encontrado/Saves", exists: true, files: 1, selected: true }], alternatives: [] });
    case "converter_caminho":
      return r(String(args.caminho));
    case "descobrir_save":
      if (cenario === "descoberta-enrolada") {
        // Uma descoberta de verdade só termina quando o usuário fecha o jogo.
        await new Promise<void>(() => {});
      }
      return r(0);
    case "candidatos_save": {
      if (cenario === "sem-descoberta") return r([]);
      return r([
        { logical: "<winDocuments>/SavedGames/RogueLegacy", resolved: "C:/Users/voce/Documents/SavedGames/RogueLegacy", added: 3, removed: 0, bytes: 2048 },
        { logical: "<winAppData>/Rogue Legacy/Saves", resolved: "C:/Users/voce/AppData/Roaming/Rogue Legacy/Saves", added: 1, removed: 0, bytes: 128 },
      ]);
    }
    case "listar_jogos":
      return r(jogos);
    case "abrir_pasta_de_logs":
    case "sair_app":
      return r(undefined);
    case "primeiro_backup": {
      await espera(200);
      const g = jogos.find((j) => j.id === args.id);
      if (cenario === "backup-ocupado" && !backupLiberado) { backupLiberado = true; return r({ kind: "locked", pc: "PC-QUARTO" }); }
      return r(g?.name === "Outro jogo" ? { kind: "needs_decision" } : { kind: "created" });
    }
    case "adicionar_jogo": {
      const g = args.jogo as Game;
      if (jogos.some((j) => j.name.toLowerCase() === g.name.toLowerCase())) throw "Já existe um jogo com este nome.";
      const id = Math.max(0, ...jogos.map((j) => j.id)) + 1;
      jogos = [...jogos, jogo(id, g.name, { ...g, last_sync_at: null, last_played_at: null, playtime_secs: 0, base_digest: null })];
      return r(id);
    }
    case "editar_jogo": {
      const novo = args.jogo as Pick<Game, "name" | "exe" | "install_dir" | "catalog_id" | "save_paths">;
      if (jogos.some((j) => j.id !== args.id && j.name.toLowerCase() === novo.name.toLowerCase())) throw "Já existe um jogo com este nome.";
      jogos = jogos.map((j) => j.id === args.id ? { ...j, ...novo } : j);
      return r(undefined);
    }
    case "remover_jogo":
      jogos = jogos.filter((j) => j.id !== args.id);
      return r(undefined);
    case "parar_jogo":
      return r(true);
    case "forcar_parada_jogo":
      terminarJogo?.();
      terminarJogo = undefined;
      return r(true);
    case "jogar": {
      const o = (args.opcoes ?? {}) as { conflito?: string; forcar?: boolean };
      if (cenario === "conflito" && !o.conflito) return r({ kind: "conflict" });
      if (args.id === 2 && !o.forcar) return r({ kind: "locked", pc: "PC-QUARTO" });
      if (cenario === "parada") {
        window.dispatchEvent(new CustomEvent("sgs:mock-jogo", { detail: { id: args.id, estado: "jogando" } }));
        await new Promise<void>((resolve) => { terminarJogo = resolve; });
      } else await espera(600);
      return r({ kind: "finished", exit_code: 0, synced: true, notice: "Save enviado para a pasta de sincronização." });
    }
    case "capa_do_jogo":
      return r(null);
    case "arte_do_jogo": {
      // Arte pública da Steam só para dar realismo às capturas; jogos sem id ficam sem arte.
      const nome = jogos.find((j) => j.id === args.id)?.name ?? "";
      const steam = Object.entries(STEAM).find(([n]) => nome.startsWith(n))?.[1];
      const arquivo = { header: "header.jpg", hero: "library_hero.jpg", logo: "logo.png", cover: "library_600x900.jpg" }[args.tipo as string];
      return r(steam && arquivo ? `https://cdn.cloudflare.steamstatic.com/steam/apps/${steam}/${arquivo}` : null);
    }
    case "versoes":
      return r(["1793920203_PC-SALA", "1793919686_PC-SALA", "1793900000_PC-QUARTO"]);
    case "restaurar_versao":
      return r("C:/Users/voce/AppData/Local/sync-game-save/state/1/backups/1793920300_restaurar");
    case "conquistas":
      return r(conquistasDoJogo());
    case "definir_credenciais_steam": {
      const chave = String(args.chave).trim();
      const steamId = String(args.steamId).trim();
      estado = { ...estado, steam_configurada: chave !== "" && steamId !== "", steam_id: steamId || "76561198000000000" };
      return r(undefined);
    }
    case "definir_chave_steamgriddb": {
      const chave = String(args.chave).trim();
      estado = { ...estado, steamgriddb_configurada: chave !== "" };
      return r(undefined);
    }
    case "conquistas_hub":
      return r(conquistasHub());
    case "temas":
      return r({ atual: temaAtual, temas });
    case "definir_tema": {
      const id = String(args.id);
      if (!temas.some((t) => t.id === id)) throw "Este tema não está instalado. Abra as configurações e escolha outro.";
      temaAtual = id;
      localStorage.setItem(CHAVE_TEMA, id);
      return r({ atual: temaAtual, temas });
    }
    default:
      throw `Comando desconhecido: ${cmd}`;
  }
}
