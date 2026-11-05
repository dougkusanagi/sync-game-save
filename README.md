# sync-game-save

Launcher/HUB de jogos para PC que sincroniza o save com uma pasta de nuvem (OneDrive, Google Drive etc.) ao abrir e ao fechar cada jogo, sem perder saves.

## Estado

Em reescrita para Tauri 2 + React (ver [plano](docs/plans/launcher-hub.md) e [ADR 0001](docs/adr/0001-stack-tauri-react.md)). O código Electron abaixo será removido no início da Fase 1.

## Protótipo da interface

```bash
cd prototype
bun install
bun run dev
```

Abra `http://localhost:5199`. Detalhes em [prototype/README.md](prototype/README.md).

## App Electron antigo (será removido)

### Development

```bash
$ npm run dev
```

### Build

```bash
# For windows
$ npm run build:win

# For macOS
$ npm run build:mac

# For Linux
$ npm run build:linux
```
