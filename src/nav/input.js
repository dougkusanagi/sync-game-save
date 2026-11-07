// Teclado e controle viram "ações" únicas. Também detecta o último dispositivo
// usado (teclado, Xbox ou PlayStation) para trocar a legenda de botões.
// No app final o controle será lido no Rust (gilrs), para funcionar sem foco na janela.

const KEYS = {
	ArrowUp: "up",
	ArrowDown: "down",
	ArrowLeft: "left",
	ArrowRight: "right",
	Enter: "confirm",
	" ": "confirm",
	Escape: "back",
	Backspace: "back",
	f: "filter",
	"/": "search",
	m: "menu",
	q: "prevTab",
	e: "nextTab",
	o: "options",
	ContextMenu: "options",
	F1: "quick",
	"?": "help",
	g: "home",
	l: "library",
	n: "addGame",
	h: "details",
	PageUp: "pageUp",
	PageDown: "pageDown",
};

// Mapeamento "standard" da Gamepad API.
const PAD = {
	0: "confirm",
	1: "back",
	2: "filter",
	3: "options",
	4: "prevTab",
	5: "nextTab",
	6: "pageUp",
	7: "pageDown",
	8: "search",
	9: "menu",
	10: "help",
	11: "quick",
	12: "up",
	13: "down",
	14: "left",
	15: "right",
	16: "quick",
};
const REPEATS = new Set(["up", "down", "left", "right", "pageUp", "pageDown"]);
const FIRST_DELAY = 350;
const REPEAT_EVERY = 80;

// `gamepad: false` desliga a leitura do controle pelo navegador (no app, o Rust lê o controle).
export function initInput({ onAction, onDevice, gamepad = true }) {
	document.addEventListener("keydown", (e) => {
		const inInput = e.target.matches?.("input, textarea");
		let action = null;
		if ((e.ctrlKey || e.metaKey) && (e.key === "k" || e.key === "f")) action = "search";
		else if ((e.ctrlKey || e.metaKey) && e.key === ",") action = "settings";
		else if (e.ctrlKey || e.metaKey || e.altKey) return;
		else if (inInput) {
			if (["Escape", "ArrowUp", "ArrowDown", "Enter", "F1"].includes(e.key)) action = KEYS[e.key];
		} else {
			action = KEYS[e.key] ?? KEYS[e.key.toLowerCase()];
		}
		if (!action) return;
		if (e.key === "Tab") return;
		onDevice("keyboard");
		e.preventDefault();
		onAction(action, { repeat: e.repeat });
	});
	document.addEventListener("pointerdown", (e) => {
		if (e.pointerType === "mouse") onDevice("keyboard");
	});
	// Botão "voltar" do mouse.
	document.addEventListener("mouseup", (e) => {
		if (e.button === 3) {
			e.preventDefault();
			onAction("back", {});
		}
	});

	const held = new Map();
	function press(action, t, kind) {
		const h = held.get(action);
		if (!h) {
			held.set(action, { next: t + FIRST_DELAY });
			onDevice(kind);
			onAction(action, { repeat: false });
		} else if (REPEATS.has(action) && t >= h.next) {
			h.next = t + REPEAT_EVERY;
			onAction(action, { repeat: true });
		}
	}

	function loop(t) {
		const pads = navigator.getGamepads?.() ?? [];
		const active = new Set();
		for (const pad of pads) {
			if (!pad) continue;
			const kind = /054c|dualsense|dualshock|wireless controller/i.test(pad.id) ? "ps" : "xbox";
			pad.buttons.forEach((b, i) => {
				if (b.pressed && PAD[i]) {
					active.add(PAD[i]);
					press(PAD[i], t, kind);
				}
			});
			const [x = 0, y = 0] = pad.axes;
			const dirs = [];
			if (y < -0.5) dirs.push("up");
			if (y > 0.5) dirs.push("down");
			if (x < -0.5) dirs.push("left");
			if (x > 0.5) dirs.push("right");
			for (const d of dirs) {
				active.add(d);
				press(d, t, kind);
			}
		}
		for (const a of [...held.keys()]) if (!active.has(a)) held.delete(a);
		requestAnimationFrame(loop);
	}
	if (gamepad) requestAnimationFrame(loop);
}

// Rótulos de cada ação por dispositivo.
export const GLYPHS = {
	keyboard: {
		confirm: "Enter",
		back: "Esc",
		filter: "F",
		options: "O",
		search: "/",
		menu: "M",
		prevTab: "Q",
		nextTab: "E",
		quick: "F1",
		help: "?",
		home: "G",
		library: "L",
		addGame: "N",
		details: "H",
		settings: "Ctrl ,",
		pageUp: "PgUp",
		pageDown: "PgDn",
	},
	xbox: {
		confirm: "A",
		back: "B",
		filter: "X",
		options: "Y",
		search: "⧉",
		menu: "☰",
		prevTab: "LB",
		nextTab: "RB",
		quick: "R3",
		help: "L3",
		pageUp: "LT",
		pageDown: "RT",
	},
	ps: {
		confirm: "✕",
		back: "○",
		filter: "□",
		options: "△",
		search: "CREATE",
		menu: "OPTIONS",
		prevTab: "L1",
		nextTab: "R1",
		quick: "R3",
		help: "L3",
		pageUp: "L2",
		pageDown: "R2",
	},
};
