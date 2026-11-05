// Dados simulados para rodar a interface no navegador (testes E2E e capturas de tela).
// Cenário pela URL: ?cenario=primeiro-uso | vazio | jogos (padrão) | conflito
import type { Game, Estado, Reconhecimento } from "./api";

const cenario = new URLSearchParams(location.search).get("cenario") ?? "jogos";

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
      return r({ nome: "Sync Game Save", versao: "0.1.0" });
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
    case "converter_caminho":
      return r(String(args.caminho));
    case "listar_jogos":
      return r(jogos);
    case "adicionar_jogo": {
      const g = args.jogo as Game;
      if (jogos.some((j) => j.name.toLowerCase() === g.name.toLowerCase())) throw "Já existe um jogo com este nome.";
      const id = Math.max(0, ...jogos.map((j) => j.id)) + 1;
      jogos = [...jogos, jogo(id, g.name, { ...g, last_sync_at: null, last_played_at: null, playtime_secs: 0, base_digest: null })];
      return r(id);
    }
    case "remover_jogo":
      jogos = jogos.filter((j) => j.id !== args.id);
      return r(undefined);
    case "jogar": {
      const o = (args.opcoes ?? {}) as { conflito?: string; forcar?: boolean };
      if (cenario === "conflito" && !o.conflito) return r({ kind: "conflict" });
      if (args.id === 2 && !o.forcar) return r({ kind: "locked", pc: "PC-QUARTO" });
      await espera(600);
      return r({ kind: "finished", exit_code: 0, synced: true, notice: "Save enviado para a pasta de sincronização." });
    }
    case "capa_do_jogo":
      return r(null);
    case "versoes":
      return r(["1793920203_PC-SALA", "1793919686_PC-SALA", "1793900000_PC-QUARTO"]);
    case "restaurar_versao":
      return r("C:/Users/voce/AppData/Local/sync-game-save/state/1/backups/1793920300_restaurar");
    default:
      throw `Comando desconhecido: ${cmd}`;
  }
}
