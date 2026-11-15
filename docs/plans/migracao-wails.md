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

## Progressão por etapa

Painel visual: [progresso da migração](progresso-wails.html). As barras contam os itens marcados nos passos abaixo, com a etapa de processos dividida em entregas verificáveis. Os itens têm tamanhos diferentes e não representam uma estimativa de tempo. A mudança da contagem anterior (12 de 45) reflete essa divisão, não trabalho novo.

- [ ] **0. Medir a base:** medição do Tauri concluída; falta o tamanho do instalador.
- [ ] **1. Prova de conceito:** framework escolhido e builds feitos; falta a conferência visual.
- [ ] **2. Núcleo:** caminhos, diário, índice, trava, snapshots, sincronização e descoberta portados; processos e testes de paridade em andamento.
  - [x] Acompanhamento do processo e seus descendentes, registro de execuções e heartbeat.
  - [x] Pedido de fechamento e fechamento forçado dos processos no Job Object do Windows, com teste de executável temporário para a força.
  - [x] Fechamento normal e forçado de processos rastreados fora do Job no Windows, com verificação do horário de criação; teste de força com processos temporários e identidade divergente.
  - [x] Parada forçada no Linux via pidfd, com teste no WSL para processo temporário e identidade divergente.
  - [x] Teste de janela real: fechamento normal do jogo sem atingir outra janela temporária.
  - [x] Engine: cenários de nuvem incompleta, save ausente, alterações contínuas e abertura com trava de outro PC testados no Windows e Linux; escolha do save local em conflito testada no Windows.
  - [ ] Integrar o núcleo ao serviço e completar a paridade dos demais testes.
- [ ] **3. Catálogo, biblioteca, conquistas e serviço:** porte pendente.
- [ ] **4. CLI:** porte pendente.
- [ ] **5. App Wails:** serviços, eventos, bandeja, diálogos e controle pendentes.
- [ ] **6. Frontend:** diálogos, eventos e arte preparados; comandos, atualizador e E2E do aplicativo real dependem dos serviços Go.
- [ ] **7. Distribuição e atualização:** instaladores e atualização a partir da v1 pendentes.
- [ ] **8. Remoção do Tauri e documentação:** pendentes após validar a paridade.
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
- [x] Engine de sincronização portado, com testes de dois PCs, conflito com backup, trava, abertura forçada, estabilização, alterações contínuas, nuvem incompleta e save ausente. Os testes do engine passaram no Windows e no WSL; ainda não está ligado ao aplicativo.
- [x] Ambiente e descoberta portados: marcadores locais, Documentos redirecionada, captura e comparação das pastas de save, exclusão de ruído e agrupamento por jogo. Testes locais no Windows e Linux; ainda sem integração com o aplicativo.
- [x] Registro das execuções ativas ligado ao acompanhamento, incluindo o intervalo de espera e execuções simultâneas do mesmo programa; testes verificam registro e remoção.
- [x] Acompanhar o processo, descendentes e executáveis relançados; listar processos e manter o heartbeat da trava.
- [x] Encaminhar o pedido de fechamento normal no Windows, inclusive a processos rastreados fora do Job, com conferência da identidade.
- [x] Implementar a parada forçada no Windows após a espera de dez segundos, com teste de processos temporários e identidade divergente.
- [x] Implementar a parada forçada no Linux com pidfd e verificar a identidade, com teste no WSL. O fechamento normal segue sem sinal, como no Rust.
- [x] Testar o fechamento de uma janela real no Windows e confirmar que outra janela temporária permanece aberta.
- [x] Testar o fechamento normal de uma janela rastreada fora do Job no Windows.
- [ ] Completar os demais testes de integração e concorrência da parada; `go test -race` ainda não rodou porque CGO está desativado.
- [x] Testar em conjunto o engine, o processo real e a renovação da trava: um jogo com saída de erro ainda sincroniza e libera a trava após encerrar; validado no Windows e no WSL.
- [x] Integrar o acompanhamento e a parada ao serviço Go: `Jogar`, `PararJogo` e `ForcarParadaJogo` (`app_play.go`), com testes de sync completo, jogo sem sync, trava de outro PC, pasta de sync ausente e abertura duplicada, usando o binário de teste como jogo. Conflito com escolha e emissão real do evento ainda sem teste próprio. `Versoes` e `RestaurarVersao` (`app_versions.go`) portados, com teste de restauração e backup.
- [ ] Portar primeiro os testes Rust de `sgs-core` para Go (`go test`); eles são a especificação.
- [ ] Portar `paths`, `env`, `index`, `journal`, `lock`, `snapshot`, `sync`, `engine`, `discover`.
- [ ] `process`: Job Objects, ToolHelp e `WM_CLOSE` com `golang.org/x/sys/windows`; listagem de processos sem `sysinfo` (x/sys ou `gopsutil`).
- [ ] Hash com `crypto/sha256`. Conferir que IDs de snapshot e o formato da pasta da nuvem (ADR 0003, 0007) saem iguais byte a byte.
- [ ] Teste de compatibilidade: fixture de pasta da nuvem e snapshots gerados pela versão Rust, lidos e atualizados pela versão Go.
- [ ] Medir de novo o incremental (`go build` e `wails3 build`) com o núcleo portado.

### 3. Portar catálogo, biblioteca, conquistas e serviço

- [x] Adicionar SQLite Go puro (`modernc.org/sqlite`) e testar FTS5 localmente.
- [x] Abrir biblioteca SQLite e aplicar migrações atômicas 1–3; testes cobrem preservação de dados de esquema 1, persistência e recusa de esquema futuro.
- [x] Portar operações da biblioteca de jogos, configurações, estado de sync, tempo jogado e cache de conquistas; testes cobrem edição, duplicidade, exclusão e persistência.
- [ ] Testar leitura de bancos existentes gerados pela versão Rust e concluir a integração da biblioteca e do catálogo sem migração destrutiva (ADR 0002).
- [x] Catálogo em `internal/catalog`: conversão do manifesto do Ludusavi (`gopkg.in/yaml.v3`) para SQLite com FTS5, consultas, reconhecimento de pasta e varredura, com os casos dos testes Rust portados e `ReconhecerPasta` ligado ao catálogo. Download do manifesto com ETag e `AtualizarCatalogo` ligados; importação do manifesto real (17,8 MB, 53.546 jogos) medida em 2026-11-14 no Windows: 28 s e ~585 MB de memória do processo, sem comparação com o Rust ainda (o app deve rodar a importação fora do caminho de abertura). Busca "half life" não acha "Half-Life", como no Rust (frase literal no trigram). O E2E Wails falhou uma vez por tempo no primeiro uso e passou na repetição.
- [ ] `sgs-catalog` (com o registro de erros por pasta da varredura, já no `main`), `sgs-library`, `sgs-achievements` (`net/http` no lugar de `ureq`), `sgs-service` (arte, HUB, temas — ADR 0005).
- [x] Temas (`app_themes.go`): pacotes `themes/<id>/theme.json`, lista restrita de tokens e escolha persistida, com os testes do Rust portados. Diferença: `min_app_version` compara com a versão do app (1.0.0); o Rust comparava com a do crate (0.1.0), o que recusava qualquer mínimo acima disso.
- [x] Descoberta de save por mudança (`app_discover.go`): `DescobrirSave` e `CandidatosSave`, com teste de ponta a ponta usando o binário de teste como jogo. A captura usa a mesma chave (sha256 do caminho) e o mesmo JSON da versão Rust; ainda falta teste com captura gerada pelo Rust.
- [x] Pacote `internal/achievements`: provedores Steam (chave de API e perfil público), raridade, XML da comunidade e `loginusers.vdf`, com os testes do Rust portados e JSON compatível com o cache da biblioteca. Falta ligar ao serviço (`Conquistas`, HUB).
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

- [x] Distinguir Wails, Tauri e navegador simulado, inclusive antes da configuração do runtime; teste no WebView real confirma que Wails sem os serviços não apresenta dados fictícios.
- [x] Usar os diálogos nativos do Wails para selecionar pasta e executável, preservando a ponte Tauri durante o porte.
- [x] Preparar inscrição e cancelamento dos eventos `jogo-estado` e `controle-acao` pelo runtime Wails; a emissão de controle no Go ainda falta.
- [x] Converter caminhos locais de arte para `/capa/<arquivo>` no Wails, com teste de codificação e nomes inválidos; preservar a conversão Tauri e o mock.
- [x] Executar os 27 cenários E2E mock existentes; regressão adicional valida a identificação Wails simulada. Layout de indisponibilidade verificado em 390 e 1280 px.
- [x] Integrar a versão da tela Sobre ao binding real `App.Versao`; teste de runtime real e regressão da ponte passaram.
- [x] Integrar listagem e inclusão de jogos ao Wails com biblioteca SQLite temporária no E2E real e regressão da ponte.
- [x] Integrar estado e configuração inicial ao Wails; primeiro uso verificado no WebView real, preservando a lista de PCs e recusando dados incompletos.
- [x] Integrar cadastro manual sem catálogo, reconhecimento da pasta e conversão do caminho de save; fluxo verificado no WebView real. A consulta de catálogo existente ainda não foi portada.
- [x] Integrar edição e remoção de jogos; testes no WebView real e no backend cobrem preservação dos saves, invalidação de base e rollback após edição inválida.
- [ ] Integrar os demais comandos aos bindings gerados quando os serviços Go estiverem implementados, mantendo a interface de `api.ts`.
- [ ] Integrar o atualizador Wails quando o provedor e a ponte de atualização estiverem definidos na etapa 7.
- [ ] Remover os pacotes `@tauri-apps/*` após a substituição funcional e a validação da versão Wails; manter o mock.
- [x] Criar `bun run e2e:wails`: build de teste isolado, WebView2 real, binding de versão e evento de jogo; a porta de depuração só é habilitada na build tag `sgs_e2e`.
- [ ] Adaptar e executar os fluxos completos de `e2e/app-real.cjs` com os serviços Go reais.

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

- O teste de parada usou `PING.EXE` diretamente em `System32`. Como o rastreador inclui executáveis na pasta do jogo, o fechamento normal e o forçado podiam atingir processos do sistema. A integração de parada e seu teste foram removidos antes de commit; o rastreador passou a excluir a inclusão por pasta dentro de `SystemRoot`, com regressão sem encerrar processos. Na inspeção de 14/11/2026 às 17:26, o Explorer tinha iniciado às 17:23:58 e o boot do Windows era de 16:56:03; os registros consultados não identificaram quais processos o teste atingiu. A parada exige nova implementação e revisão antes de uso.
- `wails3 init -t react-ts`: falha com "invalid remote git URL". O modelo `react-ts` não existe na v3.1; o nome certo é `react` (lista em `wails3 init -l`).
- Remover `build/darwin`: a tarefa de ícones do Wails grava nela, e o build falha.

- A variável WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS é limpa pelo carregador do Wails; WindowsOptions.AdditionalBrowserArgs, em build tag exclusiva de teste, abriu a porta CDP e permitiu o teste real.
- Rodar go test ao mesmo tempo que un run build causou erro temporário ao limpar dist, usado pelo go:embed. Os comandos passaram em sequência.

## Dúvidas abertas

- O `modernc.org/sqlite` lê os bancos atuais (FTS5, versão do esquema) sem conversão?
- O atualizador do Wails cobre o fluxo atual (verificar, baixar, reiniciar) e aceita a ponte a partir da v1.x?
- O Wails exige cgo no Linux? Afeta o build do AppImage.

## Próximo passo

Ligar os comandos restantes (arte, conquistas, controle); a varredura de pasta (`ScanFolder` já portado) depende de trazer o `main` ao branch, pois a interface de varredura só existe lá; validar a biblioteca Go com dados gerados pela versão Rust; completar os testes de paridade do núcleo e integrar o engine ao serviço; conferir na tela a bandeja, as capas e o binding (fim do passo 1).
