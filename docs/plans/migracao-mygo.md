# Plano: migrar de Tauri 2 (Rust) para MyGo (Go)

Tipo: tarefa

Branch: `feat/migracao-mygo`

## Objetivo

Trocar o Tauri 2 e todo o backend Rust pelo [MyGo](https://github.com/egoist/mygo) (Go, WebView do sistema), mantendo o frontend React, para:

1. Reduzir o espaço dos arquivos de desenvolvimento (hoje `src-tauri/target` ocupa 31,87 GiB).
2. Reduzir o tempo de compilação em desenvolvimento e em release.

Restrições:

- Paridade de funções: o app migrado faz tudo o que a versão Tauri faz hoje.
- **Compatibilidade de dados**: o app lê e grava os mesmos arquivos que o usuário já tem (pasta da nuvem, biblioteca SQLite, catálogo, snapshots, temas, configurações). Nenhum save se perde na troca.
- Quem tem a v1.x instalada recebe a versão Go pelo atualizador.

Critério de conclusão: a suíte Go cobre os mesmos casos dos testes Rust e passa; `bun run e2e` e o E2E do app real passam no Windows; o instalador NSIS atualiza uma instalação v1.x sem perder dados; as medições de espaço e tempo estão registradas neste plano.

## Estado atual (2026-11-14)

- ~5.240 linhas de Rust em 7 crates (`src-tauri/`):
  - `sgs-core` (~1.450): sync, snapshots, índice, journal, lock, caminhos, descoberta de pastas de save, processos (Job Objects e ToolHelp no Windows, `sysinfo`).
  - `sgs-service` (~1.520): serviço da interface, arte, HUB, temas.
  - `sgs-achievements` (~470), `sgs-catalog` (~530, SQLite FTS5 + manifesto YAML do Ludusavi), `sgs-library` (~280, SQLite), `sgs-cli` (~320).
  - App Tauri (`src-tauri/src`, ~650): 28 comandos, eventos `jogo-estado` e `controle-acao`, bandeja, fechar para a bandeja, protocolo `asset:` para capas locais, controle por `gilrs` (`controle.rs`).
- Plugins: dialog, notification, updater (minisign, `latest.json` no último release), process.
- Frontend: toda a ponte com o backend está em `src/api.ts` (`invoke`, `listen`, `open`, updater) e `src/useCapa.ts` (`convertFileSrc`). Há um modo mock (`src/mock`) para rodar sem backend.
- E2E: `e2e/run.cjs` (frontend com mock) e `e2e/app-real.cjs` (abre o app real e controla o WebView2 por CDP).
- Distribuição: NSIS (Windows) e AppImage (Linux), conforme ADR 0006.

## O que o MyGo oferece (conferido em 2026-11-14, v0.9.0)

- Go puro, sem cgo; exige Go 1.27.3+. WebView2 no Windows, WebKitGTK 4.1 no Linux.
- Bindings: `mygo.Bind(&T{})` expõe métodos exportados; `mygo generate` gera o cliente TS tipado (`src/mygo.ts`). Erros viram `CallError`. Eventos: `mygo.NewEvent[T]` + `Emit`/`Broadcast`.
- Bandeja (`mygo.NewTray`), diálogos, notificações, atualizador assinado (GitHub Releases), protocolo próprio (`mygo.Protocol.Handle`), várias janelas, `devUrl` para HMR do Vite.
- Fechar para a bandeja: manual (`OnClose` → `PreventDefault` + `Hide`).
- Instalador NSIS por usuário em `%LOCALAPPDATA%\Programs`.
- **Não gera AppImage** (gera tar.gz, `.deb`, `.rpm`, PKGBUILD).
- Pré-1.0, com quebras de API em 0.8.0 e 0.9.0.

## Passos

### 0. Medir a base

- [ ] Registrar neste plano, com a versão Tauri: tamanho de `src-tauri/target` após `cargo clean` + `cargo build` (debug), + `cargo test --workspace`, + `bun run tauri build`; tempo de build limpo, build incremental (uma linha alterada em `sgs-service`) e release.
- [ ] Registrar o tamanho do instalador NSIS e do executável.

### 1. Prova de conceito (descarta ou continua)

- [ ] Instalar Go 1.27.3+ e o CLI do MyGo; criar `go.mod` na raiz e `mygo.config.ts` com `devUrl: http://localhost:1420`.
- [ ] Janela única carregando o React atual, um binding (`app_version`), um evento, a bandeja com fechar para a bandeja e uma capa local servida por protocolo próprio.
- [ ] Gerar o NSIS e medir espaço e tempo como no passo 0.
- [ ] **Ponto de decisão**: se o ganho não compensar ou faltar uma função essencial (protocolo, bandeja, updater), parar aqui e registrar o resultado.

### 2. Portar o núcleo (`sgs-core` → `internal/core`)

- [ ] Portar primeiro os testes Rust de `sgs-core` para Go (`go test`); eles são a especificação.
- [ ] Portar `paths`, `env`, `index`, `journal`, `lock`, `snapshot`, `sync`, `engine`, `discover`.
- [ ] `process`: Job Objects, ToolHelp e `WM_CLOSE` com `golang.org/x/sys/windows`; listagem de processos sem `sysinfo` (x/sys ou `gopsutil`).
- [ ] Hash com `crypto/sha256`. Conferir que IDs de snapshot e o formato da pasta da nuvem (ADR 0003, 0007) saem iguais byte a byte.
- [ ] Teste de compatibilidade: fixture de pasta da nuvem e snapshots gerados pela versão Rust, lidos e atualizados pela versão Go.

### 3. Portar catálogo, biblioteca, conquistas e serviço

- [ ] SQLite com `modernc.org/sqlite` (Go puro, com FTS5), para manter o build sem cgo. Abrir os bancos existentes sem migração destrutiva (ADR 0002).
- [ ] Manifesto do Ludusavi com `gopkg.in/yaml.v3`; medir tempo e memória da importação (30 MB).
- [ ] `sgs-catalog` (incluir as mudanças de `recognize.rs` que estiverem no `main`), `sgs-library`, `sgs-achievements` (`net/http` no lugar de `ureq`), `sgs-service` (arte, HUB, temas — ADR 0005).
- [ ] Portar os testes de cada crate antes do código.

### 4. CLI

- [ ] Portar `sgs-cli` para `cmd/sgs`, com os mesmos comandos, saídas e códigos de saída.

### 5. App MyGo

- [ ] `main.go`: janela 1280×720, título "Sync Game Save", bindings dos 28 comandos com os mesmos nomes de argumentos e formatos JSON.
- [ ] Eventos `jogo-estado` e `controle-acao`.
- [ ] Bandeja, fechar para a bandeja, notificações, diálogo de pasta.
- [ ] Capas locais: protocolo próprio restrito à pasta de arte (equivale ao `asset_protocol_scope` atual). Atualizar a CSP.
- [ ] Controle (`controle.rs`): XInput no Windows via `x/sys/windows`; evdev no Linux. Conferir os mapeamentos de botões atuais.
- [ ] Instância única, se a versão Tauri tiver esse comportamento (conferir em `lib.rs`).

### 6. Frontend

- [ ] Trocar em `src/api.ts` `invoke`/`listen`/`open`/updater pelo cliente gerado `src/mygo.ts`; manter a mesma interface pública de `api.ts` para não mexer nas telas.
- [ ] Trocar `convertFileSrc` em `src/useCapa.ts` pela URL do protocolo próprio.
- [ ] Manter o modo mock funcionando; remover os pacotes `@tauri-apps/*` do `package.json`.
- [ ] Rodar `bun run e2e` (mock) e adaptar `e2e/app-real.cjs` para o executável Go (o CDP do WebView2 deve continuar igual; conferir a variável que liga a porta de depuração).

### 7. Distribuição e atualização

- [ ] Windows: NSIS do MyGo. Mesmo nome de produto e mesmo modo por usuário; conferir se o caminho de instalação muda em relação ao NSIS do Tauri e se dados em `AppData` continuam no mesmo lugar.
- [ ] Linux: o MyGo não gera AppImage. Gerar a AppImage com `appimagetool` a partir da saída do MyGo, ou registrar no ADR a troca para `.deb`/`.rpm`/tar.gz.
- [ ] Atualizador: o MyGo usa outro formato de assinatura e de manifesto. Planejar a ponte: uma última versão Tauri (v1.x) cujo `latest.json`, assinado com a chave minisign atual, entrega o instalador Go; a partir dela, o atualizador do MyGo assume. Testar de ponta a ponta numa máquina com a v1.x instalada.
- [ ] Nova chave de assinatura do MyGo fora do repositório; registrar o local no `MEMORY.md`.

### 8. Remoção do Tauri e documentação

- [ ] Remover `src-tauri/`, `Cargo.*` e os scripts `tauri` do `package.json`.
- [ ] ADR 0008 "Stack MyGo + React", marcando a ADR 0001 como "Substituído por 0008"; atualizar a ADR 0006 por nova ADR se a distribuição mudar.
- [ ] Atualizar `README.md` (como rodar, testar e gerar build), `MEMORY.md` e `docs/architecture.md` (criar com o diagrama Mermaid, já que há vários componentes).
- [ ] Fragmento em `changelog/unreleased/` e nova versão **major** (2.0.0), por causa da troca de atualizador e de instalador.
- [ ] Repetir as medições do passo 0 e registrar o resultado.

## Riscos

- **Perda ou corrupção de saves** se o port do sync divergir da versão Rust. Mitigação: testes portados antes do código e fixtures de compatibilidade gerados pela versão Rust.
- MyGo pré-1.0 com quebras frequentes: fixar a versão no `go.mod` e só atualizar de propósito.
- Ponte do atualizador: se falhar, usuários da v1.x ficam presos. Testar antes de publicar.
- Linux sem AppImage nativa.
- Esforço: reescrever ~5.200 linhas e seus testes. A versão Tauri continua no `main` até o fim.

## Tentativas que falharam

Nenhuma até agora.

## Dúvidas abertas

- Manter AppImage (montagem manual) ou aceitar os formatos Linux do MyGo?
- O app Tauri tem instância única? (conferir `lib.rs`).
- O `modernc.org/sqlite` lê os bancos atuais (FTS5, versão do esquema) sem conversão? Confirmar na PoC.

## Próximo passo

Passo 0: medir a base da versão Tauri.
