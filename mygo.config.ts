import { defineConfig } from "mygo-cli";

// Prova de conceito da migração para MyGo (docs/plans/migracao-mygo.md).
export default defineConfig({
  name: "Sync Game Save",
  identifier: "com.syncgamesave.app",
  version: "1.0.0",
  devUrl: "http://localhost:1420",
  devCommand: "bun run dev",
  buildCommand: "bun run build",
  frontendDist: "dist",
  bindings: "src/mygo.ts",
  out: "build",
});
