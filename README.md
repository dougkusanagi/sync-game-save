# sync-game-save

Launcher/HUB de jogos para PC que sincroniza o save com uma pasta de nuvem (OneDrive, Google Drive etc.) ao abrir e ao fechar cada jogo, sem perder saves.

## Estado

Em reescrita para Tauri 2 + React (ver [plano](docs/plans/launcher-hub.md) e [ADR 0001](docs/adr/0001-stack-tauri-react.md)). A Fase 1 está pronta: núcleo de sync em Rust (`src-tauri/crates/sgs-core`), catálogo de jogos (`sgs-catalog`), CLI `sgs` (`sgs-cli`) e um app Tauri com tela mínima. A interface final ainda está no protótipo.

## Protótipo da interface

```bash
cd prototype
bun install
bun run dev
```

Abra `http://localhost:5199`. Detalhes em [prototype/README.md](prototype/README.md).

## Rodar o app e testar

```bash
bun install
bun run tauri dev          # app Tauri + React
cd src-tauri && cargo test --workspace   # núcleo, catálogo e CLI
```

## CLI `sgs`

```bash
cd src-tauri
cargo run -p sgs-cli -- help
cargo run -p sgs-cli -- run --sync-dir <pasta da nuvem> --game "Nome" --save-dir <pasta do save> -- <jogo.exe>
```

O `run` sincroniza antes de abrir, abre o programa, espera ele fechar e envia o save. Uma queda ou um erro do jogo não perde o save: o formato da pasta da nuvem está no [ADR 0003](docs/adr/0003-formato-da-pasta-da-nuvem.md).
