import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// https://vite.dev/config/
export default defineConfig(() => ({
  plugins: [react()],

  // O `wails3 dev` espera o Vite numa porta fixa e falha se ela estiver ocupada.
  clearScreen: false,
  server: {
    port: 1420,
    strictPort: true,
    watch: {
      // O backend Go e os arquivos gerados não fazem parte do frontend.
      ignored: ["**/bin/**", "**/bindings/**", "**/internal/**", "**/*.go"],
    },
  },
}));
