// Motor de navegação espacial.
// - Elementos focáveis têm [data-f]; grupos ([data-group]) lembram o último item.
// - Só o elemento do topo da pilha de camadas (tela base ou overlay) recebe foco.
// - O mouse foca ao passar por cima; Tab também funciona (focusin).

let base = null;
let cur = null;
const layers = [];
const memory = new WeakMap();

export const reducedMotion = () =>
	document.documentElement.dataset.motion === "reduced" ||
	(document.documentElement.dataset.motion !== "full" &&
		matchMedia("(prefers-reduced-motion: reduce)").matches);

export function init(baseEl) {
	base = baseEl;
	document.addEventListener("focusin", (e) => {
		const el = e.target.closest?.("[data-f]");
		if (el && el !== cur && scope().contains(el)) focus(el, { scroll: false });
	});
	let lastX = -1;
	let lastY = -1;
	document.addEventListener("pointermove", (e) => {
		if (e.pointerType !== "mouse" || (e.clientX === lastX && e.clientY === lastY)) return;
		lastX = e.clientX;
		lastY = e.clientY;
		const el = e.target.closest?.("[data-f]");
		if (el && el !== cur && scope().contains(el) && !el.matches("input")) focus(el, { scroll: false });
	});
}

export const scope = () => layers.at(-1)?.el ?? base;
export const current = () => cur;

function visible(el) {
	if (el.closest("[hidden]")) return false;
	const r = el.getBoundingClientRect();
	return r.width > 0 && r.height > 0;
}

const groupOf = (el) => el.closest("[data-group]");

export function focusables(root = scope()) {
	return [...root.querySelectorAll("[data-f]")].filter(visible);
}

export function focus(el, { scroll = true } = {}) {
	if (!el) return;
	if (cur && cur !== el) cur.classList.remove("is-focused");
	cur = el;
	el.classList.add("is-focused");
	if (document.activeElement !== el) el.focus({ preventScroll: true });
	const g = groupOf(el);
	if (g) memory.set(g, el);
	if (scroll) scrollTo(el);
	document.dispatchEvent(new CustomEvent("sgs:focus", { detail: el }));
}

export function focusFirst(root = scope()) {
	const auto = focusables(root).find((e) => e.hasAttribute("data-autofocus"));
	focus(auto ?? focusables(root)[0]);
}

/** Foca o último item lembrado do grupo, ou o primeiro. */
export function focusGroup(group) {
	if (!group) return;
	const m = memory.get(group);
	focus(m && group.contains(m) && visible(m) ? m : focusables(group)[0]);
}

function scrollTo(el) {
	const behavior = reducedMotion() ? "auto" : "smooth";
	const shelf = el.closest('[data-scroll="x"]');
	if (shelf) {
		const pad = Number.parseFloat(getComputedStyle(shelf).paddingLeft) || 0;
		shelf.scrollTo({ left: Math.max(0, el.offsetLeft - pad), behavior });
		return;
	}
	el.scrollIntoView({ block: "nearest", inline: "nearest", behavior });
}

function gap(a1, a2, b1, b2) {
	if (b2 < a1) return a1 - b2;
	if (b1 > a2) return b1 - a2;
	return 0;
}

function score(a, b, dir) {
	const acx = (a.left + a.right) / 2;
	const acy = (a.top + a.bottom) / 2;
	const bcx = (b.left + b.right) / 2;
	const bcy = (b.top + b.bottom) / 2;
	let primary;
	let ortho;
	let off;
	switch (dir) {
		case "right":
			if (bcx <= acx + 1 || b.left < a.left + 1) return Number.POSITIVE_INFINITY;
			primary = Math.max(0, b.left - a.right);
			ortho = gap(a.top, a.bottom, b.top, b.bottom);
			off = Math.abs(bcy - acy);
			break;
		case "left":
			if (bcx >= acx - 1 || b.right > a.right - 1) return Number.POSITIVE_INFINITY;
			primary = Math.max(0, a.left - b.right);
			ortho = gap(a.top, a.bottom, b.top, b.bottom);
			off = Math.abs(bcy - acy);
			break;
		case "down":
			if (bcy <= acy + 1 || b.top < a.top + 1) return Number.POSITIVE_INFINITY;
			primary = Math.max(0, b.top - a.bottom);
			ortho = gap(a.left, a.right, b.left, b.right);
			off = Math.abs(bcx - acx);
			break;
		default:
			if (bcy >= acy - 1 || b.bottom > a.bottom - 1) return Number.POSITIVE_INFINITY;
			primary = Math.max(0, a.top - b.bottom);
			ortho = gap(a.left, a.right, b.left, b.right);
			off = Math.abs(bcx - acx);
	}
	return primary + ortho * 4 + off * 0.2;
}

/** Move o foco na direção pedida. Devolve false se não houver destino. */
export function move(dir) {
	const sc = scope();
	if (!cur || !sc.contains(cur) || !visible(cur)) {
		focusFirst(sc);
		return true;
	}
	const a = cur.getBoundingClientRect();
	let best = null;
	let bestScore = Number.POSITIVE_INFINITY;
	for (const el of focusables(sc)) {
		if (el === cur) continue;
		const s = score(a, el.getBoundingClientRect(), dir);
		if (s < bestScore) {
			bestScore = s;
			best = el;
		}
	}
	if (!best) return false;
	const g = groupOf(best);
	if (g && g !== groupOf(cur) && !g.hasAttribute("data-nomem")) {
		const m = memory.get(g);
		if (m && m !== best && g.contains(m) && visible(m) && score(a, m.getBoundingClientRect(), dir) < Number.POSITIVE_INFINITY) best = m;
	}
	focus(best);
	return true;
}

// Seletor que reencontra o "mesmo" elemento depois que a tela for redesenhada.
const KEY_ATTRS = ["game", "rail", "section", "libtab", "action", "click", "toggle"];
function keyOf(el) {
	if (!el) return null;
	if (el.id) return `#${CSS.escape(el.id)}`;
	for (const a of KEY_ATTRS) if (el.dataset[a]) return `[data-${a}="${CSS.escape(el.dataset[a])}"]`;
	return null;
}

let fallback = () => focusFirst(scope());
export const setFallback = (fn) => {
	fallback = fn;
};

/** Abre uma camada (overlay). O foco volta ao elemento anterior ao fechar. */
export function pushLayer(el, { onClose } = {}) {
	layers.push({ el, returnTo: cur, returnKey: keyOf(cur), onClose });
	focusFirst(el);
}

export function popLayer() {
	const layer = layers.pop();
	if (!layer) return null;
	layer.onClose?.();
	const sc = scope();
	let target = layer.returnTo;
	if (!target?.isConnected && layer.returnKey) target = sc.querySelector(`${layer.returnKey}[data-f]`);
	if (target && sc.contains(target) && visible(target)) focus(target, { scroll: false });
	else if (layers.length) focusFirst(sc);
	else fallback();
	return layer.el;
}

export const topLayer = () => layers.at(-1)?.el ?? null;
export const layerCount = () => layers.length;
