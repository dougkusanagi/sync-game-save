# sync-game-save

Launcher/HUB de jogos para PC que sincroniza o save com uma pasta de nuvem (OneDrive, Google Drive etc.) ao abrir e ao fechar cada jogo, sem perder saves.

![Biblioteca de jogos do Sync Game Save, com os botões Jogar, Conquistas, Versões e Remover em cada jogo](.github/preview.png)

## Estado

Em desenvolvimento com Tauri 2 + React (ver [plano](docs/plans/launcher-hub.md) e [ADR 0001](docs/adr/0001-stack-tauri-react.md)). Estão prontos o núcleo de sync em Rust (`sgs-core`), o catálogo (`sgs-catalog`), a biblioteca local (`sgs-library`), o serviço (`sgs-service`), a CLI `sgs` e as telas do MVP: primeiro uso, biblioteca, adicionar jogo, versões, conflito, configurações e Sobre. Os sons da interface são sintetizados em `src/som.ts`.

As conquistas da Steam aparecem em cada jogo quando você informa a chave de API e o SteamID64 em Configurações. A chave fica só neste PC, no banco local. Faltam o assistente de varredura de pastas, os temas e os instaladores.

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
cd src-tauri && cargo test --workspace   # núcleo, catálogo, biblioteca, serviço e CLI
bun run e2e                # testes E2E da interface (dados simulados)
(cd src-tauri && cargo build) && bun run e2e:app   # abre o app real e confere o fluxo (Windows)
```

`SGS_DATA_DIR` troca a pasta de dados do app (útil para testes sem mexer na sua biblioteca). Na primeira vez, instale o navegador dos testes E2E com `bunx playwright install chromium`. A interface também abre no navegador (`bun run dev`, `http://localhost:1420/?cenario=jogos`) com dados simulados; os cenários são `jogos`, `vazio`, `primeiro-uso` e `conflito`.

## CLI `sgs`

```bash
cd src-tauri
cargo run -p sgs-cli -- help
cargo run -p sgs-cli -- run --sync-dir <pasta da nuvem> --game "Nome" --save-dir <pasta do save> -- <jogo.exe>
```

O `run` sincroniza antes de abrir, abre o programa, espera ele fechar e envia o save. Uma queda ou um erro do jogo não perde o save: o formato da pasta da nuvem está no [ADR 0003](docs/adr/0003-formato-da-pasta-da-nuvem.md).
