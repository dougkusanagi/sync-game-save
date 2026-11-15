# Memória do projeto

- A chave privada que assina as atualizações fica em `C:\Users\dl_ag\.tauri\sync-game-save.key` (sem senha). Um build de release precisa de `TAURI_SIGNING_PRIVATE_KEY` apontando para ela; a chave pública está em `src-tauri/tauri.conf.json`.
- A chave privada do atualizador da versão Go (Ed25519, PEM, sem senha) fica em `C:\Users\dl_ag\.sgs\updater-ed25519.pem`; a pública está em `build/updater-public.pem`. Gere o manifesto de cada release com `go run ./cmd/assinar manifesto` (ADR 0009). Se a privada se perder, as instalações existentes não conseguem atualizar pelo app.
