import { defineConfig } from "vite-plus";

export default defineConfig({
	// Não herdar o postcss.config.js da raiz do repositório (projeto Electron antigo).
	css: { postcss: { plugins: [] } },
	server: { port: 5199 },
});
