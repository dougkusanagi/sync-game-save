# Memória do projeto

- A chave minisign do atualizador do Tauri (v1.x) fica em `C:\Users\dl_ag\.tauri\sync-game-save.key` (sem senha); a pública está em `src-tauri/tauri.conf.json` no `main`. Só a versão de transição da v1.x ao app Go precisa dela (`TAURI_SIGNING_PRIVATE_KEY` no build do Tauri).
- `wails3 package` precisa do `makensis` no PATH; existe um NSIS 3.13 em `C:\Users\dl_ag\AppData\Local\mygo\nsis-3.13\Bin`.
- Testes do app com WebView real (`bun run e2e:wails`) abrem a janela fora da tela; janela oculta (`Hidden`) não renderiza e trava capturas.
- A chave privada do atualizador da versão Go (Ed25519, PEM, sem senha) fica em `C:\Users\dl_ag\.sgs\updater-ed25519.pem`; a pública está em `build/updater-public.pem`. Gere o manifesto de cada release com `go run ./cmd/assinar manifesto` (ADR 0009). Se a privada se perder, as instalações existentes não conseguem atualizar pelo app.
- A migração para o Wails (ADR 0008) terminou sem estes testes, que exigem máquina ou hardware: AppImage em Ubuntu real (só testado no WSL, com `gtk3`), leitura de controle físico (XInput e evdev, só testados sem hardware) e ver o toast aparecer (o aviso foi entregue ao Windows e está no histórico de notificações, `wpndatabase.db-wal`, mas um app em tela cheia o escondeu).
- A instalação e a desinstalação silenciosas (`/S`) do instalador NSIS funcionam; o instalador por usuário usa `%LOCALAPPDATA%\Sync Game Save`. Teste de toast: provoque um erro em `Jogar` (executável inexistente) com o build `sgs_e2e` e uma pasta de dados temporária.
