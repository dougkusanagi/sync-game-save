# sync-game-save

Launcher/HUB de jogos para PC que sincroniza o save com uma pasta de nuvem (OneDrive, Google Drive etc.) ao abrir e ao fechar cada jogo, sem perder saves.

## Estado

Em reescrita para Tauri 2 + React (ver [plano](docs/plans/launcher-hub.md) e [ADR 0001](docs/adr/0001-stack-tauri-react.md)). O código Electron foi removido. Por enquanto existe o núcleo de sync em Rust (`src-tauri/crates/sgs-core`) e o protótipo da interface.

## Protótipo da interface

```bash
cd prototype
bun install
bun run dev
```

Abra `http://localhost:5199`. Detalhes em [prototype/README.md](prototype/README.md).

## Testar o núcleo de sync

```bash
cd src-tauri
cargo test
```
