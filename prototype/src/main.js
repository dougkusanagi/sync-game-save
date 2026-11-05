import "./tokens.css";
import "./styles.css";
import { achievementsFor, catalog, games } from "./data.js";
import * as F from "./focus.js";
import { icon } from "./icons.js";
import { GLYPHS, initInput } from "./input.js";

// ---------------------------------------------------------------------------
// Utilidades
// ---------------------------------------------------------------------------
const $ = (s, r = document) => r.querySelector(s);
const $$ = (s, r = document) => [...r.querySelectorAll(s)];
const esc = (s) =>
	String(s ?? "").replace(/[&<>"']/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[c]);
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
const root = document.documentElement;
const app = $("#app");

const rtf = new Intl.RelativeTimeFormat("pt-BR", { numeric: "auto" });
const dtf = new Intl.DateTimeFormat("pt-BR", { dateStyle: "short", timeStyle: "short" });
const dateF = new Intl.DateTimeFormat("pt-BR", { dateStyle: "medium" });
const timeF = new Intl.DateTimeFormat("pt-BR", { hour: "2-digit", minute: "2-digit" });
const nf = new Intl.NumberFormat("pt-BR", { maximumFractionDigits: 1 });

function ago(ts) {
	if (!ts) return "nunca";
	const s = (ts - Date.now()) / 1000;
	const a = Math.abs(s);
	if (a < 60) return "agora mesmo";
	if (a < 3600) return rtf.format(Math.round(s / 60), "minute");
	if (a < 86400) return rtf.format(Math.round(s / 3600), "hour");
	if (a < 86400 * 30) return rtf.format(Math.round(s / 86400), "day");
	if (a < 86400 * 365) return rtf.format(Math.round(s / 86400 / 30), "month");
	return rtf.format(Math.round(s / 86400 / 365), "year");
}
const size = (b) => (b < 1024 ? `${b} B` : b < 1048576 ? `${nf.format(b / 1024)} KB` : `${nf.format(b / 1048576)} MB`);
function playtime(min) {
	if (!min) return "Nunca jogado";
	const h = Math.floor(min / 60);
	return h ? `${h} h ${min % 60} min` : `${min} min`;
}
const norm = (s) => s.normalize("NFD").replace(/\p{Diacritic}/gu, "").toLowerCase();

// ---------------------------------------------------------------------------
// Estado
// ---------------------------------------------------------------------------
const state = {
	screen: "home",
	shelf: 0,
	libTab: 0,
	filters: { status: new Set(), source: new Set(), sort: "recent" },
	settings: "nuvem",
	device: "keyboard",
	glyphPref: "auto",
	sounds: true,
	accent: "game",
	cloudOffline: false,
	emptyLibrary: false,
	hero: null,
	pcName: "PC-SALA",
	cloudPath: "C:\\Users\\Doug\\OneDrive\\SyncGameSave",
	retention: 20,
	notifications: [
		{ at: Date.now() - 2 * 3600e3, text: "Save de Animal Well enviado para a nuvem.", tone: "ok" },
		{ at: Date.now() - 20 * 3600e3, text: "Hollow Knight: conflito de save entre PC-SALA e NOTEBOOK.", tone: "danger" },
		{ at: Date.now() - 3 * 86400e3, text: "Catálogo de jogos atualizado: 34.812 jogos conhecidos.", tone: "info" },
	],
};

const SYNC = {
	ok: { label: "Sincronizado", icon: "cloudCheck", tone: "ok" },
	"cloud-newer": { label: "Save mais novo na nuvem", icon: "cloudDown", tone: "info" },
	"local-newer": { label: "Save deste PC mais novo", icon: "cloudUp", tone: "info" },
	conflict: { label: "Conflito de save", icon: "alert", tone: "danger" },
	"cloud-syncing": { label: "Nuvem ainda baixando", icon: "cloudSync", tone: "warn" },
	off: { label: "Sync desligado", icon: "cloudOff", tone: "dim" },
};
const OFFLINE = { label: "Nuvem indisponível", icon: "cloudOff", tone: "warn" };
const syncInfo = (g) => (state.cloudOffline && g.sync !== "off" ? OFFLINE : SYNC[g.sync]);

function syncDetail(g) {
	if (state.cloudOffline && g.sync !== "off")
		return "A pasta da nuvem não foi encontrada. Você ainda pode jogar; o save será enviado quando a nuvem voltar.";
	const v = g.versions[0];
	switch (g.sync) {
		case "ok":
			return v ? `Última versão ${ago(v.at)}, de ${v.device}.` : "Pronto para o primeiro backup.";
		case "cloud-newer":
			return "O NOTEBOOK enviou um save mais novo. Ele será baixado quando você jogar.";
		case "local-newer":
			return "Este PC tem um save mais novo. Ele será enviado quando você jogar ou sincronizar.";
		case "conflict":
			return "Os saves mudaram nos dois PCs. Você escolhe qual usar quando jogar.";
		case "cloud-syncing":
			return "O OneDrive ainda está baixando o save deste jogo.";
		default:
			return "O save deste jogo não é copiado para a nuvem.";
	}
}

const lib = () => (state.emptyLibrary ? [] : games.filter((g) => !g.hidden));
const byId = (id) => games.find((g) => g.id === id);
const needsAttention = (g) =>
	g.sync === "conflict" || g.sync === "cloud-syncing" || (g.sync !== "off" && g.savePaths.some((p) => !p.found));

const SORTS = {
	recent: { label: "Último jogado", fn: (a, b) => (b.lastPlayed ?? 0) - (a.lastPlayed ?? 0) },
	name: { label: "Nome", fn: (a, b) => a.name.localeCompare(b.name, "pt-BR") },
	playtime: { label: "Tempo de jogo", fn: (a, b) => b.playtime - a.playtime },
	added: { label: "Adicionado", fn: (a, b) => b.added - a.added },
};
const sortGames = (list, key) => [...list].sort(SORTS[key].fn);

const SHELVES = [
	{ id: "recent", title: "Jogados recentemente", list: () => sortGames(lib(), "recent") },
	{ id: "fav", title: "Favoritos", list: () => sortGames(lib().filter((g) => g.fav), "recent"), empty: "Nenhum favorito ainda. Abra as opções de um jogo e escolha “Favoritar”." },
	{ id: "added", title: "Adicionados", list: () => sortGames(lib(), "added") },
	{ id: "attention", title: "Precisam de atenção", list: () => lib().filter(needsAttention), empty: "Tudo certo: nenhum save precisa da sua atenção." },
];

const LIB_TABS = [
	{ id: "all", label: "Todos", f: () => true },
	{ id: "fav", label: "Favoritos", f: (g) => g.fav },
	{ id: "sync", label: "Com sync", f: (g) => g.sync !== "off" },
	{ id: "attention", label: "Precisam de atenção", f: needsAttention },
	{ id: "emu", label: "Emuladores", f: (g) => g.source === "Emulador" },
];
const SOURCES = ["Steam", "GOG", "itch.io", "Emulador", "Outros"];

// ---------------------------------------------------------------------------
// Som de navegação (WebAudio, sem arquivos)
// ---------------------------------------------------------------------------
let actx;
function sound(kind) {
	if (!state.sounds) return;
	try {
		actx ??= new AudioContext();
		const t = actx.currentTime;
		const o = actx.createOscillator();
		const g = actx.createGain();
		o.type = "sine";
		o.frequency.value = { move: 1180, confirm: 760, back: 520, open: 640, error: 240 }[kind] ?? 700;
		g.gain.setValueAtTime(0.0001, t);
		g.gain.exponentialRampToValueAtTime(kind === "move" ? 0.018 : 0.03, t + 0.008);
		g.gain.exponentialRampToValueAtTime(0.0001, t + 0.09);
		o.connect(g).connect(actx.destination);
		o.start(t);
		o.stop(t + 0.1);
	} catch {
		/* áudio indisponível: segue sem som */
	}
}

// ---------------------------------------------------------------------------
// Arte e espaços reservados
// ---------------------------------------------------------------------------
function initials(name) {
	return name
		.split(/\s+/)
		.filter((w) => /^[\p{L}\d]/u.test(w) && w.length > 2)
		.slice(0, 2)
		.map((w) => w[0])
		.join("")
		.toUpperCase();
}
const placeholder = (g) => `<span class="ph" style="--ph:${g.accent}"><span>${esc(initials(g.name))}</span></span>`;
const art = (g, kind) =>
	g.art ? `<img src="${g.art[kind]}" alt="" loading="lazy" decoding="async" data-fallback="ph">` : placeholder(g);

// Imagem que falhou vira espaço reservado (ou título em texto, no caso do logo).
document.addEventListener(
	"error",
	(e) => {
		const img = e.target;
		if (!(img instanceof HTMLImageElement)) return;
		if (img.dataset.fallback === "title") {
			img.nextElementSibling?.classList.remove("sr-only");
			img.remove();
		} else if (img.dataset.fallback === "ph") {
			const g = byId(img.closest("[data-game]")?.dataset.game ?? img.closest("[data-game-ctx]")?.dataset.gameCtx);
			if (g) img.outerHTML = placeholder(g);
		}
	},
	true,
);

// ---------------------------------------------------------------------------
// Legenda de botões
// ---------------------------------------------------------------------------
const effectiveDevice = () => (state.glyphPref === "auto" ? state.device : state.glyphPref);
function glyph(action, device = effectiveDevice()) {
	const label = GLYPHS[device]?.[action];
	if (!label) return "";
	const face = ["confirm", "back", "filter", "options"].includes(action) && device !== "keyboard";
	return `<kbd class="glyph ${face ? "face" : ""} g-${action}">${esc(label)}</kbd>`;
}

function legendItems() {
	const top = F.topLayer();
	const cur = F.current();
	if (top) {
		const m = meta.get(top);
		if (typeof m.legend === "function") return m.legend(cur);
		return m.legend ?? [["confirm", "Selecionar"], ["back", "Voltar"]];
	}
	const common = [["search", "Buscar"], ["menu", "Menu"], ["help", "Atalhos"]];
	if (state.screen === "home") {
		if (cur?.dataset.game) return [["confirm", "Jogar"], ["options", "Opções"], ["details", "Saves"], ["prevTab", ""], ["nextTab", "Seção"], ["filter", "Filtros"], ...common];
		return [["confirm", "Selecionar"], ["prevTab", ""], ["nextTab", "Seção"], ["filter", "Filtros"], ...common];
	}
	if (state.screen === "library")
		return [["confirm", cur?.dataset.game ? "Detalhes" : "Selecionar"], ["options", "Opções"], ["prevTab", ""], ["nextTab", "Aba"], ["filter", "Filtros"], ["back", "Início"], ...common];
	return [["confirm", "Selecionar"], ["prevTab", ""], ["nextTab", "Seção"], ["back", "Início"], ...common];
}

function updateLegend() {
	const items = legendItems().filter(([a]) => GLYPHS[effectiveDevice()]?.[a]);
	// "Seção anterior/próxima" aparece como um item só: [Q][E] Seção.
	const html = [];
	for (let i = 0; i < items.length; i++) {
		const [a, label] = items[i];
		if (a === "prevTab" && items[i + 1]?.[0] === "nextTab") {
			html.push(`<button class="legend-item" data-action="nextTab" tabindex="-1">${glyph("prevTab")}${glyph("nextTab")}<span>${esc(items[i + 1][1])}</span></button>`);
			i++;
			continue;
		}
		html.push(`<button class="legend-item" data-action="${a}" tabindex="-1">${glyph(a)}${label ? `<span>${esc(label)}</span>` : ""}</button>`);
	}
	$("#legend").innerHTML = html.join("");
}

// ---------------------------------------------------------------------------
// Overlays
// ---------------------------------------------------------------------------
const meta = new WeakMap();

function syncInert() {
	const open = $$("#overlays > .overlay:not(.closing)");
	const top = open.at(-1);
	$("#base").inert = !!top;
	for (const o of open) o.inert = o !== top;
}

function openOverlay({ kind, label, html, blocking = false, onBack, onTab, legend, game }) {
	const el = document.createElement("div");
	el.className = `overlay ov-${kind}`;
	el.setAttribute("role", "dialog");
	el.setAttribute("aria-modal", "true");
	el.setAttribute("aria-label", label);
	if (game) el.dataset.gameCtx = game.id;
	el.innerHTML = `<div class="ov-backdrop"></div>${html}`;
	$("#overlays").append(el);
	meta.set(el, { kind, blocking, onBack, onTab, legend });
	el.querySelector(".ov-backdrop").addEventListener("click", () => F.topLayer() === el && back());
	syncInert();
	F.pushLayer(el);
	requestAnimationFrame(() => el.classList.add("is-open"));
	sound("open");
	updateLegend();
	return el;
}

function closeOverlay(el = F.topLayer()) {
	if (!el || F.topLayer() !== el) return;
	el.classList.add("closing");
	el.classList.remove("is-open");
	syncInert();
	F.popLayer();
	setTimeout(() => el.remove(), 260);
	updateLegend();
}
const closeAll = () => {
	while (F.topLayer()) closeOverlay();
};
const topKind = () => (F.topLayer() ? meta.get(F.topLayer()).kind : null);

// ---------------------------------------------------------------------------
// Toasts
// ---------------------------------------------------------------------------
function toast(text, tone = "ok", ic = tone === "ok" ? "check" : tone === "danger" ? "alert" : "info") {
	const box = $("#toasts");
	const el = document.createElement("div");
	el.className = `toast tone-border-${tone}`;
	el.innerHTML = `<span class="tone-${tone}">${icon(ic)}</span><span>${esc(text)}</span>`;
	box.append(el);
	while (box.children.length > 3) box.firstElementChild.remove();
	requestAnimationFrame(() => el.classList.add("is-in"));
	setTimeout(() => {
		el.classList.remove("is-in");
		setTimeout(() => el.remove(), 300);
	}, 4800);
	state.notifications.unshift({ at: Date.now(), text, tone });
}

// ---------------------------------------------------------------------------
// Barra lateral e barra superior
// ---------------------------------------------------------------------------
const RAIL = [
	{ id: "menu", icon: "menu", label: "Menu principal", run: () => run("menu"), pos: "top" },
	{ id: "home", icon: "home", label: "Início", run: () => run("home") },
	{ id: "library", icon: "library", label: "Biblioteca", run: () => run("library") },
	{ id: "recent", icon: "clock", label: "Jogados recentemente", run: () => goShelf("recent") },
	{ id: "fav", icon: "star", label: "Favoritos", run: () => goShelf("fav") },
	{ id: "emu", icon: "chip", label: "Emuladores", run: () => goLibTab("emu") },
	{ id: "add", icon: "plus", label: "Adicionar jogo", run: () => run("addGame") },
	{ id: "settings", icon: "settings", label: "Configurações", run: () => run("settings"), pos: "bottom" },
];

function renderRail() {
	const btn = (r) =>
		`<button class="rail-btn" data-f data-rail="${r.id}" aria-label="${esc(r.label)}">${icon(r.icon)}<span class="rail-label">${esc(r.label)}</span></button>`;
	$(".rail").innerHTML = `
		<div class="rail-top">${RAIL.filter((r) => r.pos === "top").map(btn).join("")}</div>
		<div class="rail-mid">${RAIL.filter((r) => !r.pos).map(btn).join("")}</div>
		<div class="rail-bottom">${RAIL.filter((r) => r.pos === "bottom").map(btn).join("")}</div>`;
	for (const b of $$(".rail-btn")) b.addEventListener("click", () => RAIL.find((r) => r.id === b.dataset.rail).run());
	markRail();
}
function markRail() {
	const active =
		state.screen === "home"
			? SHELVES[state.shelf].id === "fav"
				? "fav"
				: "home"
			: state.screen === "library"
				? LIB_TABS[state.libTab].id === "emu"
					? "emu"
					: "library"
				: "settings";
	for (const b of $$(".rail-btn")) {
		if (b.dataset.rail === active) b.setAttribute("aria-current", "page");
		else b.removeAttribute("aria-current");
	}
}

function renderTopbar() {
	$("#clock").textContent = timeF.format(new Date());
	const list = lib().filter((g) => g.sync !== "off");
	const conflicts = list.filter((g) => g.sync === "conflict").length;
	const waiting = list.filter((g) => g.sync === "cloud-syncing").length;
	let ic = "cloudCheck";
	let tone = "ok";
	let text = "Nuvem: tudo em dia";
	if (state.cloudOffline) [ic, tone, text] = ["cloudOff", "warn", "Nuvem indisponível"];
	else if (conflicts) [ic, tone, text] = ["alert", "danger", conflicts === 1 ? "1 conflito de save" : `${conflicts} conflitos de save`];
	else if (waiting) [ic, tone, text] = ["cloudSync", "warn", "Nuvem baixando saves"];
	const pill = $("#cloud-pill");
	pill.innerHTML = `<span class="tone-${tone}">${icon(ic)}</span><span>${text}</span>`;
	pill.setAttribute("aria-label", `${text}. Abrir central rápida`);
	$("#pc-name").innerHTML = `${icon("monitor")}<span>${esc(state.pcName)}</span>`;
	const [s, q] = $$(".topbar-right .icon-btn");
	s.innerHTML = icon("search");
	q.innerHTML = icon("bell");
}
$("#cloud-pill").addEventListener("click", () => run("quick"));
setInterval(() => ($("#clock").textContent = timeF.format(new Date())), 15000);

// ---------------------------------------------------------------------------
// Fundo (arte hero com crossfade)
// ---------------------------------------------------------------------------
let bgIdx = 0;
let bgToken = 0;
function setBackground(g) {
	const token = ++bgToken;
	const layers = $$(".bg-layer");
	const url = g?.art?.hero;
	const apply = (ok) => {
		if (token !== bgToken) return;
		const next = layers[1 - bgIdx];
		next.style.backgroundImage = ok ? `url("${url}")` : "";
		next.style.setProperty("--ph", g?.accent ?? "#3b4a6b");
		next.classList.toggle("is-ph", !ok);
		next.classList.add("is-on");
		layers[bgIdx].classList.remove("is-on");
		bgIdx = 1 - bgIdx;
	};
	if (!url) return apply(false);
	const im = new Image();
	im.onload = () => apply(true);
	im.onerror = () => apply(false);
	im.src = url;
}
function applyAccent(g) {
	const c = state.accent === "game" ? g?.accent : state.accent;
	if (c) app.style.setProperty("--game-accent", c);
	else app.style.removeProperty("--game-accent");
}

// ---------------------------------------------------------------------------
// Início
// ---------------------------------------------------------------------------
function renderHome() {
	$("#screen-home").innerHTML = `
		<div class="home">
			<div class="hero" id="hero"></div>
			<div class="shelf-head">
				<h2 class="shelf-title" id="shelf-title"></h2>
				<div class="shelf-tabs" id="shelf-tabs" role="tablist" aria-label="Seções do início"></div>
			</div>
			<div class="shelf" id="shelf" data-scroll="x" data-group="shelf"></div>
		</div>`;
	const shelf = $("#shelf");
	shelf.addEventListener(
		"wheel",
		(e) => {
			if (Math.abs(e.deltaY) > Math.abs(e.deltaX)) {
				shelf.scrollLeft += e.deltaY;
				e.preventDefault();
			}
		},
		{ passive: false },
	);
	renderShelf();
}

function renderHero(g) {
	const hero = $("#hero");
	if (!hero) return;
	hero.dataset.rendered = g?.id ?? "empty";
	if (!g) {
		delete hero.dataset.gameCtx;
		hero.innerHTML = `
			<div class="hero-logo"><h1 class="hero-title">Sua biblioteca está vazia</h1></div>
			<p class="hero-desc">Adicione um jogo para abrir pelo HUB e manter o save sincronizado com a sua pasta da nuvem.</p>
			<div class="hero-actions" data-group="hero-actions">
				<button class="btn btn-play" data-f data-action="addGame">${icon("plus")}<span>Adicionar jogo</span></button>
				<button class="btn btn-glass" data-f data-click="scan">${icon("folder")}<span>Procurar jogos numa pasta</span></button>
			</div>`;
		setBackground(null);
		applyAccent(null);
		return;
	}
	const s = syncInfo(g);
	hero.dataset.gameCtx = g.id;
	hero.innerHTML = `
		<div class="hero-logo">
			${g.art ? `<img src="${g.art.logo}" alt="" data-fallback="title">` : ""}
			<h1 class="hero-title ${g.art ? "sr-only" : ""}">${esc(g.name)}</h1>
		</div>
		<div class="hero-meta">
			<span>${esc(g.dev)}</span><span class="dot">•</span><span>${g.year}</span><span class="dot">•</span>
			<span class="meta-ico" title="Funciona com controle">${icon("gamepad")}</span><span class="dot">•</span>
			<span>${playtime(g.playtime)}</span>
			${g.ach ? `<span class="dot">•</span><span class="meta-ico">${icon("trophy")}</span><span>${g.ach[0]}/${g.ach[1]}</span>` : ""}
		</div>
		<div class="chips">${[g.source, ...g.tags].map((t, i) => `<span class="chip ${i === 0 ? "chip-source" : ""}">${esc(t)}</span>`).join("")}</div>
		<p class="hero-desc">${esc(g.desc)}</p>
		<div class="hero-actions" data-group="hero-actions">
			<button class="btn btn-play" data-f data-click="play">${icon("play")}<span>Jogar</span></button>
			<button class="btn btn-glass" data-f data-click="details">${icon("history")}<span>Saves e conquistas</span></button>
			<button class="btn btn-glass btn-icon" data-f data-click="options" aria-label="Mais opções">${icon("more")}</button>
		</div>
		<div class="hero-status" role="status">
			<span class="tone-${s.tone}">${icon(s.icon)}</span>
			<strong class="tone-${s.tone}">${s.label}</strong>
			<span class="hero-status-detail">${esc(syncDetail(g))}</span>
		</div>`;
	setBackground(g);
	applyAccent(g);
}

function setHero(g) {
	if (state.hero === g && $("#hero")?.dataset.rendered === (g?.id ?? "empty")) return;
	state.hero = g;
	renderHero(g);
}

function capsule(g) {
	const s = syncInfo(g);
	const flag = s.tone === "danger" || s.tone === "warn";
	return `<button class="cap" data-f data-game="${g.id}" aria-label="${esc(g.name)}. ${s.label}" title="${esc(g.name)}">
		<span class="cap-art">${art(g, "header")}</span>
		${g.isNew ? '<span class="cap-badge">Novo</span>' : ""}
		${flag ? `<span class="cap-flag tone-${s.tone}">${icon(s.icon)}</span>` : ""}
		<span class="cap-bar"><span class="cap-name">${esc(g.name)}</span><span class="cap-sync tone-${s.tone}">${icon(s.icon)}</span></span>
	</button>`;
}

function renderShelf() {
	const sh = SHELVES[state.shelf];
	$("#shelf-title").textContent = sh.title;
	$("#shelf-tabs").innerHTML = SHELVES.map(
		(s, i) => `<button class="shelf-tab" role="tab" aria-selected="${i === state.shelf}" data-shelf="${i}" tabindex="-1">${esc(s.title)}</button>`,
	).join("");
	const list = sh.list();
	const tiles = `
		<button class="cap cap-tile" data-f data-action="library"><span class="cap-art">${icon("library")}</span><span class="cap-bar"><span class="cap-name">Ver biblioteca</span></span></button>
		<button class="cap cap-tile" data-f data-action="addGame"><span class="cap-art">${icon("plus")}</span><span class="cap-bar"><span class="cap-name">Adicionar jogo</span></span></button>`;
	$("#shelf").innerHTML = list.length
		? list.map(capsule).join("") + tiles
		: `<div class="shelf-empty" data-f tabindex="0">${icon(lib().length ? "info" : "plus")}<span>${esc(lib().length ? sh.empty : "Nenhum jogo ainda.")}</span></div>${tiles}`;
	$("#shelf").scrollLeft = 0;
	markRail();
	if (!state.hero || !list.includes(state.hero)) setHero(list[0] ?? (lib().length ? state.hero : null));
}

function setShelf(i, { focusShelf = true } = {}) {
	state.shelf = (i + SHELVES.length) % SHELVES.length;
	const keepHero = state.hero;
	renderShelf();
	if (focusShelf) F.focus($("#shelf [data-f]"));
	else if (keepHero && !SHELVES[state.shelf].list().includes(keepHero)) setHero(SHELVES[state.shelf].list()[0] ?? keepHero);
	updateLegend();
}

function goShelf(id) {
	if (state.screen !== "home") showScreen("home", { focus: false });
	setShelf(SHELVES.findIndex((s) => s.id === id));
}

// Atualiza o herói quando um jogo recebe foco (com pequeno atraso para navegação rápida).
let heroTimer;
document.addEventListener("sgs:focus", (e) => {
	const id = e.detail.dataset.game;
	if (id && state.screen === "home" && !F.topLayer()) {
		clearTimeout(heroTimer);
		heroTimer = setTimeout(() => setHero(byId(id)), 90);
	}
	updateLegend();
});

// ---------------------------------------------------------------------------
// Biblioteca
// ---------------------------------------------------------------------------
function filteredLibrary() {
	const tab = LIB_TABS[state.libTab];
	const { status, source, sort } = state.filters;
	return sortGames(
		lib().filter((g) => tab.f(g) && (!status.size || status.has(g.sync)) && (!source.size || source.has(g.source))),
		sort,
	);
}
const filterCount = () => state.filters.status.size + state.filters.source.size;

function renderLibrary() {
	const list = filteredLibrary();
	const total = lib().length;
	const fc = filterCount();
	$("#screen-library").innerHTML = `
		<div class="lib">
			<div class="lib-head">
				<h1>Biblioteca</h1>
				<span class="lib-count">${list.length === total ? `${total} jogos` : `${list.length} de ${total} jogos`}</span>
			</div>
			<div class="lib-bar">
				<div class="tabs" role="tablist" data-group="lib-tabs">
					${LIB_TABS.map((t, i) => `<button class="tab" role="tab" data-f data-libtab="${i}" aria-selected="${i === state.libTab}">${esc(t.label)}</button>`).join("")}
				</div>
				<div class="lib-tools" data-group="lib-tools">
					<button class="btn btn-glass btn-sm" data-f data-action="filter">${icon("filter")}<span>Filtros${fc ? ` (${fc})` : ""}</span>${glyph("filter")}</button>
					<button class="btn btn-glass btn-sm" data-f data-click="sort">${icon("sort")}<span>Ordenar: ${SORTS[state.filters.sort].label}</span></button>
				</div>
			</div>
			${
				list.length
					? `<div class="lib-grid" data-group="lib-grid">${list
							.map((g) => {
								const s = syncInfo(g);
								return `<button class="card" data-f data-game="${g.id}" title="${esc(g.name)}" aria-label="${esc(g.name)}. ${s.label}">
									<span class="card-art">${art(g, "cover")}${g.isNew ? '<span class="cap-badge">Novo</span>' : ""}</span>
									<span class="card-name">${esc(g.name)}</span>
									<span class="card-sub tone-${s.tone}">${icon(s.icon)}<span>${s.label}</span></span>
								</button>`;
							})
							.join("")}</div>`
					: total
						? emptyState("filter", "Nenhum jogo com estes filtros", "Mude a aba ou limpe os filtros para ver mais jogos.", `<button class="btn btn-play" data-f data-click="clear-filters">${icon("x")}<span>Limpar filtros</span></button>`)
						: emptyState("library", "Nenhum jogo ainda", "Adicione um jogo pelo executável ou procure jogos numa pasta.", `<button class="btn btn-play" data-f data-action="addGame">${icon("plus")}<span>Adicionar jogo</span></button><button class="btn btn-glass" data-f data-click="scan">${icon("folder")}<span>Procurar numa pasta</span></button>`)
			}
		</div>`;
	markRail();
}
const emptyState = (ic, title, text, actions) =>
	`<div class="empty" data-group="empty">${icon(ic, "empty-ico")}<h2>${esc(title)}</h2><p>${esc(text)}</p><div class="empty-actions">${actions}</div></div>`;

function refreshLibrary({ keepFocus = true } = {}) {
	if (state.screen !== "library") return;
	const cur = F.current();
	const key = cur && $("#screen-library").contains(cur) ? (cur.dataset.game ? `[data-game="${cur.dataset.game}"]` : cur.dataset.libtab ? `[data-libtab="${cur.dataset.libtab}"]` : cur.dataset.action ? `[data-action="${cur.dataset.action}"]` : cur.dataset.click ? `[data-click="${cur.dataset.click}"]` : null) : null;
	renderLibrary();
	if (!keepFocus || F.topLayer()) return;
	F.focus((key && $(`#screen-library ${key}`)) || $("#screen-library .card") || F.focusables($("#screen-library"))[0], { scroll: false });
}

function goLibTab(id) {
	state.libTab = LIB_TABS.findIndex((t) => t.id === id);
	showScreen("library");
}

// ---------------------------------------------------------------------------
// Configurações
// ---------------------------------------------------------------------------
const SECTIONS = [
	{ id: "nuvem", label: "Nuvem", icon: "cloud" },
	{ id: "backups", label: "Backups e versões", icon: "history" },
	{ id: "biblioteca", label: "Biblioteca", icon: "library" },
	{ id: "controles", label: "Controles", icon: "gamepad" },
	{ id: "aparencia", label: "Aparência", icon: "star" },
	{ id: "sobre", label: "Sobre", icon: "info" },
];

const row = (label, hint, ctrl) =>
	`<div class="set-row"><div class="set-text"><div class="set-label">${esc(label)}</div>${hint ? `<div class="set-hint">${hint}</div>` : ""}</div><div class="set-ctrl">${ctrl}</div></div>`;
const toggle = (key, on, label) =>
	`<button class="toggle" role="switch" data-f data-toggle="${key}" aria-checked="${on}" aria-label="${esc(label)}"><span></span></button>`;
const seg = (key, value, opts, label) =>
	`<div class="seg" role="radiogroup" aria-label="${esc(label)}">${opts.map(([v, l]) => `<button role="radio" data-f data-seg="${key}" data-value="${v}" aria-checked="${v === value}">${esc(l)}</button>`).join("")}</div>`;
const act = (click, label, ic) => `<button class="btn btn-glass btn-sm" data-f data-click="${click}">${ic ? icon(ic) : ""}<span>${esc(label)}</span></button>`;

function settingsBody(id) {
	switch (id) {
		case "nuvem":
			return `<h2>Nuvem</h2>
				${row("Pasta da nuvem", `<code>${esc(state.cloudPath)}</code><br>Use uma pasta que o OneDrive, Google Drive, Dropbox ou Syncthing mantém sincronizada.`, act("toast:Seletor de pasta: no app final abre a janela do sistema.", "Alterar…", "folder") + act("verify", "Verificar pasta", "check"))}
				${row("Nome deste PC", "Aparece no histórico de versões e no aviso de jogo aberto em outro PC.", `<span class="value">${esc(state.pcName)}</span>${act("toast:Edição do nome: no app final abre um campo de texto.", "Alterar…")}`)}
				${row("Avisar quando outro PC estiver jogando", "Evita sobrescrever um save que ainda está em uso.", toggle("lockWarn", true, "Avisar quando outro PC estiver jogando"))}
				${row("Dica para o OneDrive", "Marque a pasta <b>SyncGameSave</b> como “Sempre manter neste dispositivo”. Assim os saves não ficam só na nuvem.", "")}`;
		case "backups":
			return `<h2>Backups e versões</h2>
				${row("Versões guardadas por jogo", "As versões mais antigas saem primeiro. A versão atual nunca é apagada.", seg("retention", String(state.retention), [["10", "10"], ["20", "20"], ["50", "50"]], "Versões guardadas por jogo"))}
				${row("Guardar uma versão por dia", "Além das últimas versões, mantém uma por dia nos últimos 30 dias.", toggle("daily", true, "Guardar uma versão por dia"))}
				${row("Backups locais", "Antes de restaurar ou resolver um conflito, o save deste PC é copiado para cá.<br><code>%LOCALAPPDATA%\\SyncGameSave\\backups</code>", act("toast:No app final abre o Explorador de Arquivos.", "Abrir pasta", "folder"))}`;
		case "biblioteca":
			return `<h2>Biblioteca</h2>
				${row("Pastas para procurar jogos", "<code>C:\\Games</code> · <code>D:\\Jogos</code>", act("scan", "Adicionar pasta", "plus"))}
				${row("Catálogo de jogos conhecidos", "Manifesto do Ludusavi: nomes e caminhos de save. Atualizado há 3 dias · 34.812 jogos.", act("toast:Catálogo já está na versão mais recente.", "Atualizar agora", "refresh"))}
				${row("Chave da SteamGridDB", "Opcional. Busca capas, logos e fundos para jogos sem arte.", `<span class="value dim">Não configurada</span>${act("toast:No app final abre um campo para colar a chave.", "Configurar…")}`)}`;
		case "controles":
			return `<h2>Controles</h2>
				${row("Ícones dos botões", "Automático usa o último dispositivo que você tocou.", seg("glyphPref", state.glyphPref, [["auto", "Automático"], ["xbox", "Xbox"], ["ps", "PlayStation"], ["keyboard", "Teclado"]], "Ícones dos botões"))}
				${row("Vibração", "Vibra de leve ao concluir um envio ou ao encontrar um conflito.", toggle("rumble", true, "Vibração"))}
				${row("Atalhos", "Veja todos os atalhos de teclado e controle.", act("help", "Ver atalhos", "keyboard"))}`;
		case "aparencia": {
			const sw = [["game", "Da arte do jogo"], ["#5aa9ff", "Azul"], ["#b46cf0", "Roxo"], ["#3ddc97", "Verde"], ["#ff8a3d", "Laranja"]];
			return `<h2>Aparência</h2>
				${row("Cor de destaque", "", `<div class="swatches" role="radiogroup" aria-label="Cor de destaque">${sw.map(([v, l]) => `<button class="swatch" role="radio" data-f data-seg="accent" data-value="${v}" aria-checked="${state.accent === v}" aria-label="${l}" title="${l}" style="--c:${v === "game" ? "conic-gradient(#f2c14e,#e05a8a,#5aa9ff,#3ddc97,#f2c14e)" : v}"></button>`).join("")}</div>`)}
				${row("Transparência", "Desligue em PCs mais fracos: os painéis ficam sólidos.", toggle("glass", root.dataset.glass === "on", "Transparência"))}
				${row("Movimento", "Reduzido tira zoom e transições.", seg("motion", root.dataset.motion, [["auto", "Do sistema"], ["full", "Completo"], ["reduced", "Reduzido"]], "Movimento"))}
				${row("Sons de navegação", "", toggle("sounds", state.sounds, "Sons de navegação"))}
				${row("Tema", "Temas da comunidade chegam numa versão futura.", '<span class="value dim">Padrão escuro</span>')}`;
		}
		default:
			return `<h2>Sobre</h2>
				<div class="about">
					<div class="about-logo">${icon("cloudCheck")}</div>
					<div><div class="about-name">Sync Game Save</div><div class="dim">Versão 0.0.0 (protótipo) · build 1</div></div>
				</div>
				${row("Caminhos de save", "Manifesto do <b>Ludusavi</b>, de mtkennerly (licença MIT).", "")}
				${row("Artes dos jogos", "Artes da Steam usadas só neste protótipo, para dar realismo.", "")}
				${row("Licenças", "Lista das bibliotecas usadas e suas licenças.", act("toast:No app final abre a lista de licenças.", "Ver licenças"))}`;
	}
}

function renderSettings() {
	$("#screen-settings").innerHTML = `
		<div class="settings">
			<nav class="set-nav" data-group="set-nav" aria-label="Seções das configurações">
				<h1>Configurações</h1>
				${SECTIONS.map((s) => `<button class="set-nav-item" data-f data-section="${s.id}" aria-current="${s.id === state.settings}">${icon(s.icon)}<span>${s.label}</span></button>`).join("")}
			</nav>
			<div class="set-body panel" data-group="set-body" id="set-body">${settingsBody(state.settings)}</div>
		</div>`;
}

function setSection(id) {
	if (state.settings === id) return;
	state.settings = id;
	$("#set-body").innerHTML = settingsBody(id);
	for (const b of $$(".set-nav-item")) b.setAttribute("aria-current", String(b.dataset.section === id));
}
document.addEventListener("sgs:focus", (e) => {
	if (e.detail.dataset.section) setSection(e.detail.dataset.section);
});

// ---------------------------------------------------------------------------
// Troca de telas
// ---------------------------------------------------------------------------
function showScreen(name, { focus = true } = {}) {
	closeAll();
	state.screen = name;
	app.dataset.screen = name;
	for (const s of $$(".screen")) s.hidden = s.id !== `screen-${name}`;
	if (name === "home") {
		renderShelf();
		renderHero(state.hero ?? SHELVES[state.shelf].list()[0] ?? null);
		if (focus) F.focusGroup($("#shelf"));
	} else if (name === "library") {
		renderLibrary();
		if (focus) F.focus($("#screen-library .card") ?? F.focusables($("#screen-library"))[0]);
	} else {
		renderSettings();
		if (focus) F.focus($(`.set-nav-item[data-section="${state.settings}"]`));
	}
	markRail();
	updateLegend();
}

function refreshAll() {
	renderTopbar();
	if (state.screen === "home") {
		const cur = F.current();
		const id = cur?.dataset.game;
		const inShelf = cur && $("#shelf")?.contains(cur);
		renderShelf();
		renderHero(state.hero);
		if (inShelf && !F.topLayer()) F.focus((id && $(`#shelf [data-game="${id}"]`)) || $("#shelf [data-f]"), { scroll: true });
	} else if (state.screen === "library") refreshLibrary();
}

// ---------------------------------------------------------------------------
// Menu principal
// ---------------------------------------------------------------------------
function openMenu() {
	const items = [
		["home", "home", "Início"],
		["library", "library", "Biblioteca"],
		["search", "search", "Buscar"],
		["addGame", "plus", "Adicionar jogo"],
		["syncAll", "refresh", "Sincronizar tudo agora"],
		["quick", "bell", "Central rápida"],
		["settings", "settings", "Configurações"],
		["help", "keyboard", "Atalhos"],
		["tray", "tray", "Minimizar para a bandeja"],
		["quit", "power", "Sair do HUB"],
	];
	const el = openOverlay({
		kind: "menu",
		label: "Menu principal",
		legend: [["confirm", "Selecionar"], ["back", "Fechar"], ["menu", "Fechar"]],
		html: `<aside class="sheet sheet-left panel">
			<div class="sheet-head"><div class="brand">${icon("cloudCheck")}<span>Sync Game Save</span></div></div>
			<nav class="menu-list" data-group="menu">
				${items.map(([a, ic, l], i) => `<button class="menu-item" data-f data-menu="${a}" ${i === 0 && state.screen === "home" ? "data-autofocus" : ""}>${icon(ic)}<span>${l}</span><span class="menu-hint">${glyph(a, "keyboard")}</span></button>`).join("")}
			</nav>
		</aside>`,
	});
	for (const b of $$(".menu-item", el))
		b.addEventListener("click", () => {
			const a = b.dataset.menu;
			closeOverlay(el);
			if (a === "syncAll") syncAll();
			else if (a === "tray") toast("No app final a janela some e o HUB fica no ícone da bandeja.", "info", "tray");
			else if (a === "quit") toast("No app final: o HUB fecha. Jogos abertos continuam sendo acompanhados até fechar.", "info", "power");
			else run(a);
		});
}

// ---------------------------------------------------------------------------
// Busca
// ---------------------------------------------------------------------------
function openSearch() {
	const el = openOverlay({
		kind: "search",
		label: "Buscar jogos",
		legend: (cur) => (cur?.matches("input") ? [["confirm", "Ir aos resultados"], ["back", "Fechar"]] : [["confirm", "Abrir"], ["options", "Opções"], ["back", "Fechar"]]),
		html: `<div class="search-panel panel">
			<label class="search-field">${icon("search")}
				<input type="search" data-f data-autofocus placeholder="Buscar na biblioteca e no catálogo" aria-label="Buscar jogos" autocomplete="off" spellcheck="false">
			</label>
			<div class="search-results" data-group="results" aria-live="polite"></div>
			<p class="search-note">No app final, a busca usa um índice SQLite FTS5 do catálogo, com resultados em milissegundos.</p>
		</div>`,
	});
	const input = $("input", el);
	const out = $(".search-results", el);
	const render = () => {
		const q = norm(input.value.trim());
		const games = lib().filter((g) => !q || norm(`${g.name} ${g.dev} ${g.tags.join(" ")}`).includes(q));
		const cat = q ? catalog.filter((c) => norm(c.name).includes(q)) : [];
		const shown = q ? games : sortGames(games, "recent").slice(0, 5);
		out.innerHTML = `
			${shown.length ? `<h3>${q ? "Na biblioteca" : "Jogados recentemente"}</h3>${shown.map((g) => `<button class="result" data-f data-game="${g.id}">${art(g, "header")}<span class="result-text"><span class="result-name">${esc(g.name)}</span><span class="result-sub">${esc(g.dev)} · <span class="tone-${syncInfo(g).tone}">${syncInfo(g).label}</span></span></span></button>`).join("")}` : ""}
			${cat.length ? `<h3>No catálogo de jogos conhecidos</h3>${cat.map((c) => `<button class="result" data-f data-catalog="${esc(c.name)}"><span class="ph result-ph">${icon("plus")}</span><span class="result-text"><span class="result-name">${esc(c.name)}</span><span class="result-sub">${c.saves} ${c.saves === 1 ? "caminho de save conhecido" : "caminhos de save conhecidos"} · Adicionar à biblioteca</span></span></button>`).join("")}` : ""}
			${q && !shown.length && !cat.length ? `<div class="search-empty"><p>Nenhum jogo encontrado para “${esc(input.value.trim())}”.</p><button class="btn btn-glass btn-sm" data-f data-action="addGame">${icon("plus")}<span>Adicionar jogo manualmente</span></button></div>` : ""}`;
	};
	input.addEventListener("input", render);
	out.addEventListener("click", (e) => {
		const b = e.target.closest(".result");
		if (!b) return;
		if (b.dataset.game) {
			const g = byId(b.dataset.game);
			closeOverlay(el);
			openDetails(g);
		} else {
			closeOverlay(el);
			openAddGame(b.dataset.catalog);
		}
	});
	render();
}

// ---------------------------------------------------------------------------
// Filtros
// ---------------------------------------------------------------------------
function openFilters() {
	if (state.screen !== "library") showScreen("library");
	const chip = (group, value, label, on) =>
		`<button class="fchip" data-f data-fgroup="${group}" data-value="${esc(value)}" aria-pressed="${on}">${on ? icon("check") : ""}<span>${esc(label)}</span></button>`;
	const body = () => `
		<h3>Situação do save</h3>
		<div class="fchips" data-group="f-status">${Object.entries(SYNC).map(([k, v]) => chip("status", k, v.label, state.filters.status.has(k))).join("")}</div>
		<h3>Origem</h3>
		<div class="fchips" data-group="f-source">${SOURCES.map((s) => chip("source", s, s, state.filters.source.has(s))).join("")}</div>
		<h3>Ordenar por</h3>
		<div class="fchips" data-group="f-sort">${Object.entries(SORTS).map(([k, v]) => chip("sort", k, v.label, state.filters.sort === k)).join("")}</div>`;
	const el = openOverlay({
		kind: "filters",
		label: "Filtros da biblioteca",
		legend: [["confirm", "Marcar"], ["back", "Concluir"], ["filter", "Fechar"]],
		html: `<aside class="sheet sheet-right panel">
			<div class="sheet-head"><h2>${icon("filter")}<span>Filtros</span></h2><span class="dim" id="f-count"></span></div>
			<div class="sheet-body" id="f-body">${body()}</div>
			<div class="sheet-foot" data-group="f-foot">
				<button class="btn btn-glass" data-f data-click="clear-filters">${icon("x")}<span>Limpar filtros</span></button>
				<button class="btn btn-play" data-f data-click="close-top">${icon("check")}<span>Concluir</span></button>
			</div>
		</aside>`,
	});
	const count = () => ($("#f-count", el).textContent = `${filteredLibrary().length} jogos`);
	count();
	$("#f-body", el).addEventListener("click", (e) => {
		const b = e.target.closest(".fchip");
		if (!b) return;
		const { fgroup, value } = b.dataset;
		if (fgroup === "sort") state.filters.sort = value;
		else {
			const set = state.filters[fgroup];
			set.has(value) ? set.delete(value) : set.add(value);
		}
		const sel = `[data-fgroup="${fgroup}"][data-value="${value}"]`;
		$("#f-body", el).innerHTML = body();
		F.focus($(sel, el), { scroll: false });
		count();
		refreshLibrary({ keepFocus: false });
	});
	el.addEventListener("sgs:refresh", () => {
		$("#f-body", el).innerHTML = body();
		count();
	});
}

function clearFilters() {
	state.filters.status.clear();
	state.filters.source.clear();
	state.libTab = 0;
	const fo = $("#overlays .ov-filters:not(.closing)");
	if (fo) fo.dispatchEvent(new Event("sgs:refresh"));
	refreshLibrary({ keepFocus: !fo });
	if (fo) F.focus($('[data-click="close-top"]', fo));
	else F.focus($("#screen-library .card") ?? F.focusables($("#screen-library"))[0]);
	toast("Filtros limpos.", "info", "filter");
}

// ---------------------------------------------------------------------------
// Opções do jogo
// ---------------------------------------------------------------------------
function openOptions(g) {
	const items = [
		["play", "play", "Jogar"],
		["play-nosync", "cloudOff", "Jogar sem sincronizar"],
		["details", "history", "Saves e conquistas"],
		["sync-now", "refresh", "Sincronizar agora"],
		["fav", "star", g.fav ? "Remover dos favoritos" : "Favoritar"],
		["toggle-sync", "cloud", g.sync === "off" ? "Ligar sincronização" : "Desligar sincronização"],
		["folder", "folder", "Abrir pasta do save"],
		["hide", "eyeOff", "Ocultar da biblioteca"],
	];
	const el = openOverlay({
		kind: "options",
		label: `Opções de ${g.name}`,
		game: g,
		legend: [["confirm", "Selecionar"], ["back", "Fechar"]],
		html: `<aside class="sheet sheet-right panel">
			<div class="sheet-head game-head"><span class="mini-art">${art(g, "header")}</span><div><h2 class="clamp-2">${esc(g.name)}</h2><span class="dim">${esc(g.dev)}</span></div></div>
			<nav class="menu-list" data-group="opts">${items.map(([a, ic, l]) => `<button class="menu-item" data-f data-opt="${a}">${icon(ic)}<span>${esc(l)}</span></button>`).join("")}</nav>
		</aside>`,
	});
	for (const b of $$(".menu-item", el))
		b.addEventListener("click", () => {
			const a = b.dataset.opt;
			closeOverlay(el);
			if (a === "play") play(g);
			else if (a === "play-nosync") play(g, { noSync: true });
			else if (a === "details") openDetails(g);
			else if (a === "sync-now") syncGame(g);
			else if (a === "fav") {
				g.fav = !g.fav;
				toast(g.fav ? `${g.name} está nos favoritos.` : `${g.name} saiu dos favoritos.`, "info", "star");
				refreshAll();
			} else if (a === "toggle-sync") {
				g.sync = g.sync === "off" ? "local-newer" : "off";
				toast(g.sync === "off" ? `Sincronização de ${g.name} desligada. As versões na nuvem continuam guardadas.` : `Sincronização de ${g.name} ligada. O save será enviado ao jogar.`, "info", "cloud");
				refreshAll();
			} else if (a === "folder") toast("No app final abre a pasta do save no Explorador de Arquivos.", "info", "folder");
			else if (a === "hide") {
				g.hidden = true;
				if (state.hero === g) state.hero = null;
				toast(`${g.name} foi ocultado. Para mostrar de novo, use o filtro “Ocultos” (no app final).`, "info", "eyeOff");
				refreshAll();
			}
		});
}

// ---------------------------------------------------------------------------
// Detalhes: saves, conquistas e informações
// ---------------------------------------------------------------------------
const DETAIL_TABS = [
	["save", "Saves"],
	["ach", "Conquistas"],
	["info", "Informações"],
];

function openDetails(g, tab = "save") {
	let current = tab;
	const s = () => syncInfo(g);
	const tabBody = () => {
		if (current === "save") {
			const v = g.versions;
			return `<div class="status-card panel-soft">
					<span class="status-ico tone-${s().tone}">${icon(s().icon)}</span>
					<div><div class="status-title tone-${s().tone}">${s().label}</div><div class="dim">${esc(syncDetail(g))}</div></div>
					<button class="btn btn-glass btn-sm" data-f data-click="sync-game">${icon("refresh")}<span>Sincronizar agora</span></button>
				</div>
				<h3>Histórico de versões <span class="dim">· ${v.length} guardadas</span></h3>
				${
					v.length
						? `<ol class="timeline" data-group="timeline">${v
								.map(
									(x, i) => `<li><button class="tl-item" data-f data-version="${x.id}">
								<span class="tl-dot ${i === 0 ? "is-current" : ""}"></span>
								<span class="tl-main"><span class="tl-date">${dtf.format(x.at)} <span class="dim">· ${ago(x.at)}</span></span>
								<span class="tl-sub">${icon("monitor")}${esc(x.device)} · ${esc(x.reason)} · ${x.files} ${x.files === 1 ? "arquivo" : "arquivos"} · ${size(x.size)}</span></span>
								${i === 0 ? '<span class="badge">Atual</span>' : `<span class="tl-action">${icon("history")}Restaurar</span>`}
							</button></li>`,
								)
								.join("")}</ol>`
						: `<p class="dim">Nenhuma versão na nuvem. Ligue a sincronização nas opções do jogo para guardar versões.</p>`
				}`;
		}
		if (current === "ach") {
			const list = achievementsFor(g);
			if (!g.ach) return `<div class="empty small">${icon("trophy", "empty-ico")}<h2>Sem conquistas conhecidas</h2><p>Este jogo não tem conquistas em nenhum provedor configurado.</p></div>`;
			const pct = Math.round((g.ach[0] / g.ach[1]) * 100);
			return `<div class="ach-summary">
					<div class="ring" style="--p:${pct}"><span>${pct}%</span></div>
					<div><div class="status-title">${g.ach[0]} de ${g.ach[1]} conquistas</div><div class="dim">Fonte: Steam Web API (exemplo)</div></div>
				</div>
				<div class="ach-grid" data-group="ach">${list
					.map(
						(a) => `<div class="ach ${a.unlocked ? "" : "is-locked"}" data-f tabindex="0">
						<span class="ach-ico">${icon(a.unlocked ? "trophy" : "lock")}</span>
						<span><span class="ach-name">${esc(a.name)}</span><span class="ach-sub">${a.unlocked ? `Desbloqueada em ${dateF.format(a.at)}` : "Bloqueada"}</span></span>
					</div>`,
					)
					.join("")}</div>`;
		}
		return `<dl class="info-list">
				<div><dt>Origem</dt><dd>${esc(g.source)}</dd></div>
				<div><dt>Tempo de jogo</dt><dd>${playtime(g.playtime)}</dd></div>
				<div><dt>Última vez jogado</dt><dd>${g.lastPlayed ? `${dtf.format(g.lastPlayed)} (${ago(g.lastPlayed)})` : "Nunca"}</dd></div>
				<div><dt>Adicionado em</dt><dd>${dateF.format(g.added)}</dd></div>
				<div><dt>Instalação</dt><dd><code>${esc(g.installPath)}</code></dd></div>
			</dl>
			<h3>Caminhos do save</h3>
			<ul class="paths">${g.savePaths.map((p) => `<li class="${p.found ? "" : "is-missing"}">${icon(p.found ? "file" : "alert")}<code>${esc(p.path)}</code><span class="dim">${p.found ? `${p.files} ${p.files === 1 ? "arquivo" : "arquivos"} · ${size(p.size)}` : "Não encontrado neste PC"}</span></li>`).join("")}</ul>
			<div class="row-actions" data-group="info-actions">${act("toast:No app final abre a pasta de instalação.", "Abrir pasta de instalação", "folder")}${act("toast:No app final abre o editor de caminhos do save.", "Editar caminhos do save", "file")}</div>`;
	};
	const el = openOverlay({
		kind: "details",
		label: `${g.name}: saves e conquistas`,
		game: g,
		onTab: (d) => switchTab((DETAIL_TABS.findIndex(([k]) => k === current) + d + DETAIL_TABS.length) % DETAIL_TABS.length),
		legend: (cur) => [["confirm", cur?.dataset.version ? "Restaurar" : "Selecionar"], ["prevTab", ""], ["nextTab", "Aba"], ["options", "Opções"], ["back", "Voltar"]],
		html: `<div class="details">
			<div class="details-bg" style="${g.art ? `background-image:url('${g.art.hero}')` : `--ph:${g.accent}`}" data-ph="${g.art ? "" : "1"}"></div>
			<div class="details-side">
				<div class="hero-logo small">${g.art ? `<img src="${g.art.logo}" alt="" data-fallback="title">` : ""}<h1 class="hero-title ${g.art ? "sr-only" : ""}">${esc(g.name)}</h1></div>
				<div class="hero-meta"><span>${esc(g.dev)}</span><span class="dot">•</span><span>${g.year}</span><span class="dot">•</span><span>${playtime(g.playtime)}</span></div>
				<div class="hero-actions" data-group="d-actions">
					<button class="btn btn-play" data-f data-click="play-ctx">${icon("play")}<span>Jogar</span></button>
					<button class="btn btn-glass btn-icon" data-f data-click="options-ctx" aria-label="Mais opções">${icon("more")}</button>
				</div>
			</div>
			<div class="details-main panel">
				<div class="tabs" role="tablist" data-group="d-tabs">${DETAIL_TABS.map(([k, l]) => `<button class="tab" role="tab" data-f data-dtab="${k}" aria-selected="${k === current}">${l}</button>`).join("")}<span class="tab-hint">${glyph("prevTab")}${glyph("nextTab")}</span></div>
				<div class="details-body" id="d-body">${tabBody()}</div>
			</div>
		</div>`,
	});
	function switchTab(i) {
		current = DETAIL_TABS[i][0];
		for (const t of $$("[data-dtab]", el)) t.setAttribute("aria-selected", String(t.dataset.dtab === current));
		$("#d-body", el).innerHTML = tabBody();
		F.focus($(`[data-dtab="${current}"]`, el), { scroll: false });
		sound("move");
	}
	el.addEventListener("click", (e) => {
		const t = e.target.closest("[data-dtab]");
		if (t) switchTab(DETAIL_TABS.findIndex(([k]) => k === t.dataset.dtab));
		const v = e.target.closest("[data-version]");
		if (v) {
			const ver = g.versions.find((x) => x.id === v.dataset.version);
			if (ver === g.versions[0]) toast("Esta já é a versão atual do save.", "info", "info");
			else confirmRestore(g, ver, () => ($("#d-body", el).innerHTML = tabBody()));
		}
	});
	el.addEventListener("sgs:refresh", () => ($("#d-body", el).innerHTML = tabBody()));
	if (tab !== "save") F.focus($(`[data-dtab="${tab}"]`, el));
}

function confirmRestore(g, v, done) {
	const el = openOverlay({
		kind: "confirm",
		label: "Restaurar versão do save",
		legend: [["confirm", "Selecionar"], ["back", "Cancelar"]],
		html: `<div class="modal panel">
			<h2>Restaurar a versão de ${dtf.format(v.at)}?</h2>
			<p>O save atual deste PC será guardado como backup antes. Você pode voltar a ele pelo histórico.</p>
			<ul class="kv"><li><span>Origem</span><b>${esc(v.device)}</b></li><li><span>Motivo</span><b>${esc(v.reason)}</b></li><li><span>Arquivos</span><b>${v.files} · ${size(v.size)}</b></li></ul>
			<div class="modal-actions" data-group="m-actions">
				<button class="btn btn-glass" data-f data-autofocus data-click="close-top">Cancelar</button>
				<button class="btn btn-play" data-f data-click="do-restore">${icon("history")}<span>Restaurar</span></button>
			</div>
		</div>`,
	});
	$('[data-click="do-restore"]', el).addEventListener("click", async () => {
		closeOverlay(el);
		toast("Backup do save atual criado. Restaurando…", "info", "history");
		await sleep(900);
		g.versions.unshift({ ...v, id: `${g.id}-r${Date.now()}`, at: Date.now(), device: state.pcName, reason: "Restaurado de versão antiga" });
		g.sync = "ok";
		done?.();
		refreshAll();
		toast(`Versão de ${dtf.format(v.at)} restaurada e conferida.`);
	});
}

// ---------------------------------------------------------------------------
// Fluxo de jogar: sincronizar → abrir → acompanhar → enviar
// ---------------------------------------------------------------------------
function stepper(container) {
	return (text) => {
		const li = document.createElement("li");
		li.className = "step is-run";
		li.innerHTML = `<span class="step-ico"><span class="spinner"></span></span><span class="step-text"></span>`;
		const t = li.querySelector(".step-text");
		const ic = li.querySelector(".step-ico");
		t.textContent = text;
		container.append(li);
		const set = (cls, iconName, txt) => {
			li.className = `step ${cls}`;
			ic.innerHTML = iconName ? icon(iconName) : '<span class="spinner"></span>';
			if (txt) t.textContent = txt;
		};
		return {
			ok: (txt) => set("is-ok", "check", txt),
			fail: (txt) => set("is-err", "alert", txt),
			warn: (txt) => set("is-warn", "alert", txt),
			wait: (txt) => set("is-wait", null, txt),
		};
	};
}

function launchShell(g, title) {
	return `<div class="launch">
		<div class="details-bg" style="${g.art ? `background-image:url('${g.art.hero}')` : `--ph:${g.accent}`}" data-ph="${g.art ? "" : "1"}"></div>
		<div class="launch-box">
			<div class="hero-logo">${g.art ? `<img src="${g.art.logo}" alt="" data-fallback="title">` : ""}<h1 class="hero-title ${g.art ? "sr-only" : ""}">${esc(g.name)}</h1></div>
			<p class="launch-kicker">${esc(title)}</p>
			<ol class="steps" aria-live="polite"></ol>
			<div class="launch-actions" data-group="l-actions"></div>
		</div>
	</div>`;
}

async function play(g, { noSync = false } = {}) {
	const ctx = { cancelled: false };
	const el = openOverlay({
		kind: "launch",
		label: `Abrindo ${g.name}`,
		game: g,
		blocking: true,
		onBack: () => {
			ctx.cancelled = true;
			closeOverlay(el);
			toast(`Abertura de ${g.name} cancelada. Nada foi alterado.`, "info", "x");
		},
		legend: [["confirm", "Selecionar"], ["back", "Cancelar"]],
		html: launchShell(g, "Preparando o jogo"),
	});
	const step = stepper($(".steps", el));
	const actions = $(".launch-actions", el);
	actions.innerHTML = `<button class="btn btn-glass" data-f data-click="cancel-launch">${icon("x")}<span>Cancelar</span></button>`;
	F.focus($("button", actions));
	const wait = async (ms) => {
		await sleep(ms);
		return !ctx.cancelled;
	};

	if (noSync || g.sync === "off") {
		const s = step(noSync ? "Jogando sem sincronizar: o save não será baixado nem enviado." : "Sincronização desligada para este jogo.");
		s.warn();
	} else {
		const s1 = step("Conferindo a pasta da nuvem");
		if (!(await wait(700))) return;
		if (state.cloudOffline) {
			s1.fail("Pasta da nuvem não encontrada. Confira se o OneDrive está aberto e conectado.");
			sound("error");
			actions.innerHTML = `
				<button class="btn btn-play" data-f data-click="retry-launch">${icon("refresh")}<span>Tentar de novo</span></button>
				<button class="btn btn-glass" data-f data-click="nosync-launch">${icon("cloudOff")}<span>Jogar sem sincronizar</span></button>
				<button class="btn btn-glass" data-f data-click="cancel-launch">${icon("x")}<span>Cancelar</span></button>`;
			F.focus($("button", actions));
			$('[data-click="retry-launch"]', actions).onclick = () => {
				closeOverlay(el);
				play(g);
			};
			$('[data-click="nosync-launch"]', actions).onclick = () => {
				closeOverlay(el);
				play(g, { noSync: true });
			};
			return;
		}
		s1.ok("Pasta da nuvem encontrada (OneDrive)");
		const s2 = step("Comparando o save deste PC com o da nuvem");
		if (!(await wait(800))) return;
		let restore = false;
		if (g.sync === "ok") s2.ok("Os saves já estão iguais");
		else if (g.sync === "local-newer") {
			s2.ok("Este PC tem o save mais novo");
			const s3 = step("Enviando o save para a nuvem");
			if (!(await wait(900))) return;
			s3.ok("Versão nova criada na nuvem e conferida");
			g.versions.unshift({ id: `${g.id}-u${Date.now()}`, at: Date.now(), device: state.pcName, reason: "Antes de jogar", size: 240_000, files: 3 });
		} else if (g.sync === "cloud-newer") {
			s2.ok("A nuvem tem um save mais novo, do NOTEBOOK");
			restore = true;
		} else if (g.sync === "cloud-syncing") {
			s2.wait("Aguardando o OneDrive terminar de baixar o save (2 de 5 arquivos)");
			if (!(await wait(1400))) return;
			s2.wait("Aguardando o OneDrive terminar de baixar o save (4 de 5 arquivos)");
			if (!(await wait(1200))) return;
			s2.ok("Download da nuvem concluído e conferido");
			restore = true;
		} else if (g.sync === "conflict") {
			s2.warn("Os saves mudaram nos dois PCs: escolha qual usar");
			sound("error");
			const choice = await conflictDialog(g);
			if (ctx.cancelled) return;
			if (choice === "later") {
				closeOverlay(el);
				toast(`${g.name} não foi aberto. O conflito continua esperando sua escolha.`, "warn", "alert");
				return;
			}
			const sb = step(choice === "cloud" ? "Guardando o save deste PC como backup" : "Guardando o save da nuvem como versão antiga");
			if (!(await wait(700))) return;
			sb.ok();
			if (choice === "cloud") restore = true;
			else {
				const su = step("Enviando o save deste PC para a nuvem");
				if (!(await wait(800))) return;
				su.ok("Save deste PC agora é a versão atual");
			}
		}
		if (restore) {
			const s3 = step("Guardando um backup do save deste PC");
			if (!(await wait(600))) return;
			s3.ok();
			const s4 = step("Restaurando o save da nuvem");
			if (!(await wait(900))) return;
			s4.ok("Save restaurado e conferido");
		}
		g.sync = "ok";
		const sl = step("Marcando o jogo como aberto neste PC");
		if (!(await wait(400))) return;
		sl.ok();
	}
	const s5 = step(`Abrindo ${g.name}`);
	if (!(await wait(900))) return;
	s5.ok();
	sound("confirm");
	closeOverlay(el);
	openRunning(g, { noSync: noSync || g.sync === "off" });
}

function conflictDialog(g) {
	return new Promise((resolve) => {
		const local = { at: Date.now() - 19 * 3600e3, play: 74, size: 412_000, files: 4 };
		const cloud = { at: Date.now() - 22 * 3600e3, play: 51, size: 398_000, files: 4, device: "NOTEBOOK" };
		let done = false;
		const finish = (v) => {
			if (done) return;
			done = true;
			closeOverlay(el);
			resolve(v);
		};
		const side = (title, ic, d, key) => `
			<button class="conflict-side" data-f data-choice="${key}">
				<span class="cs-head">${icon(ic)}<span>${title}</span></span>
				<ul class="kv">
					<li><span>Alterado</span><b>${dtf.format(d.at)}</b></li>
					<li><span>Sessão</span><b>${playtime(d.play)}</b></li>
					<li><span>Tamanho</span><b>${size(d.size)} · ${d.files} arquivos</b></li>
				</ul>
				<span class="cs-cta">Usar este save</span>
			</button>`;
		const el = openOverlay({
			kind: "conflict",
			label: "Conflito de save",
			blocking: true,
			onBack: () => finish("later"),
			legend: [["confirm", "Escolher"], ["back", "Decidir depois"]],
			html: `<div class="modal modal-wide panel">
				<div class="modal-icon tone-danger">${icon("alert")}</div>
				<h2>Qual save de ${esc(g.name)} você quer usar?</h2>
				<p>O save mudou neste PC e no ${cloud.device} desde a última sincronização. <b>O save não escolhido fica guardado como backup.</b></p>
				<div class="conflict" data-group="conflict">
					${side(`Neste PC (${state.pcName})`, "monitor", local, "local")}
					${side(`Na nuvem (${cloud.device})`, "cloud", cloud, "cloud")}
				</div>
				<div class="modal-actions" data-group="m-actions">
					<button class="btn btn-glass" data-f data-autofocus data-choice="later">Decidir depois (não abrir o jogo)</button>
				</div>
			</div>`,
		});
		el.addEventListener("click", (e) => {
			const c = e.target.closest("[data-choice]");
			if (c) finish(c.dataset.choice);
		});
	});
}

function openRunning(g, { noSync }) {
	const started = Date.now();
	const el = openOverlay({
		kind: "running",
		label: `Jogando ${g.name}`,
		game: g,
		blocking: true,
		onBack: () => toast("Feche o jogo para voltar. O HUB acompanha e envia o save sozinho.", "info", "info"),
		legend: [["confirm", "Selecionar"]],
		html: `<div class="launch running">
			<div class="details-bg dim" style="${g.art ? `background-image:url('${g.art.hero}')` : `--ph:${g.accent}`}" data-ph="${g.art ? "" : "1"}"></div>
			<div class="launch-box">
				<p class="launch-kicker">${icon("gamepad")}<span>Jogando agora</span></p>
				<h1 class="running-title">${esc(g.name)}</h1>
				<p class="running-time" aria-live="off">0:00:00</p>
				<ul class="running-info">
					<li>${icon(noSync ? "cloudOff" : "cloudUp")}<span>${noSync ? "Sem sincronização: o save fica só neste PC." : "O save será enviado para a nuvem quando o jogo fechar."}</span></li>
					${noSync ? "" : `<li>${icon("lock")}<span>Outros PCs veem que você está jogando ${esc(g.name)} aqui.</span></li>`}
					<li>${icon("tray")}<span>No app final, o HUB fica na bandeja e quase não usa memória enquanto você joga.</span></li>
				</ul>
				<div class="launch-actions" data-group="r-actions">
					<button class="btn btn-play" data-f data-autofocus data-click="sim-close">${icon("power")}<span>Simular: o jogo fechou</span></button>
					<button class="btn btn-glass" data-f aria-disabled="true" data-click="toast:O envio manual fica disponível depois que o jogo fechar, para não copiar um save pela metade.">${icon("upload")}<span>Enviar save agora</span></button>
				</div>
				<ol class="steps" aria-live="polite"></ol>
			</div>
		</div>`,
	});
	const timeEl = $(".running-time", el);
	const tick = setInterval(() => {
		const s = Math.floor((Date.now() - started) / 1000);
		timeEl.textContent = `${Math.floor(s / 3600)}:${String(Math.floor(s / 60) % 60).padStart(2, "0")}:${String(s % 60).padStart(2, "0")}`;
	}, 1000);
	$('[data-click="sim-close"]', el).addEventListener("click", async () => {
		clearInterval(tick);
		const minutes = Math.max(1, Math.round((Date.now() - started) / 60000));
		$(".launch-actions", el).innerHTML = "";
		$(".launch-kicker", el).innerHTML = `${icon("check")}<span>Jogo fechado</span>`;
		const step = stepper($(".steps", el));
		g.lastPlayed = Date.now();
		g.playtime += minutes;
		if (noSync) {
			step("O save ficou só neste PC (jogo sem sincronização).").warn();
			await sleep(1200);
		} else {
			const a = step("Esperando o jogo terminar de gravar o save");
			await sleep(1000);
			a.ok("Nenhum processo do jogo continua aberto");
			const b = step("Comparando com a última versão");
			await sleep(700);
			b.ok("3 arquivos mudaram");
			const c = step("Criando versão nova na nuvem");
			await sleep(1000);
			if (state.cloudOffline) {
				c.fail("Pasta da nuvem indisponível. O save ficou guardado neste PC e será enviado quando a nuvem voltar.");
				g.sync = "local-newer";
				sound("error");
				$(".launch-actions", el).innerHTML = `<button class="btn btn-play" data-f data-click="close-top">${icon("check")}<span>Entendi</span></button>`;
				F.focus($(".launch-actions button", el));
				refreshAll();
				return;
			}
			c.ok("Versão nova criada (3 arquivos, 412 KB)");
			const d = step("Conferindo os arquivos enviados");
			await sleep(700);
			d.ok("Arquivos conferidos");
			g.versions.unshift({ id: `${g.id}-c${Date.now()}`, at: Date.now(), device: state.pcName, reason: "Ao fechar o jogo", size: 412_000, files: 3 });
			g.sync = "ok";
			await sleep(500);
		}
		closeOverlay(el);
		state.hero = g;
		if (state.screen === "home" && state.shelf !== 0) setShelf(0, { focusShelf: false });
		refreshAll();
		if (state.screen === "home") F.focus($(`#shelf [data-game="${g.id}"]`));
		toast(noSync ? `Sessão de ${g.name} encerrada (${playtime(minutes)}).` : `Save de ${g.name} enviado para a nuvem.`, noSync ? "info" : "ok", noSync ? "info" : "cloudCheck");
	});
}

async function syncGame(g) {
	if (g.sync === "off") return toast(`A sincronização de ${g.name} está desligada.`, "info", "cloudOff");
	if (state.cloudOffline) return toast("Pasta da nuvem indisponível. Confira se o OneDrive está aberto.", "warn", "cloudOff");
	if (g.sync === "conflict") return toast(`${g.name} tem um conflito. Escolha o save ao jogar.`, "danger", "alert");
	toast(`Sincronizando ${g.name}…`, "info", "refresh");
	await sleep(1000);
	if (g.sync === "local-newer") g.versions.unshift({ id: `${g.id}-s${Date.now()}`, at: Date.now(), device: state.pcName, reason: "Sincronização manual", size: 300_000, files: 3 });
	g.sync = "ok";
	refreshAll();
	for (const o of $$("#overlays .overlay")) o.dispatchEvent(new Event("sgs:refresh"));
	toast(`${g.name} sincronizado.`);
}

async function syncAll() {
	if (state.cloudOffline) return toast("Pasta da nuvem indisponível. Nenhum save foi alterado.", "warn", "cloudOff");
	const list = lib().filter((g) => g.sync !== "off");
	toast(`Sincronizando ${list.length} jogos…`, "info", "refresh");
	await sleep(1400);
	for (const g of list) if (g.sync === "local-newer" || g.sync === "cloud-newer") g.sync = "ok";
	const conflicts = list.filter((g) => g.sync === "conflict");
	refreshAll();
	toast(conflicts.length ? `Pronto. ${conflicts.map((g) => g.name).join(", ")} precisa da sua escolha ao jogar.` : "Todos os saves estão em dia.", conflicts.length ? "warn" : "ok");
}

// ---------------------------------------------------------------------------
// Central rápida
// ---------------------------------------------------------------------------
function openQuick() {
	const list = lib().filter((g) => g.sync !== "off");
	const count = (k) => list.filter((g) => g.sync === k).length;
	const el = openOverlay({
		kind: "quick",
		label: "Central rápida",
		legend: [["confirm", "Selecionar"], ["back", "Fechar"]],
		html: `<div class="sheet sheet-bottom panel">
			<div class="quick" data-group="quick">
				<section class="q-card">
					<h3>${icon("cloud")}Nuvem</h3>
					<p class="q-big">${state.cloudOffline ? '<span class="tone-warn">Indisponível</span>' : `${count("ok")} de ${list.length} em dia`}</p>
					<p class="dim">${count("conflict")} conflito · ${count("cloud-syncing")} baixando · ${count("local-newer") + count("cloud-newer")} para sincronizar</p>
					<button class="btn btn-play btn-sm" data-f data-autofocus data-click="sync-all">${icon("refresh")}<span>Sincronizar tudo agora</span></button>
				</section>
				<section class="q-card q-wide">
					<h3>${icon("bell")}Notificações</h3>
					<ul class="notes">${state.notifications.slice(0, 4).map((n) => `<li><span class="tone-${n.tone}">●</span><span>${esc(n.text)}</span><span class="dim">${ago(n.at)}</span></li>`).join("")}</ul>
				</section>
				<section class="q-card">
					<h3>${icon("monitor")}${esc(state.pcName)}</h3>
					<div class="q-row"><span>Sons</span>${toggle("sounds", state.sounds, "Sons de navegação")}</div>
					<button class="btn btn-glass btn-sm" data-f data-click="toast:No app final a janela some e o HUB fica na bandeja.">${icon("tray")}<span>Minimizar para a bandeja</span></button>
				</section>
				<section class="q-card q-proto">
					<h3>${icon("chip")}Ferramentas do protótipo</h3>
					<div class="q-row"><span>Simular nuvem indisponível</span>${toggle("cloudOffline", state.cloudOffline, "Simular nuvem indisponível")}</div>
					<div class="q-row"><span>Simular biblioteca vazia</span>${toggle("emptyLibrary", state.emptyLibrary, "Simular biblioteca vazia")}</div>
				</section>
			</div>
		</div>`,
	});
	return el;
}

// ---------------------------------------------------------------------------
// Atalhos
// ---------------------------------------------------------------------------
function openHelp() {
	const pad = state.device === "ps" || state.glyphPref === "ps" ? "ps" : "xbox";
	const rows = [
		["up", "Mover o foco", "Setas", "Direcional / analógico"],
		["confirm", "Selecionar / Jogar"],
		["back", "Voltar / Fechar"],
		["options", "Opções do jogo"],
		["filter", "Filtros"],
		["search", "Buscar (também Ctrl+K)"],
		["menu", "Menu principal"],
		["prevTab", "Seção ou aba anterior"],
		["nextTab", "Próxima seção ou aba"],
		["pageDown", "Pular vários itens"],
		["quick", "Central rápida"],
		["help", "Esta lista de atalhos"],
		["home", "Ir ao Início"],
		["library", "Ir à Biblioteca"],
		["addGame", "Adicionar jogo"],
		["details", "Saves e conquistas do jogo"],
		["settings", "Configurações"],
	];
	openOverlay({
		kind: "help",
		label: "Atalhos",
		legend: [["back", "Fechar"]],
		html: `<div class="modal modal-wide panel">
			<h2>${icon("keyboard")}Atalhos</h2>
			<div class="help-table" data-group="help" tabindex="-1">
				<div class="help-row help-headrow"><span>Ação</span><span>Teclado</span><span>Controle</span></div>
				${rows
					.map(
						([a, l, k, p]) => `<div class="help-row" data-f tabindex="0"><span>${esc(l)}</span><span>${k ? `<kbd class="glyph">${k}</kbd>` : glyph(a, "keyboard")}</span><span>${p ? `<span class="dim">${p}</span>` : glyph(a, pad) || '<span class="dim">—</span>'}</span></div>`,
					)
					.join("")}
			</div>
			<p class="dim">No app final, o botão Guide/PS abre a central rápida mesmo com o jogo em primeiro plano.</p>
		</div>`,
	});
}

// ---------------------------------------------------------------------------
// Adicionar jogo (assistente)
// ---------------------------------------------------------------------------
function openAddGame(prefill) {
	const w = { step: prefill ? 1 : 0, exe: prefill ? `C:\\Games\\${prefill.replace(/[:]/g, "")}\\game.exe` : "", name: prefill ?? "Hollow Knight: Silksong" };
	const STEPS = ["Executável", "Identificar", "Caminhos do save", "Concluir"];
	const paths = () => [
		{ path: `<winAppData>/${w.name}/saves`, found: true, files: 3, size: 182_000 },
		{ path: "<base>/config.ini", found: false },
	];
	const body = () => {
		switch (w.step) {
			case 0:
				return `<h2>Qual é o executável do jogo?</h2>
					<p class="dim">Escolha o arquivo .exe que abre o jogo. Se o jogo usa um launcher próprio, o HUB acompanha os processos filhos.</p>
					<div class="pick">${w.exe ? `<code>${esc(w.exe)}</code>` : '<span class="dim">Nenhum arquivo escolhido</span>'}</div>
					<div class="row-actions" data-group="w-body">
						<button class="btn btn-glass" data-f data-autofocus data-w="pick">${icon("file")}<span>Escolher executável…</span></button>
						<button class="btn btn-glass" data-f data-click="scan">${icon("folder")}<span>Procurar jogos numa pasta…</span></button>
					</div>`;
			case 1:
				return `<h2>É este o jogo?</h2>
					<div class="ident"><span class="ph result-ph" style="--ph:#5f8cff"><span>${esc(initials(w.name))}</span></span>
						<div><div class="status-title">${esc(w.name)}</div><div class="dim">Encontrado no catálogo pelo nome da pasta · 2 caminhos de save conhecidos</div></div></div>
					<div class="row-actions" data-group="w-body"><button class="btn btn-glass btn-sm" data-f data-click="search-from-wizard">${icon("search")}<span>Não é este? Buscar</span></button></div>`;
			case 2:
				return `<h2>Onde o jogo guarda o save?</h2>
					<p class="dim">Estes caminhos vêm do catálogo. Confira se os arquivos existem neste PC.</p>
					<ul class="paths">${paths().map((p) => `<li class="${p.found ? "" : "is-missing"}">${icon(p.found ? "file" : "alert")}<code>${esc(p.path)}</code><span class="dim">${p.found ? `${p.files} arquivos · ${size(p.size)}` : "Não encontrado (pode ser criado depois)"}</span></li>`).join("")}</ul>
					<div class="row-actions" data-group="w-body">${act("toast:No app final abre o seletor de pasta ou arquivo.", "Adicionar caminho…", "plus")}</div>`;
			default:
				return `<h2>Tudo pronto</h2>
					<ul class="kv"><li><span>Jogo</span><b>${esc(w.name)}</b></li><li><span>Executável</span><b><code>${esc(w.exe)}</code></b></li><li><span>Save</span><b>1 caminho encontrado · 182 KB</b></li><li><span>Nuvem</span><b><code>${esc(state.cloudPath)}</code></b></li></ul>
					<p class="dim">O primeiro backup copia o save atual para a nuvem. Nada neste PC é alterado.</p>`;
		}
	};
	const foot = () => `
		<button class="btn btn-glass" data-f data-w="${w.step === 0 ? "cancel" : "prev"}">${w.step === 0 ? "Cancelar" : "Voltar"}</button>
		<button class="btn btn-play" data-f data-w="next" ${w.step === 0 && !w.exe ? 'aria-disabled="true"' : ""}>${w.step === 3 ? `${icon("cloudUp")}<span>Fazer o primeiro backup e concluir</span>` : "<span>Próximo</span>"}</button>`;
	const el = openOverlay({
		kind: "wizard",
		label: "Adicionar jogo",
		legend: [["confirm", "Selecionar"], ["back", "Voltar"]],
		onBack: () => (w.step === 0 ? closeOverlay(el) : go(w.step - 1)),
		html: `<div class="modal modal-wide panel">
			<ol class="wsteps">${STEPS.map((s, i) => `<li data-i="${i}">${s}</li>`).join("")}</ol>
			<div class="wbody"></div>
			<div class="modal-actions" data-group="w-foot"></div>
		</div>`,
	});
	function go(i) {
		w.step = i;
		$(".wbody", el).innerHTML = body();
		$(".modal-actions", el).innerHTML = foot();
		for (const li of $$(".wsteps li", el)) li.className = +li.dataset.i < i ? "done" : +li.dataset.i === i ? "now" : "";
		F.focus($("[data-autofocus]", el) ?? $('[data-w="next"]', el), { scroll: false });
	}
	el.addEventListener("click", async (e) => {
		const b = e.target.closest("[data-w]");
		if (!b) return;
		const a = b.dataset.w;
		if (a === "pick") {
			w.exe = "C:\\Games\\Hollow Knight Silksong\\Hollow Knight Silksong.exe";
			go(0);
			F.focus($('[data-w="next"]', el));
		} else if (a === "cancel") closeOverlay(el);
		else if (a === "prev") go(w.step - 1);
		else if (a === "next") {
			if (b.getAttribute("aria-disabled") === "true") return toast("Escolha o executável do jogo para continuar.", "warn", "file");
			if (w.step < 3) return go(w.step + 1);
			b.setAttribute("aria-disabled", "true");
			b.innerHTML = `<span class="spinner"></span><span>Fazendo o primeiro backup…</span>`;
			await sleep(1300);
			const g = {
				id: `novo-${Date.now()}`,
				appid: null,
				name: w.name,
				dev: "Adicionado por você",
				year: new Date().getFullYear(),
				tags: ["Novo"],
				desc: "Jogo adicionado pelo assistente.",
				accent: "#5f8cff",
				playtime: 0,
				lastPlayed: null,
				added: Date.now(),
				source: "Outros",
				sync: "ok",
				ach: null,
				isNew: true,
				art: null,
				installPath: w.exe.replace(/\\[^\\]+$/, ""),
				savePaths: paths(),
				versions: [{ id: `n${Date.now()}`, at: Date.now(), device: state.pcName, reason: "Primeiro backup", size: 182_000, files: 3 }],
			};
			games.unshift(g);
			closeOverlay(el);
			state.hero = g;
			if (state.screen === "home") setShelf(2);
			refreshAll();
			toast(`${g.name} adicionado. Primeiro backup criado na nuvem.`);
		}
	});
	go(w.step);
}

// ---------------------------------------------------------------------------
// Ações
// ---------------------------------------------------------------------------
const NAV = new Set(["home", "library", "settings", "menu", "search", "filter", "quick", "help", "addGame"]);
const KIND_OF = { menu: "menu", search: "search", filter: "filters", quick: "quick", help: "help" };

function contextGame() {
	const cur = F.current();
	const id = cur?.dataset.game ?? cur?.closest("[data-game-ctx]")?.dataset.gameCtx ?? F.topLayer()?.dataset.gameCtx;
	if (id) return byId(id);
	return state.screen === "home" && !F.topLayer() ? state.hero : null;
}

function back() {
	const top = F.topLayer();
	if (top) {
		const m = meta.get(top);
		if (m.onBack) m.onBack();
		else if (!m.blocking) closeOverlay(top);
		sound("back");
		return;
	}
	if (state.screen !== "home") {
		sound("back");
		return showScreen("home");
	}
	const shelf = $("#shelf");
	if (shelf && !shelf.contains(F.current())) F.focusGroup(shelf);
}

function tab(d) {
	const top = F.topLayer();
	if (top) return meta.get(top).onTab?.(d);
	sound("move");
	if (state.screen === "home") setShelf(state.shelf + d, { focusShelf: $("#shelf").contains(F.current()) || !F.current() });
	else if (state.screen === "library") {
		state.libTab = (state.libTab + d + LIB_TABS.length) % LIB_TABS.length;
		const inGrid = $("#screen-library .lib-grid")?.contains(F.current());
		renderLibrary();
		F.focus(inGrid ? ($("#screen-library .card") ?? $(`[data-libtab="${state.libTab}"]`)) : $(`[data-libtab="${state.libTab}"]`), { scroll: false });
	} else {
		const i = SECTIONS.findIndex((s) => s.id === state.settings);
		const next = SECTIONS[(i + d + SECTIONS.length) % SECTIONS.length];
		F.focus($(`.set-nav-item[data-section="${next.id}"]`));
	}
	updateLegend();
}

function page(d) {
	const cur = F.current();
	const horizontal = !!cur?.closest('[data-scroll="x"]');
	for (let i = 0; i < (horizontal ? 5 : 3); i++) F.move(horizontal ? (d > 0 ? "right" : "left") : d > 0 ? "down" : "up");
}

function run(action) {
	const top = F.topLayer();
	if (NAV.has(action) && top) {
		const m = meta.get(top);
		if (m.blocking) return;
		if (KIND_OF[action] && m.kind === KIND_OF[action]) return back();
		closeAll();
	}
	switch (action) {
		case "up":
		case "down":
		case "left":
		case "right": {
			const cur = F.current();
			if (cur?.matches("input") && (action === "left" || action === "right")) return;
			if (F.move(action)) sound("move");
			return;
		}
		case "confirm": {
			const cur = F.current();
			if (!cur) return F.focusFirst();
			if (cur.matches("input")) return F.move("down");
			sound("confirm");
			cur.click();
			return;
		}
		case "back":
			return back();
		case "options": {
			const g = contextGame();
			if (g && topKind() !== "options") openOptions(g);
			return;
		}
		case "details": {
			const g = contextGame();
			if (g && !top) openDetails(g);
			return;
		}
		case "prevTab":
			return tab(-1);
		case "nextTab":
			return tab(1);
		case "pageUp":
			return page(-1);
		case "pageDown":
			return page(1);
		case "home":
			return showScreen("home");
		case "library":
			return showScreen("library");
		case "settings":
			return showScreen("settings");
		case "menu":
			return openMenu();
		case "search":
			return openSearch();
		case "filter":
			return openFilters();
		case "quick":
			return openQuick();
		case "help":
			return openHelp();
		case "addGame":
			return openAddGame();
	}
}

// Cliques: [data-action] chama uma ação; [data-click] chama um comando local.
document.addEventListener("click", (e) => {
	const legendBtn = e.target.closest(".legend-item");
	if (legendBtn) return run(legendBtn.dataset.action);
	const a = e.target.closest("[data-action]");
	if (a && !a.closest(".legend")) return run(a.dataset.action);

	const tabBtn = e.target.closest("[data-shelf]");
	if (tabBtn) return setShelf(+tabBtn.dataset.shelf, { focusShelf: false });
	const libTab = e.target.closest("[data-libtab]");
	if (libTab) {
		state.libTab = +libTab.dataset.libtab;
		renderLibrary();
		return F.focus($(`[data-libtab="${state.libTab}"]`), { scroll: false });
	}
	const sec = e.target.closest("[data-section]");
	if (sec) return F.focus($("#set-body [data-f]") ?? sec);

	const tg = e.target.closest("[data-toggle]");
	if (tg) return onToggle(tg);
	const sg = e.target.closest("[data-seg]");
	if (sg) return onSeg(sg);

	const cap = e.target.closest(".cap[data-game]");
	if (cap) return play(byId(cap.dataset.game));
	const card = e.target.closest(".card[data-game]");
	if (card) return openDetails(byId(card.dataset.game));

	const c = e.target.closest("[data-click]");
	if (!c) return;
	const cmd = c.dataset.click;
	const g = contextGame();
	if (cmd.startsWith("toast:")) return toast(cmd.slice(6), "info", "info");
	switch (cmd) {
		case "play":
			return play(state.hero);
		case "details":
			return openDetails(state.hero);
		case "options":
			return openOptions(state.hero);
		case "play-ctx": {
			const gg = byId(F.topLayer()?.dataset.gameCtx);
			closeOverlay();
			return play(gg);
		}
		case "options-ctx":
			return openOptions(byId(F.topLayer()?.dataset.gameCtx));
		case "sync-game":
			return g && syncGame(g);
		case "sync-all":
			closeOverlay();
			return syncAll();
		case "close-top":
			return closeOverlay();
		case "cancel-launch":
			return back();
		case "clear-filters":
			return clearFilters();
		case "sort": {
			const keys = Object.keys(SORTS);
			state.filters.sort = keys[(keys.indexOf(state.filters.sort) + 1) % keys.length];
			return refreshLibrary();
		}
		case "scan":
			return toast("Procurando jogos em C:\\Games… No app final aparece a lista para você escolher.", "info", "folder");
		case "verify":
			return state.cloudOffline
				? toast("Pasta não encontrada. Confira se o OneDrive está aberto e conectado.", "warn", "cloudOff")
				: toast("Pasta encontrada, com permissão de escrita e 18 jogos guardados.", "ok", "check");
		case "help":
			return openHelp();
		case "search-from-wizard":
			closeOverlay();
			return openSearch();
	}
});

function onToggle(b) {
	const on = b.getAttribute("aria-checked") !== "true";
	b.setAttribute("aria-checked", String(on));
	const key = b.dataset.toggle;
	if (key === "sounds") state.sounds = on;
	else if (key === "glass") root.dataset.glass = on ? "on" : "off";
	else if (key === "cloudOffline") {
		state.cloudOffline = on;
		refreshAll();
	} else if (key === "emptyLibrary") {
		state.emptyLibrary = on;
		state.hero = null;
		refreshAll();
	}
}

function onSeg(b) {
	const key = b.dataset.seg;
	const v = b.dataset.value;
	for (const o of $$(`[data-seg="${key}"]`)) o.setAttribute("aria-checked", String(o === b));
	if (key === "glyphPref") {
		state.glyphPref = v;
		app.dataset.input = effectiveDevice();
		updateLegend();
	} else if (key === "motion") root.dataset.motion = v;
	else if (key === "accent") {
		state.accent = v;
		applyAccent(state.hero);
	} else if (key === "retention") state.retention = +v;
}

// ---------------------------------------------------------------------------
// Início do app
// ---------------------------------------------------------------------------
F.init($("#base"));
// Quando o elemento de origem some (ex.: lista redesenhada), o foco vai para o ponto principal da tela.
F.setFallback(() => {
	if (state.screen === "home") F.focus($("#shelf .cap[data-game]") ?? $("#hero [data-f]") ?? $("#shelf [data-f]"));
	else if (state.screen === "library") F.focus($("#screen-library .card") ?? F.focusables($("#screen-library"))[0]);
	else F.focus($(`.set-nav-item[data-section="${state.settings}"]`));
});
initInput({
	onAction: run,
	onDevice: (d) => {
		if (state.device === d) return;
		state.device = d;
		app.dataset.input = effectiveDevice();
		updateLegend();
	},
});
renderRail();
renderTopbar();
renderHome();
renderSettings();
showScreen("home");
F.focus($("#shelf [data-f]"));
