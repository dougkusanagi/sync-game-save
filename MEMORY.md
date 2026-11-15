# Memória do projeto

- A chave minisign do atualizador do Tauri (v1.x) fica em `C:\Users\dl_ag\.tauri\sync-game-save.key` (sem senha); a pública está em `src-tauri/tauri.conf.json` no `main`. Só a versão de transição da v1.x ao app Go precisa dela (`TAURI_SIGNING_PRIVATE_KEY` no build do Tauri).
- `wails3 package` precisa do `makensis` no PATH; existe um NSIS 3.13 em `C:\Users\dl_ag\AppData\Local\mygo\nsis-3.13\Bin`.
- Testes do app com WebView real (`bun run e2e:wails`) abrem a janela fora da tela; janela oculta (`Hidden`) não renderiza e trava capturas.
- A chave privada do atualizador da versão Go (Ed25519, PEM, sem senha) fica em `C:\Users\dl_ag\.sgs\updater-ed25519.pem`; a pública está em `build/updater-public.pem`. Gere o manifesto de cada release com `go run ./cmd/assinar manifesto` (ADR 0009). Se a privada se perder, as instalações existentes não conseguem atualizar pelo app.
