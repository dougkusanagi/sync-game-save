# Plano: migrar de Tauri 2 (Rust) para Wails v3 (Go)

Tipo: tarefa

Branch: `feat/migracao-mygo` (o nome ficou da primeira escolha, MyGo; ver a ADR 0008).

## Objetivo

Trocar o Tauri 2 e todo o backend Rust pelo [Wails v3](https://github.com/wailsapp/wails) (Go, WebView do sistema), mantendo o frontend React, para:

1. Reduzir o espaço dos arquivos de desenvolvimento (hoje `src-tauri/target` ocupa 31,87 GiB).
2. Reduzir o tempo de compilação, principalmente o de release.

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

## O que o Wails v3 oferece (conferido em 2026-11-14, v3.1.0)

- Go com WebView do sistema (WebView2 no Windows, WebKitGTK no Linux). 37 mil estrelas; v3 estável desde 26/10/2026; a v2 ainda recebe correções.
- Serviços: `application.NewService(&T{})` expõe os métodos exportados; `wails3 generate bindings` gera os clientes TS em `bindings/` (usam `@wailsio/runtime`). Eventos tipados com `application.RegisterEvent[T]` e `app.Event.Emit`.
- Bandeja (`app.SystemTray.New()`), menus, diálogos, notificações, servidor de assets com `Middleware` (usado para as capas locais), atualizador com provedor do GitHub (não conferido em detalhe).
- Fechar para a bandeja: `win.RegisterHook(events.Common.WindowClosing, ...)` com `e.Cancel()` + `win.Hide()`.
- Empacotamento pelo Taskfile (`wails3 package`): NSIS no Windows (o `makensis` precisa estar no PATH; o Wails não baixa), AppImage, `.deb`/`.rpm` no Linux.
- Android e iOS: experimentais desde a v3.0.0, fora da promessa de estabilidade.
- No Linux provavelmente exige cgo (não conferido).

## Layout no repositório

- Raiz: `main.go`, `go.mod`, `Taskfile.yml` (do `wails3 init -t react`), `build/` (configuração e ativos de empacotamento; `build/config.yml` tem nome, identificador e versão).
- O frontend continua na raiz (não em `frontend/`, o padrão do Wails), para não quebrar o Tauri durante a migração. Os Taskfiles em `build/` foram ajustados: `dir: .`, `bindings/` na raiz e exclusões de `src/`, `dist/`, `src-tauri/` etc. nas fontes dos bindings e no `dev_mode`.
- Gerados e fora do git: `bin/`, `bindings/`, `.task/`, `*.syso`, `build/windows/nsis/MicrosoftEdgeWebview2Setup.exe`.
- `build/darwin` fica, embora o projeto não gere build para macOS: a tarefa de ícones do Wails grava nela.
- Comandos: `wails3 build` (gera `bin/sync-game-save.exe`), `wails3 package` (gera `bin/sync-game-save-amd64-installer.exe`), `wails3 dev` (Vite na porta 1420).

## Passos

### 0. Medir a base

- [x] Medição da versão Tauri (2026-11-14, Windows, worktree sem `target`, uma rodada só; o tamanho de `target` é acumulado e foi lido com `du`):

  | Etapa | Tempo | `target` depois |
  |---|---|---|
  | `cargo build` (debug, do zero) | 225 s | 4.768 MB |
  | `cargo test --workspace --no-run` | 30 s | 7.466 MB |
  | `cargo build` incremental (1 linha em `sgs-service`) | 13 s | 5.926 MB (leitura duvidosa: menor que a anterior) |
  | `cargo build --release` (LTO, `codegen-units = 1`) | 1.948 s (~32 min) | 7.790 MB |

  Executável release: 10,1 MiB (`sync-game-save.exe`). O `target` do checkout principal, com builds acumulados, chegou a 31,87 GiB.
- [ ] Tamanho do instalador NSIS (`bun run tauri build`; exige a chave de assinatura).

### 1. Prova de conceito e escolha do framework

- [x] PoC em MyGo 0.9.0 e em Wails 3.1.0, com o mesmo escopo: janela 1280×720 com o React atual, um binding (`Versao`), evento `jogo-estado`, bandeja com "Abrir"/"Sair", fechar para a bandeja e capas locais servidas só da pasta `art`. `SGS_DATA_DIR` troca a pasta de dados. Nos dois, o executável abre e cria a pasta `art`. Uma rodada de medição, cache do Go isolado:

  | | Tauri (Rust) | MyGo 0.9.0 | Wails 3.1.0 |
  |---|---|---|---|
  | Espaço do build | 4,8–7,8 GB | 262 MB | 134 MB |
  | Compilação Go/Rust do zero | 225 s (debug) | 148 s | 167 s |
  | Incremental só do compilador | 13 s | 22 s | 21 s |
  | Build completo do CLI do framework, incremental | — | 74 s (`mygo build -skip-installer`) | 99 s (`wails3 build`, refaz bindings e frontend) |
  | Build completo do zero | — | 381 s (com 1ª geração de bindings e download do NSIS) | 346 s |
  | Release | ~32 min (LTO) | ~2,5 min | já incluso no `wails3 build` (`-ldflags "-w -s"`) |
  | Executável | 10,1 MB | 10,4 MB | 10,7 MB |
  | Instalador NSIS | não medido | 3,5 MB (NSIS baixado pelo CLI) | 6,1 MB (inclui o instalador do WebView2; `wails3 package` 61 s) |
  | Instalar o CLI | — | < 1 min | > 10 min |

- [x] **Decisão (2026-11-14): Wails v3** — ver ADR 0008. Espaço e tempo empatam com o MyGo; o Wails ganha em maturidade, AppImage e mobile experimental. A recompilação incremental fica mais lenta que no Rust (21 s × 13 s); confirmar no passo 2 com o núcleo portado.
- [x] PoC do Wails no branch (substitui a do MyGo): `main.go`, `Taskfile.yml`, `build/`. `wails3 build` e `wails3 package` passam; o app abre e cria a pasta `art`.
- [ ] Conferir com a interface aberta: bandeja, fechar para a bandeja, capa em `/capa/<arquivo>` e o binding `Versao` chamado pelo frontend.

### 2. Portar o núcleo (`sgs-core` → `internal/core`)

- [x] Início do porte: caminhos lógicos e diário de operações em `internal/core`, com testes de portabilidade, rejeição de `..` e formato JSON do diário. Ainda não estão ligados ao aplicativo.
- [x] Portados índice, decisão em três vias e trava com heartbeat; testes Go cobrem hashes, decisões e posse da trava.
- [x] Snapshots: criação, verificação, restauração com backup, listagem e retenção; testes de falha de cópia e fixture gerada pelo Rust, lida e atualizada pelo Go. Ainda não estão ligados ao aplicativo.
- [x] Tentativas de leitura e cópia para arquivos ocupados, incluindo teste de compartilhamento no Windows.
- [x] A varredura segue diretórios simbólicos e detecta ciclos; no Unix, snapshots aceitam `:` e `\` em nomes de arquivos. No Windows, a verificação continua recusando nomes incompatíveis com a plataforma.
- [x] Engine de sincronização portado, com testes de dois PCs, conflito com backup, trava e estabilização. Ainda não está ligado ao aplicativo.
- [x] Ambiente e descoberta portados: marcadores locais, Documentos redirecionada, captura e comparação das pastas de save, exclusão de ruído e agrupamento por jogo. Testes locais no Windows e Linux; ainda sem integração com o aplicativo.
- [ ] Portar primeiro os testes Rust de `sgs-core` para Go (`go test`); eles são a especificação.
- [ ] Portar `paths`, `env`, `index`, `journal`, `lock`, `snapshot`, `sync`, `engine`, `discover`.
- [ ] `process`: Job Objects, ToolHelp e `WM_CLOSE` com `golang.org/x/sys/windows`; listagem de processos sem `sysinfo` (x/sys ou `gopsutil`).
- [ ] Hash com `crypto/sha256`. Conferir que IDs de snapshot e o formato da pasta da nuvem (ADR 0003, 0007) saem iguais byte a byte.
- [ ] Teste de compatibilidade: fixture de pasta da nuvem e snapshots gerados pela versão Rust, lidos e atualizados pela versão Go.
- [ ] Medir de novo o incremental (`go build` e `wails3 build`) com o núcleo portado.

### 3. Portar catálogo, biblioteca, conquistas e serviço

- [ ] SQLite com `modernc.org/sqlite` (Go puro, com FTS5). Abrir os bancos existentes sem migração destrutiva (ADR 0002).
- [ ] Manifesto do Ludusavi com `gopkg.in/yaml.v3`; medir tempo e memória da importação (30 MB).
- [ ] `sgs-catalog` (com o registro de erros por pasta da varredura, já no `main`), `sgs-library`, `sgs-achievements` (`net/http` no lugar de `ureq`), `sgs-service` (arte, HUB, temas — ADR 0005).
- [ ] Portar os testes de cada crate antes do código.

### 4. CLI

- [ ] Portar `sgs-cli` para `cmd/sgs`, com os mesmos comandos, saídas e códigos de saída.

### 5. App Wails

- [ ] Serviços com os 28 comandos, com os mesmos nomes de argumentos e formatos JSON.
- [ ] Eventos `jogo-estado` e `controle-acao`.
- [ ] Bandeja, fechar para a bandeja, notificações, diálogo de pasta.
- [ ] Capas locais pelo middleware `/capa/`, restrito à pasta de arte (equivale ao `asset_protocol_scope` atual). Conferir que não há como sair da pasta (`..`, caminhos absolutos).
- [ ] Controle (`controle.rs`): XInput no Windows via `x/sys/windows`; evdev no Linux. Conferir os mapeamentos de botões atuais.

### 6. Frontend

- [ ] Trocar em `src/api.ts` `invoke`/`listen`/`open`/updater pelos clientes de `bindings/` e `@wailsio/runtime`; manter a mesma interface pública de `api.ts` para não mexer nas telas.
- [ ] Trocar `convertFileSrc` em `src/useCapa.ts` por `/capa/<arquivo>`.
- [ ] Manter o modo mock funcionando; remover os pacotes `@tauri-apps/*` do `package.json`.
- [ ] Rodar `bun run e2e` (mock) e adaptar `e2e/app-real.cjs` para o executável Go (conferir como ligar a porta de depuração do WebView2).

### 7. Distribuição e atualização

- [ ] Windows: NSIS do Wails (`build/windows/nsis/project.nsi`). Mesmo nome de produto e instalação por usuário, como o NSIS atual; conferir se o caminho de instalação muda e se os dados em `AppData` continuam no mesmo lugar. Definir como obter o `makensis` (instalar o NSIS na máquina de build).
- [ ] Linux: AppImage pelo Wails (`build/linux/appimage`), mantendo a ADR 0006. Testar no Ubuntu.
- [ ] Atualizador: o Tauri usa minisign + `latest.json`. Planejar a ponte: uma última versão Tauri (v1.x) cujo `latest.json`, assinado com a chave atual, entrega o instalador Go; a partir dela, o atualizador do Wails assume. Testar de ponta a ponta numa máquina com a v1.x instalada.
- [ ] Chave de assinatura do novo atualizador fora do repositório; registrar o local no `MEMORY.md`.

### 8. Remoção do Tauri e documentação

- [ ] Remover `src-tauri/`, `Cargo.*` e os scripts `tauri` do `package.json`; ajustar `vite.config.ts` (comentários e variáveis do Tauri).
- [ ] Marcar a ADR 0001 como "Substituído por 0008" e a ADR 0008 como "Aceito"; nova ADR se a distribuição mudar (ADR 0006).
- [ ] Atualizar `README.md` (como rodar, testar e gerar build), `MEMORY.md` e `docs/architecture.md` (criar com o diagrama Mermaid).
- [ ] Fragmento em `changelog/unreleased/` e nova versão **major** (2.0.0), por causa da troca de atualizador e de instalador.
- [ ] Repetir as medições e registrar o resultado.

## Riscos

- **Perda ou corrupção de saves** se o port do sync divergir da versão Rust. Mitigação: testes portados antes do código e fixtures de compatibilidade gerados pela versão Rust.
- Ponte do atualizador: se falhar, usuários da v1.x ficam presos. Testar antes de publicar.
- Recompilação incremental mais lenta que no Rust.
- Esforço: reescrever ~5.200 linhas e seus testes. A versão Tauri continua no `main` até o fim.
- Mobile no Wails é experimental; não contar com ele no planejamento.

## Tentativas que falharam

- `wails3 init -t react-ts`: falha com "invalid remote git URL". O modelo `react-ts` não existe na v3.1; o nome certo é `react` (lista em `wails3 init -l`).
- Remover `build/darwin`: a tarefa de ícones do Wails grava nela, e o build falha.

## Dúvidas abertas

- O `modernc.org/sqlite` lê os bancos atuais (FTS5, versão do esquema) sem conversão?
- O atualizador do Wails cobre o fluxo atual (verificar, baixar, reiniciar) e aceita a ponte a partir da v1.x?
- O Wails exige cgo no Linux? Afeta o build do AppImage.

## Próximo passo

Portar processos, completar os testes de paridade do engine e integrar o núcleo ao serviço; conferir na tela a bandeja, as capas e o binding (fim do passo 1).
