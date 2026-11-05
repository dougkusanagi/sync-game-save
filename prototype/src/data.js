// Dados fictícios do protótipo. Os jogos reais usam a arte pública da Steam
// só para dar realismo à tela; nomes de PCs, saves e conquistas são inventados.

const HOUR = 3600e3;
const DAY = 24 * HOUR;
const now = Date.now();

export const steamArt = (appid, file) =>
	`https://cdn.cloudflare.steamstatic.com/steam/apps/${appid}/${file}`;

const raw = [
	{ id: "animal-well", appid: 813230, name: "Animal Well", dev: "Shared Memory", year: 2024, tags: ["Metroidvania", "Quebra-cabeça", "Pixel art"], desc: "Explore um labirinto vivo e cheio de segredos, onde cada animal esconde uma mecânica nova.", accent: "#f2c14e", playtime: 1312, lastPlayed: 2 * HOUR, added: 40 * DAY, source: "Steam", sync: "ok", ach: [24, 33], fav: true },
	{ id: "hollow-knight", appid: 367520, name: "Hollow Knight", dev: "Team Cherry", year: 2017, tags: ["Metroidvania", "Ação", "Difícil"], desc: "Desça a um reino de insetos em ruínas e enfrente criaturas, chefes e segredos de Hallownest.", accent: "#8fb3d9", playtime: 3980, lastPlayed: 20 * HOUR, added: 300 * DAY, source: "GOG", sync: "conflict", ach: [41, 63], fav: true },
	{ id: "celeste", appid: 504230, name: "Celeste", dev: "Maddy Makes Games", year: 2018, tags: ["Plataforma", "Difícil", "Narrativa"], desc: "Ajude Madeline a escalar a montanha Celeste, um desafio de plataforma preciso e cheio de coração.", accent: "#e05a8a", playtime: 1860, lastPlayed: 2 * DAY, added: 200 * DAY, source: "itch.io", sync: "cloud-newer", ach: [18, 32] },
	{ id: "hades", appid: 1145360, name: "Hades", dev: "Supergiant Games", year: 2020, tags: ["Roguelike", "Ação", "Mitologia"], desc: "Lute para escapar do Submundo como Zagreus, com a ajuda dos deuses do Olimpo.", accent: "#e4572e", playtime: 5120, lastPlayed: 3 * DAY, added: 500 * DAY, source: "Steam", sync: "local-newer", ach: [44, 49], fav: true },
	{ id: "sea-of-stars", appid: 1244090, name: "Sea of Stars", dev: "Sabotage Studio", year: 2023, tags: ["RPG", "Turnos", "Pixel art"], desc: "Um RPG por turnos com combate em tempo certo, inspirado nos clássicos dos anos 90.", accent: "#f08c3c", playtime: 960, lastPlayed: 4 * DAY, added: 2 * DAY, source: "GOG", sync: "cloud-syncing", ach: [12, 46], isNew: true },
	{ id: "aggelos", appid: 717310, name: "Aggelos", dev: "Storybird", year: 2018, tags: ["Ação", "Aventura", "Retrô"], desc: "Uma aventura de ação no estilo 16 bits para salvar o reino e a princesa Lisa.", accent: "#3fa9f5", playtime: 240, lastPlayed: 6 * DAY, added: 1 * DAY, source: "Outros", sync: "ok", ach: null, isNew: true },
	{ id: "transistor", appid: 237930, name: "Transistor", dev: "Supergiant Games", year: 2014, tags: ["RPG de ação", "Ficção científica", "Estratégia"], desc: "Empunhe uma arma misteriosa e lute por uma cidade futurista tomada por uma força robótica.", accent: "#3cc8c8", playtime: 610, lastPlayed: 9 * DAY, added: 3 * DAY, source: "Steam", sync: "ok", ach: [20, 33], isNew: true },
	{ id: "dead-cells", appid: 588650, name: "Dead Cells", dev: "Motion Twin", year: 2018, tags: ["Roguelite", "Metroidvania", "Ação"], desc: "Explore uma ilha em mudança constante, morra, aprenda e volte mais forte.", accent: "#4fd1c5", playtime: 2210, lastPlayed: 12 * DAY, added: 420 * DAY, source: "Steam", sync: "ok", ach: [37, 59] },
	{ id: "stardew", appid: 413150, name: "Stardew Valley", dev: "ConcernedApe", year: 2016, tags: ["Simulação", "Fazenda", "Relaxante"], desc: "Herde a fazenda do avô e construa uma vida nova no vale.", accent: "#6bbf59", playtime: 7400, lastPlayed: 15 * DAY, added: 700 * DAY, source: "GOG", sync: "ok", ach: [29, 40], fav: true },
	{ id: "ori-wotw", appid: 1057090, name: "Ori and the Will of the Wisps", dev: "Moon Studios", year: 2020, tags: ["Plataforma", "Aventura", "Metroidvania"], desc: "Uma jornada emocionante para descobrir o destino de Ori numa floresta em perigo.", accent: "#58c4f0", playtime: 820, lastPlayed: 20 * DAY, added: 260 * DAY, source: "Steam", sync: "ok", ach: [31, 46] },
	{ id: "tunic", appid: 553420, name: "Tunic", dev: "TUNIC Team", year: 2022, tags: ["Aventura", "Ação", "Exploração"], desc: "Uma pequena raposa numa terra de lendas, guiada por um manual cheio de segredos.", accent: "#f0b03c", playtime: 700, lastPlayed: 28 * DAY, added: 150 * DAY, source: "Steam", sync: "ok", ach: [10, 41] },
	{ id: "katana-zero", appid: 460950, name: "Katana ZERO", dev: "Askiisoft", year: 2019, tags: ["Ação", "Neo-noir", "Pixel art"], desc: "Um assassino com espada e controle do tempo, numa história neo-noir cheia de reviravoltas.", accent: "#b46cf0", playtime: 330, lastPlayed: 35 * DAY, added: 90 * DAY, source: "GOG", sync: "ok", ach: [14, 22] },
	{ id: "blasphemous", appid: 774361, name: "Blasphemous", dev: "The Game Kitchen", year: 2019, tags: ["Metroidvania", "Ação", "Sombrio"], desc: "Lute pela terra amaldiçoada de Cvstodia como o Penitente.", accent: "#c9a227", playtime: 1120, lastPlayed: 50 * DAY, added: 380 * DAY, source: "Steam", sync: "ok", ach: [26, 46] },
	{ id: "hyper-light", appid: 257850, name: "Hyper Light Drifter", dev: "Heart Machine", year: 2016, tags: ["Ação", "Exploração", "Pixel art"], desc: "Explore um mundo vasto e arruinado, com combate rápido e uma história sem palavras.", accent: "#e04f9a", playtime: 540, lastPlayed: 80 * DAY, added: 600 * DAY, source: "itch.io", sync: "ok", ach: [8, 30] },
	{ id: "omori", appid: 1150690, name: "OMORI", dev: "OMOCAT", year: 2020, tags: ["RPG", "Psicológico", "Narrativa"], desc: "Explore um mundo estranho de memórias e sonhos e enfrente seus medos.", accent: "#9ab0f5", playtime: 1500, lastPlayed: 120 * DAY, added: 640 * DAY, source: "Steam", sync: "ok", ach: [30, 52] },
	{ id: "terraria", appid: 105600, name: "Terraria", dev: "Re-Logic", year: 2011, tags: ["Sandbox", "Aventura", "Construção"], desc: "Cave, lute, explore e construa num mundo 2D gerado ao acaso.", accent: "#6abf4b", playtime: 9100, lastPlayed: 160 * DAY, added: 1200 * DAY, source: "Steam", sync: "off", ach: [60, 115] },
	{ id: "undertale", appid: 391540, name: "Undertale", dev: "tobyfox", year: 2015, tags: ["RPG", "Escolhas", "Humor"], desc: "Um RPG em que ninguém precisa morrer, num mundo subterrâneo de monstros.", accent: "#f5d742", playtime: 480, lastPlayed: 300 * DAY, added: 900 * DAY, source: "GOG", sync: "ok", ach: null },
	{ id: "cartucho", appid: null, name: "Aventura de Cartucho (SNES)", dev: "Emulador · Snes9x", year: 1994, tags: ["Emulador", "Plataforma"], desc: "Exemplo de jogo de emulador: o save fica por ROM, na pasta de saves do emulador.", accent: "#9b7bff", playtime: 95, lastPlayed: 45 * DAY, added: 4 * DAY, source: "Emulador", sync: "ok", ach: [3, 25], isNew: true },
	{ id: "cronicas", appid: null, name: "Crônicas da Fortaleza Esquecida: Edição Definitiva de Aniversário com Todas as Expansões", dev: "Estúdio Independente Muito Pequeno", year: 2021, tags: ["RPG", "Sem arte"], desc: "Exemplo de jogo sem arte e com nome muito longo, para testar quebra de texto e o espaço reservado.", accent: "#5f8cff", playtime: 0, lastPlayed: null, added: 10 * DAY, source: "Outros", sync: "off", ach: null },
];

const DEVICES = ["PC-SALA", "NOTEBOOK"];

function makeVersions(g, i) {
	const count = g.sync === "off" ? 0 : 4 + (i % 5);
	const list = [];
	let t = now - (g.lastPlayed ?? 30 * DAY);
	for (let n = 0; n < count; n++) {
		list.push({
			id: `${g.id}-v${n}`,
			at: t,
			device: DEVICES[(n + i) % 2],
			reason: n === 2 ? "Antes de restaurar" : "Ao fechar o jogo",
			size: 18_000 + ((i * 7919 + n * 3301) % 900_000),
			files: 1 + ((i + n) % 7),
		});
		t -= (1 + ((n * 3 + i) % 5)) * DAY;
	}
	return list;
}

export const games = raw.map((g, i) => ({
	...g,
	lastPlayed: g.lastPlayed == null ? null : now - g.lastPlayed,
	added: now - g.added,
	art: g.appid
		? {
				hero: steamArt(g.appid, "library_hero.jpg"),
				logo: steamArt(g.appid, "logo.png"),
				cover: steamArt(g.appid, "library_600x900.jpg"),
				header: steamArt(g.appid, "header.jpg"),
			}
		: null,
	versions: makeVersions(g, i),
	installPath:
		g.source === "Emulador"
			? "D:\\Emuladores\\snes9x\\roms\\Aventura de Cartucho.sfc"
			: `C:\\Games\\${g.name.replace(/[:]/g, "")}`,
	savePaths:
		g.source === "Emulador"
			? [{ path: "<base>/Saves/Aventura de Cartucho.srm", found: true, files: 1, size: 8192 }]
			: [
					{ path: `<winAppData>/${g.dev}/${g.name}/saves`, found: g.sync !== "off", files: 3 + (i % 4), size: 120_000 + i * 9000 },
					...(i % 3 === 0 ? [{ path: "<base>/settings.ini", found: i % 2 === 0, files: 1, size: 2048 }] : []),
				],
}));

const ACH_NAMES = [
	"Primeiro passo", "Sem olhar para trás", "Colecionador", "Mestre do combate", "Explorador",
	"Sobrevivente", "Velocista", "Pacifista", "Perfeccionista", "Caçador de segredos",
	"Lenda", "Persistente", "Sem dano", "Cartógrafo", "Até o fim",
];

export function achievementsFor(g) {
	if (!g.ach) return [];
	const [done, total] = g.ach;
	return Array.from({ length: Math.min(total, 15) }, (_, n) => ({
		name: ACH_NAMES[n % ACH_NAMES.length],
		desc: "Descrição da conquista vinda do provedor (ex.: Steam Web API).",
		unlocked: n < Math.round((done / total) * Math.min(total, 15)),
		at: now - (n + 1) * 3 * DAY,
	}));
}

// Amostra do catálogo (no app final: manifesto do Ludusavi indexado em SQLite FTS5).
export const catalog = [
	{ name: "Hollow Knight: Silksong", saves: 2 },
	{ name: "Cuphead", saves: 1 },
	{ name: "Vampire Survivors", saves: 1 },
	{ name: "Risk of Rain 2", saves: 2 },
	{ name: "Celeste 64: Fragments of the Mountain", saves: 1 },
	{ name: "Hades II", saves: 1 },
	{ name: "Dead Cells: Return to Castlevania", saves: 1 },
];
