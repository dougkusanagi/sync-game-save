# Plano: launcher/HUB com sincronização segura de saves

Tipo: tarefa

## Objetivo

Um launcher/HUB de jogos para PC que:

1. Abre os jogos pela própria interface.
2. Sincroniza os saves com uma pasta de nuvem (OneDrive, Google Drive, Dropbox, Syncthing etc.) **ao abrir e ao fechar** cada jogo, para jogos sem save na nuvem.
3. **Nunca perde um save.** Toda ação que sobrescreve ou apaga arquivos cria antes uma cópia recuperável.
4. Tem visual moderno, inspirado no PS5 (sem copiar marca, ícones ou layout exato), com uso impecável por controle e por teclado + mouse.
5. Mostra conquistas e, no futuro, aceita temas criados pela comunidade.

Critério de conclusão do MVP: os itens P0 da seção "Prioridades do MVP" funcionam no Windows, com os testes de sync passando.

## Base original (2026-11-05, antes da migração)

- Stack: Electron 31 + electron-vite + Vue 3 + TypeScript + Tailwind 3. Gerenciador: bun (`bun.lockb`).
- `src/main/GameRepository.ts`: lista de jogos fixa no código (2 jogos).
- `src/main/GameService.ts`: abre o `.exe` com `execFile` e copia os saves **só se o código de saída for 0**, para um caminho fixo (`D:/fake_game_files/bkp_save_files`). Não restaura antes de abrir. Sobrescreve o destino sem cópia anterior.
- `src/main/utils.ts`: glob dos caminhos de save, troca de `%userprofile%` etc. e cópia recursiva.
- `src/steam/manifest.yaml` (30 MB): manifesto do Ludusavi (caminhos de save de milhares de jogos, licença MIT). Ainda não é usado pela interface.
- `src/steam/steamUrlBuilder.ts`: URLs de arte da Steam (header, hero, logo, capa).
- `prisma/schema.prisma` + `prisma/dev.db`: esquema SQLite criado, mas não usado. O `dev.db` está versionado.
- `glob.ts` na raiz: script de teste solto.
- Interface: uma tela com campo de pasta de destino e grade de jogos com botão "Abrir Jogo".

Conclusão: o código existente é pequeno (~300 linhas úteis). O usuário autorizou descartar todo o código Electron; ele sai no primeiro passo da Fase 1. O que vale preservar são as ideias: uso do manifesto do Ludusavi, URLs de arte da Steam e o modelo `Game`/`SavePath`.

Protótipo da interface: pronto em `prototype/` (HTML + CSS + JS, servidor Vite+). Veja `prototype/README.md`.

## Decisões tomadas (2026-11-05)

- **Stack: Tauri 2 + React + TypeScript** (ver `docs/adr/0001-stack-tauri-react.md`). React no lugar de Vue porque as ferramentas de IA geram e revisam React com mais qualidade.
- **Plataformas:** Windows primeiro; Ubuntu também coberto na 1.0 (AppImage). Caminhos de save do Linux e Proton entram junto.
- **Um PC por vez:** não há suporte a jogar o mesmo jogo em dois PCs ao mesmo tempo. A trava (`lock.json`) avisa e pede confirmação.
- **Bandeja:** o HUB fica na bandeja do sistema e pode iniciar com o sistema.
- **Emuladores:** fora da 1.0, mas o modelo de dados já prevê regras de save por emulador/ROM.
- **Catálogo de jogos:** manifesto do Ludusavi convertido em SQLite com FTS5 (ver `docs/adr/0002-catalogo-de-jogos-sqlite-fts5.md`, status Proposto).

## Prioridades do MVP

O MVP é a menor versão que substitui a cópia manual sem risco de perder save. Nada de P1/P2 começa antes de todo o P0 funcionar e ter testes.

**P0 — MVP**

1. **Pasta de sync**: em Configurações, escolher com o seletor de pasta nativo a pasta onde ficam os saves (ex.: dentro do OneDrive). Cada jogo ganha uma subpasta com o nome dele. Sem a pasta configurada, o botão Jogar explica o que falta e leva a essa tela.
2. **Adicionar jogo**: seletor nativo de pasta ou `.exe`. O app sugere o nome, o executável e os **caminhos do save** (pelo catálogo do Ludusavi, buscando pelo nome da pasta). Se não achar, o usuário escolhe a pasta ou os arquivos do save com o seletor nativo. Sem caminho de save, o jogo não pode ser sincronizado.
3. **Persistência**: jogos adicionados e o estado do último sync em `library.db` (SQLite, local, fora da nuvem). Editar e remover jogo.
4. **Iniciar / parar jogo**: botão Jogar e botão Parar (pede confirmação, porque fechar à força pode corromper o save). Acompanhar o processo e os filhos, para jogos abertos por um launcher que fecha logo.
5. **Sync ao abrir**: comparar o save local com o da pasta de sync.
   - Só a pasta de sync mudou: backup local → restaurar → abrir.
   - Só o local mudou: criar snapshot na pasta de sync → abrir.
   - Os dois mudaram (**conflito**): mostrar data e tamanho de cada lado e oferecer "Usar o local" ou "Usar o da pasta de sync". O lado descartado vira snapshot/backup. **Nenhum snapshot é apagado e o save original da pasta do jogo sempre tem cópia antes de ser sobrescrito.**
   - Pasta de sync indisponível: avisar e oferecer "Jogar sem sincronizar".
6. **Sync ao fechar**: esperar o jogo terminar de gravar e criar um snapshot novo (imutável, conferido por hash, `head.json` atualizado por último), mesmo se o jogo fechar com erro.
7. **Interface mínima** com navegação por controle e por teclado + mouse: lista de jogos, Jogar/Parar, estado do sync, tela de conflito, Configurações e Sobre (nome, versão, build).
8. **Rede de segurança mínima** (itens que costumam faltar):
   - Ver os snapshots de um jogo e restaurar um deles (com backup antes). Sem isso, um snapshot guardado não ajuda o usuário a recuperar o save.
   - Aviso quando o jogo é aberto em outro PC (`lock.json`), já que os dois PCs usam a mesma pasta.
   - Mensagens de erro claras e um log em arquivo, para investigar uma falha de sync.
   - Testes automáticos dos cenários de sync (seção "Cenários de teste do sync").
   - Um instalador para Windows (NSIS).

### Como cada item do P0 funciona (decisões do MVP)

**Onde encontrar o save de um jogo** (em ordem; para no primeiro passo com resultado confiável):

1. **Catálogo do Ludusavi** (`catalog.db`): identificar o jogo pelo nome da pasta escolhida (`installDir`). Se não bater, buscar pelo nome da pasta limpo e pelo nome do `.exe`. Resolver os marcadores (`<base>` = pasta do jogo, `<home>`, `<winAppData>`, `<winLocalAppData>`, `<winDocuments>`, `<winPublic>`, `<winProgramData>`, `<storeUserId>` como curinga) e as regras `when` (SO e loja). Usar só regras com tag `save`; regras `config` ficam desmarcadas por padrão. Mostrar cada caminho com "encontrado: N arquivos, tamanho" ou "não encontrado ainda".
2. **Registro do Windows**: alguns jogos salvam no registro (`registry` no manifesto). No MVP, avisar que o jogo salva no registro e que esse tipo não é sincronizado ainda (exportar `.reg` fica para P1).
3. **Detecção por mudança** (quando o catálogo não conhece o jogo ou nenhum caminho existe): o usuário clica "Descobrir onde o jogo salva", joga e salva uma vez. Antes de abrir, o app registra data e tamanho dos arquivos nas pastas comuns de save (`%USERPROFILE%\Saved Games`, `Documents`, `Documents\My Games`, `%APPDATA%`, `%LOCALAPPDATA%`, `%USERPROFILE%\AppData\LocalLow`, `%PUBLIC%\Documents` e a pasta do jogo). Ao fechar, lista as pastas com arquivos novos ou alterados, ignorando caches, logs, shaders e crash dumps, e sugere a mais provável. O usuário confirma.
4. **Manual**: seletor nativo de pasta ou arquivos. O usuário sempre pode editar a lista.

Os caminhos são guardados com marcadores (ex.: `<winAppData>/Jogo/saves`), não absolutos, para funcionar em outro PC e com outro usuário.

**Adicionar jogo**: aceitar pasta ou `.exe`. Com pasta, escolher o executável pelo `launch` do manifesto; senão, o `.exe` maior que não seja instalador, desinstalador, crash handler ou redistribuível. O nome do jogo vem do catálogo ou do nome da pasta; o usuário pode editar. Uma única tela com tudo preenchido e um botão "Adicionar".

**Pasta de sync**: `<pasta escolhida>/<nome do jogo sem caracteres inválidos>/`, com `snapshots/<data>_<PC>/` em arquivos legíveis, `head.json` e `lock.json` (layout completo em "Projeto do sync seguro"). Se o nome mudar, a pasta não é renomeada: o id do jogo fica em `game.json`.

**Primeiro sync de um jogo novo**: se a pasta de sync do jogo está vazia, o save local vira o primeiro snapshot. Se já tem snapshots (de outro PC) e o local também tem save, é conflito: o usuário escolhe, nada é apagado.

**Parar jogo**: primeiro pede para fechar (fecha a janela principal), espera 10 s e só então oferece "Forçar fechamento", com aviso de risco ao save.

**Processo**: Job Object no Windows (pega processos filhos) e, como reserva, procurar processos cujo executável esteja dentro da pasta do jogo. O jogo é considerado fechado quando nenhum desses roda por 5 s.

**Sync ao fechar**: esperar 3 s depois do fechamento e confirmar que os arquivos do save pararam de mudar (dois cálculos de hash iguais com 1 s de intervalo). Se nada mudou, não cria snapshot.

**Backups locais**: antes de sobrescrever o save local, copiar para `%LOCALAPPDATA%\SyncGameSave\backups\<jogo>\<data>_<motivo>\`. Retenção: nunca apagar automaticamente no MVP; mostrar o espaço usado em Configurações.

**Retenção na pasta de sync**: o MVP não apaga snapshots. Limpeza configurável fica para P1.

**Trava de outro PC**: `lock.json` com PC, hora e heartbeat a cada 2 min. Trava com heartbeat de menos de 10 min gera aviso com "Abrir mesmo assim" e "Cancelar". Trava mais antiga é tratada como abandonada (ex.: PC desligado no meio do jogo).

**Log**: arquivo diário em `%LOCALAPPDATA%\SyncGameSave\logs\`, guardando 14 dias, com cada passo do sync (arquivos, hashes, decisões). Botão "Abrir pasta de logs" em Configurações.

**Interface do MVP**: uma tela de lista de jogos (com Jogar/Parar e estado do sync por jogo), a tela de adicionar/editar jogo, a de snapshots com "Restaurar", o diálogo de conflito, Configurações (pasta de sync, logs) e Sobre. Usar o motor de foco e os tokens do protótipo, sem a arte de fundo (arte é P1).

**P1 — logo depois do MVP**

- Assistente de primeiro uso (explicar o OneDrive/Google Drive, perguntar se já existe a pasta).
- Varrer uma pasta e adicionar vários jogos de uma vez.
- Interface completa do protótipo (Início com arte, biblioteca, busca, filtros, menu, central rápida).
- Ícone na bandeja.
- Sons: fundo ambiente e efeitos (seção "Som").
- Arte e metadados (Steam, SteamGridDB) e tempo de jogo.
- Instalador para Ubuntu (AppImage) e atualização automática.
- CLI `sgs run` e "Adicionar à Steam".

**P2 — depois**

- Conquistas da Steam com visual inspirado no PS5.
- Temas da comunidade e tema claro.
- Emuladores (saves por ROM, RetroAchievements).
- Steam Deck.

## Assistente de primeiro uso (P1)

Abre na primeira execução, depois da instalação. O HUB não abre jogos nem copia saves até o assistente terminar. Pode ser reaberto em Configurações → Nuvem.

1. **Boas-vindas**: o que o app faz, em uma frase.
2. **Por que uma pasta na nuvem**: explica em linguagem simples que o save fica guardado numa pasta sincronizada pelo OneDrive (ou Google Drive, Dropbox, Syncthing). Assim, se o PC quebrar ou for formatado, o save continua seguro e vai para outros PCs. Recomenda marcar a pasta como "Sempre manter neste dispositivo".
3. **Você já tem uma pasta de saves na nuvem?**
   - **Sim, já uso uma (de outro PC ou do Sync Game Save)**: o usuário escolhe a pasta. O app detecta se ela já tem dados do Sync Game Save (`SyncGameSave/devices.json`) e mostra os jogos e os PCs encontrados. Nada é sobrescrito: os jogos são ligados quando forem adicionados neste PC.
   - **Sim, mas copio os saves à mão**: o usuário escolhe a pasta atual. O app **não move nem altera** esses arquivos; cria `SyncGameSave/` ao lado e, ao adicionar cada jogo, oferece importar o save manual como primeira versão (com backup do save local antes).
   - **Não tenho**: o app detecta OneDrive, Google Drive e Dropbox instalados e sugere `<pasta do serviço>\SyncGameSave`. Se nenhum for encontrado, explica como instalar um e permite escolher qualquer pasta, com aviso claro de que uma pasta só local não protege contra perda do PC.
4. **Verificação da pasta**: testa escrita, leitura e espaço livre. Se a pasta estiver só na nuvem (arquivos sob demanda), avisa e mostra como mudar.
5. **Nome deste PC**: sugerido a partir do nome do computador.
6. **Adicionar jogos**: atalho para "Procurar jogos numa pasta" ou "Adicionar um jogo". Pode pular.
7. **Pronto**: resumo do que foi configurado.

## Reconhecimento de jogos (P0: um jogo por vez; P1: varredura)

Decisão: **reconhecer pela pasta do jogo**, com o executável como confirmação. Motivo: o manifesto do Ludusavi indexa os jogos pelo nome da pasta de instalação (`installDir`), que é estável. Nomes de executáveis variam muito (`game.exe`, `LAUNCHER_x64.exe`, `UnityCrashHandler64.exe`).

**Adicionar um jogo**: o usuário escolhe a pasta (ou o `.exe`; nesse caso, usa a pasta dele e as pastas acima, até a raiz da biblioteca).

1. Comparar o nome da pasta com `install_dirs` do catálogo (sem acentos e sem diferenciar maiúsculas).
2. Se não achar: busca FTS5 pelo nome da pasta limpo (tirar versões, `v1.2`, `GOTY`, `[GOG]`, sufixos de loja) e mostrar os 5 melhores candidatos.
3. Confirmar com os arquivos de `launch` do manifesto e com os caminhos de save que existem no disco (mais pontos se o save existe).
4. Escolher o executável: o do manifesto (`launch`); senão, o maior `.exe` da pasta, ignorando instaladores, `crashhandler`, `unins*`, `redist`, `vcredist`, `dxsetup`. O usuário pode trocar.
5. Mostrar o resultado com o grau de certeza. Alta certeza: um clique para confirmar. Baixa: lista de candidatos e busca.

**Varrer uma pasta de jogos**: o usuário escolhe uma ou mais pastas (ex.: `C:\Games`). O app percorre as subpastas (profundidade configurável, padrão 2), aplica o reconhecimento acima em paralelo, sem travar a interface, e mostra uma lista com caixas de seleção: "Reconhecidos", "Precisam de confirmação" e "Não reconhecidos". O usuário marca e adiciona todos de uma vez; o primeiro backup de cada jogo é feito em seguida, com progresso real. As pastas varridas ficam salvas para novas varreduras.

Também na varredura: pastas de Steam/GOG/Epic conhecidas podem ser lidas pelos manifestos das lojas no futuro (P2).

## Som (P1)

- **Fundo ambiente**: trilha calma e contínua, no estilo pad/ambient, sem melodia marcante, sem batida e sem picos de volume, para não cansar depois de horas. Loop longo (≥ 3 minutos) e sem emenda perceptível, ou gerada em tempo real (WebAudio: camadas de pads lentos com variação aleatória suave). Volume padrão baixo, controle próprio, opção de desligar, e pausa automática quando um jogo está rodando ou a janela está minimizada.
- **Efeitos**: inspirados na sensação dos menus do PS5 (suaves, curtos, "vítreos"), mas **criados do zero**, sem copiar nem samplear sons da Sony. Efeitos para: mover foco, confirmar, voltar, abrir painel, trocar seção, erro, sucesso do sync e conquista. Duração de 40–250 ms, ataque suave, sem frequências agudas estridentes, e variação leve de altura no "mover foco" para não ficar repetitivo.
- Volume separado para fundo e efeitos; respeitar silêncio quando o jogo está em primeiro plano.
- Formato: arquivos `.ogg`/`.opus` pequenos no pacote do tema, para que temas troquem os sons. Licença dos sons registrada na tela "Sobre".
- Teste: ouvir por pelo menos 1 hora seguida antes de aprovar o fundo.

## Decisão de stack (análise original)

A análise abaixo comparou Tauri e Electron com Vue. A decisão final trocou Vue por React; o restante continua valendo.

### Recomendação: Tauri 2 (núcleo em Rust) + Vue 3 + TypeScript + Vite

| Critério | Electron (atual) | Tauri 2 |
| --- | --- | --- |
| Memória em repouso | ~150–250 MB (Chromium próprio) | ~40–90 MB (usa o WebView2 do Windows) |
| Instalador | ~80–120 MB | ~5–15 MB |
| Rodar em segundo plano enquanto o jogo roda | pesa mais | leve, ideal para launcher |
| Operações de arquivo, hash, rastreio de processos | Node (ok) | Rust (forte, tipado, testável) |
| Leitura do controle com a janela sem foco (botão Guide) | precisa de módulo nativo | `gilrs`/XInput direto no Rust |
| Igualdade de renderização entre SOs | igual em todos | WebView2 no Windows é Chromium; no Linux (WebKitGTK) blur e animações pesadas são mais lentos |
| Curva de aprendizado | nenhuma | Rust no backend |

Motivos da recomendação:

- O launcher fica aberto durante o jogo. Gastar pouca RAM e CPU importa.
- A parte crítica (sync sem perder save) fica num núcleo Rust isolado, com testes fortes e sem depender da interface.
- Vue já é conhecido e quase não há interface a migrar; manter Vue reduz risco. Svelte ou Solid seriam um pouco mais leves, mas a diferença não justifica aprender outro framework junto com Rust.
- O protótipo HTML + CSS + JS vira a base visual direto, porque o Tauri renderiza HTML.

Alternativas consideradas:

- **Manter Electron**: termina mais rápido, mas carrega o custo de memória e tamanho para sempre. Válido se Rust for um bloqueio.
- **Tauri com sidecar Node**: mantém o código TS do backend, mas soma um runtime Node ao instalador e perde o ganho principal.
- **Usar o Ludusavi como motor (CLI `ludusavi backup/restore --api` ou `wrap`)**: o Ludusavi já faz restaurar → abrir → copiar. Serve como referência e plano B, mas o controle fino sobre o fluxo seguro (snapshots, conflitos, travas) e a interface ficariam limitados à CLI dele.
- **App nativo (WinUI/Qt/Flutter)**: mais trabalho para o visual animado e para temas em CSS.

Decisão registrada em `docs/adr/0001-stack-tauri-react.md`.

### Estrutura proposta (Tauri)

```
src-tauri/
  crates/
    sgs-core/      # biblioteca Rust pura: manifesto, catálogo, resolução de caminhos, snapshots, sync, travas
    sgs-cli/       # CLI fina sobre o core (sgs run <jogo>, sgs backup, sgs restore, sgs status)
  src/             # app Tauri: comandos, eventos, processos, controle, bandeja
src/               # frontend React (a partir do protótipo)
prototype/         # protótipo HTML + CSS + JS (fase 0)
```

## Catálogo de jogos: fonte e persistência

### Fontes

| Dado | Fonte | Observação |
| --- | --- | --- |
| Nome, pastas de instalação, caminhos de save e registro, ids da Steam/GOG | Manifesto do Ludusavi (YAML, ~30 MB, licença MIT, atualizado com frequência a partir da PCGamingWiki) | Fonte principal. Baixado em tempo de execução, nunca versionado. |
| Regras próprias do usuário | `library.db` (tabela de regras personalizadas) | Sobrescrevem ou completam o manifesto, como os "jogos personalizados" do Ludusavi. |
| Arte (capa, hero, logo, ícone) | Steam CDN pelo `steam.id`; SteamGridDB com chave do usuário; imagem própria | Cache em disco. |
| Descrição, desenvolvedora, ano, gêneros | Steam Store (`appdetails`) quando há `steam.id`; IGDB opcional no futuro | O manifesto não traz esses dados. Cache em `library.db`. |
| Conquistas | Provedores (`AchievementProvider`) | Ver seção "Conquistas". |

### Persistência para busca rápida

- Arquivo separado `catalog.db` (SQLite), descartável e reconstruível. A biblioteca do usuário fica em `library.db`, que nunca é apagado por uma atualização do catálogo.
- Atualização: o Rust baixa o manifesto com `ETag` (só baixa se mudou), lê o YAML, grava num arquivo temporário numa única transação e troca o arquivo de forma atômica. A interface continua usando o catálogo antigo durante a atualização.
- Tabelas: `games` (id, nome, nome normalizado sem acentos, ids externos), `install_dirs` (nome de pasta em minúsculas, com índice B-tree para identificar um `.exe` pela pasta), `save_rules` (caminho com marcadores, tags, SO, loja), `registry_rules`.
- Busca: tabela virtual FTS5 com `tokenize='trigram'` sobre nome, aliases e pastas. Encontra trechos no meio do nome e é rápida (milissegundos para dezenas de milhares de jogos). Para buscas com menos de 3 caracteres, usar `LIKE 'ab%'` no índice do nome normalizado.
- Execução: consultas no Rust em thread de bloqueio (`spawn_blocking`), nunca na thread da interface. A interface espera ~80 ms após a digitação (debounce) e descarta respostas antigas por id de requisição.
- Alternativas descartadas: manter o YAML em memória (lento para abrir e ~200 MB de RAM); índice em memória com busca difusa (rápido, mas gasta RAM e demora para montar); Tantivy (motor de busca completo, demais para o volume); servidor de busca externo.
- Medir antes de fechar a Fase 1: tempo de montar o `catalog.db`, tamanho em disco e tempo das 20 buscas mais comuns.

O `sgs-core` não conhece a interface. Assim os testes de segurança rodam com `cargo test`, sem janela.

## Projeto do sync seguro (núcleo do produto)

### Princípios

1. **Nada é sobrescrito sem snapshot anterior.** Antes de restaurar da nuvem, copia o save local. Antes de enviar para a nuvem, o estado anterior da nuvem continua num snapshot imutável.
2. **Snapshots são imutáveis.** Uma versão escrita nunca é alterada; só a retenção apaga versões antigas, e nunca as últimas N.
3. **Commit por último.** Os arquivos do snapshot são gravados e conferidos (hash) antes de atualizar o ponteiro `head.json`. Uma interrupção no meio deixa só um snapshot incompleto, que é ignorado.
4. **Comparação em três vias.** Base (último estado sincronizado neste PC), local e nuvem. Conflito real nunca é resolvido sozinho.
5. **Nunca sincronizar com o jogo aberto.**
6. **Pasta legível por humanos.** O usuário consegue recuperar um save na mão, só com o Explorer, como já faz hoje.

### Layout na pasta da nuvem

```
<PastaNuvem>/SyncGameSave/
  devices.json                         # PCs conhecidos (id, nome, último acesso)
  games/<slug-do-jogo>/
    game.json                          # id, nome, ids externos, regras de caminho
    head.json                          # snapshot atual: id, hash do índice, dispositivo, data
    lock.json                          # "jogando em PC-X desde HH:MM" + heartbeat
    snapshots/
      2026-11-05T21-14-03Z_PC-SALA/
        index.json                     # lista de arquivos: caminho lógico, tamanho, mtime, sha256
        files/<caminho-lógico>/...     # arquivos do save, em texto claro
```

Local (fora da nuvem): `%LOCALAPPDATA%/SyncGameSave/`

- `library.db` (SQLite): biblioteca, tempo de jogo, estado base de cada jogo, fila de operações. **Nunca colocar o banco na pasta da nuvem** (risco de corromper com o sync do OneDrive).
- `backups/<slug>/<data>_<motivo>/`: cópias locais antes de ações perigosas (restore, resolver conflito, apagar jogo). Retenção própria e separada.
- `journal/`: operação em andamento, para retomar ou avisar após queda de energia ou crash.

### Fluxo ao abrir um jogo

1. Conferir se a pasta da nuvem existe e está acessível. Se não, perguntar: "Jogar sem sincronizar" ou "Cancelar".
2. Ler `lock.json`. Se outro PC está jogando com heartbeat recente, avisar e pedir confirmação.
3. Conferir o snapshot de `head.json`: todos os arquivos existem e os hashes batem com o `index.json`. No OneDrive com "Arquivos sob demanda", ler os arquivos força o download; se o hash não bate, a nuvem ainda está sincronizando: mostrar "Aguardando a nuvem terminar de baixar o save…" com opção de esperar ou cancelar.
4. Calcular o índice do save local e comparar em três vias:

| Local mudou? | Nuvem mudou? | Ação |
| --- | --- | --- |
| não | não | abrir o jogo |
| não | sim | backup local → restaurar snapshot da nuvem → conferir hash → abrir |
| sim | não | (jogou fora do launcher) criar snapshot na nuvem → abrir |
| sim | sim | **conflito**: tela com data, PC, tamanho e tempo de jogo de cada lado; o usuário escolhe; o lado descartado fica em backup |

5. Gravar `lock.json` e abrir o jogo.

### Durante o jogo

- Rastrear o processo e os filhos (Job Object no Windows), e também pelo caminho do executável. Muitos jogos abrem por um launcher que fecha logo (ex.: `LAUNCHER_x64.exe` do Aggelos).
- Atualizar o heartbeat do `lock.json` a cada poucos minutos.
- Contar o tempo de jogo.
- Opcional: janela minimizada na bandeja, sem renderizar a interface, para gastar quase nada.

### Fluxo ao fechar o jogo

1. Esperar alguns segundos (jogos gravam o save ao sair) e conferir se nenhum processo do jogo continua rodando.
2. **Copiar independentemente do código de saída** (hoje só copia se for 0; um crash também pode ter save novo).
3. Calcular o índice local. Se igual à base, só liberar a trava.
4. Se mudou: gravar `snapshots/<novo>/files`, gravar `index.json`, reler e conferir os hashes, e só então atualizar `head.json`. Atualizar a base local. Liberar a trava.
5. Notificar: "Save enviado para a nuvem" ou o problema e o próximo passo.

### Outros cuidados

- Detectar cópias de conflito do próprio OneDrive (`arquivo-NOMEDOPC.ext`) e avisar.
- Recomendar marcar `SyncGameSave` como "Sempre manter neste dispositivo" no OneDrive.
- Retenção configurável: últimas N versões (padrão 20) + uma por dia nos últimos 30 dias. Nunca apagar o snapshot do `head` nem o que está em uso.
- Caminhos de save: resolver os marcadores do manifesto do Ludusavi (`<base>`, `<home>`, `<winAppData>`, `<winLocalAppData>`, `<winDocuments>`, `<winPublic>`, `<storeUserId>` etc.) e as variáveis do Windows. Guardar os arquivos no snapshot por **caminho lógico** (com o marcador), para restaurar em outro PC com outro usuário ou outra pasta de instalação.
- Modo "simular": mostra o que seria copiado e restaurado, sem gravar nada.
- Botão "Restaurar esta versão" em qualquer snapshot, sempre com backup local antes.
- Manifesto do Ludusavi: baixar e atualizar em tempo de execução (com cache e ETag) a partir de `https://raw.githubusercontent.com/mtkennerly/ludusavi-manifest/master/data/manifest.yaml`, em vez de manter 30 MB no repositório. Exibir a atribuição (MIT) na tela "Sobre".

### Cenários de teste do sync (obrigatórios antes da 1.0)

Testes automatizados no `sgs-core` com pastas temporárias:

- Cada linha da tabela de três vias.
- Queda no meio da escrita do snapshot (simular erro após N arquivos): `head` continua no anterior e nada local é perdido.
- Arquivo de save travado pelo jogo ou pelo antivírus: repetir com espera e, se falhar, avisar sem perder nada.
- Disco cheio na nuvem ou no local.
- Nuvem parcialmente sincronizada (hash não bate).
- Dois PCs abrindo o mesmo jogo (trava).
- Caminho com acentos, espaços, nomes muito longos (>260 caracteres) e maiúsculas/minúsculas diferentes.
- Save que é pasta inteira, save com arquivos apagados entre sessões, save vazio.
- Jogo que fecha com código de erro e jogo aberto por launcher intermediário.
- Restauração em outro "PC" com outro `%USERPROFILE%`.

## Funcionalidades da biblioteca

- **Adicionar jogo** (assistente): escolher o `.exe` → identificar o jogo pelo nome da pasta (`installDir` do manifesto) e pela busca por nome → mostrar os caminhos de save encontrados e os arquivos que existem → permitir adicionar ou remover caminhos na mão → fazer o primeiro snapshot.
- **Varredura de pastas** (ex.: `C:/Games`): sugerir vários jogos de uma vez.
- **Arte**: Steam CDN quando há `steam.id` no manifesto; SteamGridDB (chave de API do usuário) para capas, heróis, logos e ícones; imagem própria do usuário. Cache local.
- **Por jogo**: argumentos, pasta de trabalho, sync ligado/desligado, scripts antes e depois, favoritar, ocultar.
- **Tempo de jogo** e "última vez jogado".
- **Importar** de uma configuração do Ludusavi e de jogos que não são da Steam.

## Conquistas

Pesquisa curta antes de implementar (spike). Fontes possíveis:

1. **Steam Web API**: lista e ícones das conquistas pelo `steam.id` (esquema público com chave de API). Para jogos sem Steam, serve como catálogo e checklist manual.
2. **RetroAchievements**: API oficial para jogos de emulador, se houver suporte a emuladores no futuro.
3. **Conquistas do próprio HUB**: marcos de uso (horas jogadas, jogos zerados marcados pelo usuário, sequências de sync sem erro). Funcionam para qualquer jogo.
4. **Provedores de conquistas por plugin**: interface `AchievementProvider` (listar, progresso, desbloqueio), para adicionar fontes depois sem mudar o núcleo.

Mostrar no card do jogo o progresso (ex.: "12 de 40 · 30%") e um aviso discreto ao desbloquear.

### Conquistas da Steam com visual inspirado no PS5

Fonte principal para jogos da Steam. O visual segue a ideia dos troféus do PS5, mas com desenho próprio: sem copiar ícones, nomes ("troféu de platina") ou sons da Sony.

**Dados (Steam Web API, chave do usuário, cache em `library.db`):**
- `ISteamUserStats/GetSchemaForGame`: nome, descrição, ícones (desbloqueado e bloqueado) e se a conquista é oculta.
- `ISteamUserStats/GetPlayerAchievements`: o que o usuário desbloqueou e quando. Exige o SteamID do usuário e o perfil de jogos público. Se o perfil for privado, o app explica como mudar e mostra só o catálogo.
- `ISteamUserStats/GetGlobalAchievementPercentagesForApp`: raridade global de cada conquista.
- Atualização: ao fechar o jogo e em segundo plano, com limite de chamadas. Funciona sem internet usando o cache.

**Níveis por raridade** (calculados pelo HUB, porque a Steam não tem níveis): Comum (> 50%), Incomum (20–50%), Rara (5–20%), Ultrarrara (< 5%). Cada nível tem uma cor e um selo próprio. Completar 100% de um jogo dá um selo especial de "jogo completo".

**Telas:**
- Card de conquistas no Início e aba Conquistas nos detalhes: anel de progresso, contagem por nível, filtros (todas, desbloqueadas, bloqueadas, por raridade) e ordenação (recentes, raridade, nome).
- Item da lista: ícone grande, nome, descrição, data do desbloqueio, % de jogadores e selo do nível. Conquistas ocultas mostram "Conquista oculta" até o desbloqueio, com opção de revelar.
- Notificação de desbloqueio (quando detectada ao fechar o jogo ou na sincronização): cartão no canto com ícone, nome e nível, e um som próprio do nível.
- Tela "Minhas conquistas": nível do perfil, total por nível, jogos completos e últimos desbloqueios de todos os jogos.

Prioridade: P2 (depois do MVP). Os dados podem ser buscados já na P1 junto com os metadados da Steam, para a tela ficar pronta mais cedo.

## Temas

- Toda cor, raio, fonte, sombra, blur e duração de animação vem de **tokens CSS** (`--sgs-*`) desde o protótipo. Isso torna os temas possíveis sem refazer a interface.
- Pacote de tema: `themes/<id>/theme.json` com nome, autor, versão, versão mínima do app e tokens. A escolha fica no banco local e vale ao reabrir.
- Segurança: temas não executam JS nem CSS. Só tokens `--sgs-*` são aplicados, e valores com `url()`, `@`, `;` ou caracteres de regra são recusados. Ver [ADR 0005](../adr/0005-formato-de-temas.md).
- Tema escuro (padrão) e claro oficiais servem de exemplo e de teste. `theme.css` e `assets/` (fundos, sons, fontes) ficam para uma decisão futura.

## Outras sugestões

- **CLI `sgs run <jogo>`**: permite usar o sync a partir da Steam (atalho "jogo não Steam"), Playnite ou Steam Deck, sem abrir o HUB.
- **Adicionar à Steam** como jogo não Steam com um clique (grava em `shortcuts.vdf`), para usar no Big Picture.
- **Modo portátil** (pasta de dados ao lado do executável).
- **Ícone na bandeja** com o estado do sync e "Sincronizar tudo agora".
- **Painel de saúde**: jogos sem save encontrado, caminhos que não existem mais, nuvem indisponível.
- **Linux/Proton** depois: resolver caminhos dentro do prefixo do Wine.
- **Atualização automática** pelo updater do Tauri.

## Interface: descrição para o protótipo (HTML + CSS + JS)

### Direção visual

- Escuro por padrão, "10-foot UI" (legível do sofá) que também funciona em janela no monitor.
- O fundo é a **arte hero do jogo em foco**, em tela cheia, com degradê escuro na base e à esquerda. Ao trocar de jogo, o fundo faz crossfade (300–400 ms).
- Cor de destaque derivada da arte (cor dominante extraída da imagem), com fallback para um azul.
- Superfícies em vidro: fundo translúcido + `backdrop-filter: blur()`; com redução de transparência ou em hardware fraco, usar fundo sólido.
- Tipografia sem serifa geométrica (ex.: "Outfit" ou "Manrope"), pesos 300/500/700. Tamanhos com `clamp()` baseados na altura da janela.
- Movimento: foco com escala 1.0 → 1.08, sombra e anel luminoso; transições de 180–250 ms com `cubic-bezier(.2,.8,.2,1)`. Com `prefers-reduced-motion`, sem escala nem crossfade.
- Sons curtos de navegação, confirmar e voltar, com volume e opção de desligar.
- Português brasileiro em todos os textos; `lang="pt-BR"`; datas e números com `Intl` em `pt-BR`.

### Telas

**1. Início (Home)**

- Barra superior: à esquerda, abas "Jogos" e "Biblioteca"; à direita, ícone de estado da nuvem (ok / sincronizando / erro), busca, configurações, perfil e relógio.
- Faixa horizontal de jogos logo abaixo da barra: o primeiro item é "Biblioteca" (ícone de grade); depois os jogos por "jogado recentemente". Itens não focados são quadrados pequenos (ícone/capa); o item focado cresce e mostra o nome ao lado.
- Área central: logo do jogo (ou nome em tipografia grande), botão primário **"Jogar"**, botão "…" (mais opções), e uma linha de estado: "Save sincronizado há 2 h · PC-SALA" ou "Save local mais novo, será enviado".
- Abaixo (ao descer com o direcional): cards em faixa horizontal:
  - **Save**: estado, última versão, número de versões, botão "Ver histórico".
  - **Conquistas**: progresso em anel + últimas 3 desbloqueadas.
  - **Tempo de jogo**: total, última sessão, gráfico simples dos últimos 14 dias.
  - **Informações**: pasta de instalação, caminhos de save, botão "Abrir pasta".

**2. Biblioteca**

- Grade de capas verticais (proporção 2:3), com filtros no topo: "Todos", "Instalados", "Favoritos", "Com sync", "Ocultos"; ordenar por nome, último jogado e tempo de jogo.
- Busca com teclado na tela quando o controle estiver em uso.
- Estado vazio: "Nenhum jogo ainda" + botões "Adicionar jogo" e "Procurar jogos numa pasta". Filtro sem resultado é diferente: "Nenhum jogo com este filtro" + "Limpar filtros".

**3. Detalhe do jogo / Histórico de saves**

- Linha do tempo vertical de snapshots: data relativa e absoluta, PC de origem, tamanho, quantidade de arquivos, motivo ("ao fechar o jogo", "antes de restaurar", "manual").
- Ações por versão: "Restaurar esta versão", "Abrir na pasta", "Comparar com o atual" (lista de arquivos adicionados, alterados, removidos).
- Confirmar restauração com texto claro: "O save atual será guardado como backup antes. Você pode voltar a ele depois."

**4. Abrindo o jogo (overlay em tela cheia)**

- Arte hero + logo + passos com estado real: "Conferindo a nuvem" → "Baixando save de PC-SALA" → "Abrindo o jogo". Cada passo mostra ok, erro ou aguardando. Sem barra de progresso inventada.
- Em erro: explicação simples e ações ("Tentar de novo", "Jogar sem sincronizar", "Cancelar").

**5. Jogo em execução**

- Tela mínima: "Jogando Animal Well · 1 h 12 min", "O save será enviado ao fechar o jogo". Botão "Forçar envio agora" desabilitado com o motivo enquanto o jogo roda.
- Ao fechar: toast "Save enviado para a nuvem ✓" ou erro com ação.

**6. Conflito de save (diálogo)**

- Duas colunas: "Neste PC" e "Na nuvem (PC-QUARTO)": data da última alteração, tempo de jogo na sessão, tamanho, quantidade de arquivos.
- Ações: "Usar o deste PC", "Usar o da nuvem", "Decidir depois (não abrir o jogo)". Texto fixo: "O save não escolhido fica guardado como backup."
- Foco inicial em "Decidir depois", para um aperto acidental não escolher um lado.

**7. Adicionar jogo (assistente em passos)**

1. Escolher executável (diálogo de arquivo) ou pasta para varrer.
2. Confirmar o jogo identificado (capa + nome + "Não é este? Buscar").
3. Caminhos de save: lista com marcadores ("encontrado, 3 arquivos, 120 KB" / "não encontrado"), "Adicionar caminho", "Remover".
4. Resumo + "Fazer o primeiro backup e concluir".

**8. Central rápida (botão Guide/Home do controle ou tecla F1)**

- Painel na base da tela, sobre o que estiver aberto: estado do sync de todos os jogos, notificações, "Sincronizar tudo agora", "Voltar ao jogo", volume dos sons, desligar/sair.

**9. Configurações**

- Seções em lista à esquerda, conteúdo à direita:
  - **Nuvem**: pasta escolhida, espaço usado, teste "Verificar pasta", nome deste PC.
  - **Backups e versões**: retenção, pasta dos backups locais, "Abrir pasta".
  - **Biblioteca**: pastas para varrer, chave da SteamGridDB, atualizar manifesto.
  - **Controles**: estilo dos ícones (Xbox, PlayStation, teclado, automático), vibração, mapa de teclas.
  - **Aparência**: tema, transparência, movimento reduzido, sons.
  - **Sobre**: nome, versão, build, licenças e atribuição do manifesto do Ludusavi.

**10. Primeiro uso**

- Ver a seção "Assistente de primeiro uso (P0)", que detalha cada passo.

### Modelo de entrada (controle + teclado + mouse)

- **Navegação espacial** única: um motor de foco escolhe o próximo elemento pela direção (cima, baixo, esquerda, direita), entre "grupos" (barra superior, faixa de jogos, ação, cards). Cada grupo lembra o último item focado.
- Mapeamento:

Mapa usado no protótipo (`prototype/src/input.js`):

| Ação | Controle (Xbox / PlayStation) | Teclado | Mouse |
| --- | --- | --- | --- |
| Mover foco | direcional / analógico esquerdo | setas | passar o mouse foca |
| Confirmar / Jogar | A / ✕ | Enter / Espaço | clique |
| Voltar / Fechar | B / ○ | Esc / Backspace | botão voltar do mouse |
| Opções do jogo | Y / △ | O / tecla Menu | botão "…" |
| Filtros | X / □ | F | botão "Filtros" |
| Buscar | View / Create | `/`, Ctrl+K, Ctrl+F | lupa |
| Menu principal | Menu / Options | M | ícone ☰ |
| Seção ou aba anterior/próxima | LB / RB (L1 / R1) | Q / E | clique na aba |
| Pular vários itens | LT / RT (L2 / R2) | Page Up / Page Down | roda do mouse |
| Central rápida | Guide / PS (no app final) e R3 | F1 | sino na barra |
| Lista de atalhos | L3 | `?` | menu → Atalhos |
| Início / Biblioteca | — | G / L | barra lateral |
| Adicionar jogo | — | N | barra lateral |
| Saves e conquistas | — (pelo botão ou pelas opções) | H | botão |
| Configurações | — | Ctrl + , | barra lateral |

A legenda de botões fica no canto inferior direito, muda conforme a tela e o foco, e cada item também é clicável.

- **Legenda de botões** no canto inferior direito, que muda para o último dispositivo usado (Xbox, PlayStation ou teclado).
- Repetição ao segurar o direcional: espera 350 ms, depois a cada 80 ms.
- Foco sempre visível (anel com contraste ≥ 3:1), alvos com ≥ 44 px, leitores de tela com `aria-label` e regiões `aria-live` para os estados do sync.
- No protótipo: usar a Gamepad API do navegador (`navigator.getGamepads()` em `requestAnimationFrame`). No app final, ler o controle no Rust e enviar eventos à interface, para funcionar também sem foco na janela.

### Arquivos do protótipo

```
prototype/
  index.html        # estrutura base: barra lateral, barra superior, telas, legenda
  vite.config.js    # Vite+ (porta 5199)
  src/tokens.css    # variáveis do tema (--sgs-*): cores, raios, fontes, durações
  src/styles.css    # layout e componentes
  src/focus.js      # motor de navegação espacial, camadas (overlays) e memória de foco
  src/input.js      # teclado + Gamepad API → ações; troca de legenda por dispositivo
  src/main.js       # telas, overlays, fluxos simulados (jogar, conflito, restaurar)
  src/data.js       # 19 jogos (arte da Steam), 1 de emulador e 1 sem arte com nome longo
  src/icons.js      # ícones SVG de traço
```

Dados fictícios devem incluir casos de borda: nome muito longo, jogo sem arte (usar placeholder com iniciais e cor), jogo com conflito, jogo com nuvem indisponível, biblioteca vazia.

Verificação do protótipo: capturas headless em 1920×1080, 1280×720 e 1366×768 (janela), mais um teste manual só com controle.

## Passos

### Fase 0 — Decidir e prototipar

- [x] Decidir a stack: Tauri 2 + React (`docs/adr/0001-stack-tauri-react.md`).
- [x] Propor a fonte e a persistência do catálogo (`docs/adr/0002-catalogo-de-jogos-sqlite-fts5.md`).
- [x] Criar o protótipo em `prototype/` conforme a seção de interface.
- [ ] Revisar o protótipo com controle real (Xbox e PlayStation) e teclado; ajustar.
- [ ] Prototipar o assistente de primeiro uso e a tela de resultado da varredura de pastas.
- [x] Confirmar o ADR 0002 (catálogo) (aprovado pelo usuário em 2026-11-05).
- [x] Registrar o layout da pasta da nuvem em `docs/adr/0003-formato-da-pasta-da-nuvem.md`.

### Fase 1 — Núcleo de sync (`sgs-core` + `sgs-cli`)

- [x] Remover todo o código Electron (autorizado), inclusive o `css.postcss` do protótipo.
- [x] Iniciar o projeto Tauri 2 + React + TypeScript com bun (tela mínima com nome e versão; os ícones ainda são os do modelo do Tauri e precisam ser trocados).
- [x] Montar o `catalog.db` a partir do manifesto do Ludusavi (download com ETag, troca atômica) e medir tempos (`sgs-catalog`; números no ADR 0002).
- [x] Caminhos lógicos: resolver/converter marcadores (`paths`). `Env` usa as pastas do sistema; há teste para a pasta Documentos.
- [x] Índice de arquivos com SHA-256; comparação em três vias (`sgs-core`: `index`, `sync`).
- [x] Snapshots imutáveis, commit por `head.json`, verificação de hash e restauração com backup. Retenção disponível no núcleo e coberta por teste; o MVP não apaga snapshots automaticamente.
- [x] Backups locais (`engine`) e journal de operações (`journal`).
- [x] `lock.json` com heartbeat (`lock`).
- [x] CLI `sgs run|backup|restore|status|list` com saída JSON opcional (`sgs-cli`).
- [x] Cenários de teste do sync passando com `cargo test`, inclusive jogo aberto por launcher intermediário (a CLI acompanha o programa e os processos descendentes; `--quiet-secs`, padrão 5). A busca pelo executável na pasta do jogo está implementada e testada; Job Object continua pendente.
- [ ] Uso real por 1–2 semanas pela CLI com atalhos (Steam, Epic ou outro), nos jogos atuais. Primeira sessão real feita em 2026-11-05 com o Rogue Legacy (Epic, sem save na nuvem). **Em andamento, em paralelo à Fase 2** (decisão do usuário em 2026-11-05).

### Fase 2 — App e biblioteca

- [x] SQLite local (`library.db`) com migrações (`sgs-library`: jogos, caminhos de save, estado do sync, configurações).
- [x] Backend do assistente de primeiro uso (`sgs-service`: verificar a pasta, registrar este PC sem apagar outros, detectar dados de outro PC). Tela em `src/screens/Wizard.tsx`, integrada ao app; fluxo coberto pelo E2E de primeiro uso.
- [x] Reconhecimento pela pasta do jogo e varredura de pastas (`sgs-catalog::recognize`; testado com o Rogue Legacy real). Tela de adicionar em `src/screens/AddGame.tsx` implementada. Faltam descoberta por mudança e interface de varredura em lote.
- [x] Abrir jogo com rastreio de processos (`sgs-core::process`, por árvore de processos). Tempo de jogo feito (`library.db` esquema 2). Falta: Job Object. Heartbeat da trava durante o jogo feito (a cada 2 min).
- [x] Integração do fluxo de abrir/fechar com o `sgs-core` e eventos para a interface (comandos Tauri sobre o `sgs-service`; evento `jogo-estado`).
- [x] Arte (Steam CDN + SteamGridDB) com cache: capa baixada uma vez em `art`, Steam primeiro e SteamGridDB pelo nome quando há chave configurada em Configurações. Verificado por testes unitários do parsing e recompilação; a busca real depende de uma chave do usuário.
- [x] Bandeja e notificações (fechar a janela esconde o app na bandeja; menu Abrir/Sair; notificação ao fim do sync, conflito e erro). Verificado: o app abre sem erro; o ícone da bandeja não foi inspecionado visualmente.

### Fase 3 — Interface final

- [x] Tokens e motor de foco/entrada do protótipo portados para o app (`src/tokens.css`, `src/nav/`). Visual do protótipo portado: Início com arte de fundo, logo, faixa de jogos por seção (Q/E) e legenda; Biblioteca em grade de capas (`src/screens/Inicio.tsx`, `src/telas.css`). Faltam busca, filtros, menu e central rápida.
- [x] Controle lido no Rust (`gilrs`) + motor de foco. A lógica (botões, zona morta do analógico, repetição, só com a janela focada) tem testes. **Não testado com um controle de verdade**: nenhum estava conectado quando verifiquei.
- [x] Telas de histórico (Versões), conflito, jogo aberto em outro PC, configurações (com as credenciais da Steam e a chave do SteamGridDB), primeiro uso, adicionar jogo, conquistas do HUB e "Sobre" (nome, versão e build). Tela de conquistas com anel de progresso, níveis por raridade, filtros, ordenação e conquistas ocultas. Verificadas por capturas em 390×844 e 1280×720, sem erro de console nem rolagem horizontal.
- [x] Testes E2E dos fluxos principais (`bun run e2e`, Playwright, 21 testes na última execução local: primeiro uso, adicionar jogo, jogar, conflito, outro PC, versões, conquistas do jogo e do HUB, configurações, tema, remover, teclado, layout no celular). Rodam com dados simulados. Além deles, `bun run e2e:app` abre o app Tauri de verdade (dados isolados por `SGS_DATA_DIR`) e confere assistente, pasta de sync, reconhecimento do Rogue Legacy real, jogar e versões.
- [x] `.github/preview.png` (1280×640, tema claro) e README atualizados.

### Fase 4 — Som, conquistas e temas (P1/P2)

- [x] Fundo ambiente e efeitos sonoros próprios, sintetizados com WebAudio (`src/som.ts`): volume separado, padrão baixo, desligável, silêncio com jogo aberto ou janela escondida; limites de duração e frequência verificados por testes (`bun test`). **Pendente, só um humano faz:** ouvir o fundo por 1 hora seguida antes de aprovar. Os sons também não foram ouvidos por mim (sem saída de áudio aqui).

- [x] Spike de fontes de conquistas e `AchievementProvider` com a Steam Web API (`sgs-achievements`): catálogo, raridade e progresso, cache em `library.db`, níveis por raridade, credenciais guardadas no banco local e tela de conquistas. Ícones vêm do CDN da Steam, liberados na CSP. O cache atualiza em segundo plano e após fechar um jogo Steam.
- [x] Conquistas do HUB (marcos de uso: horas jogadas, jogos na biblioteca, jogos completos e sequências de sync sem erro), com progresso calculado da biblioteca e do cache de conquistas e sequência guardada no banco local. Tela própria em **Meu progresso**, verificada por capturas e teste E2E.
- [x] Formato de pacote de tema (`themes/<id>/theme.json`), carregamento seguro só de tokens `--sgs-*`, escolha persistida e temas embutidos escuro e claro, com seletor rápido na barra superior. Capturas em 390×844 e 1280×720 sem rolagem horizontal; decisão no [ADR 0005](../adr/0005-formato-de-temas.md). `theme.css`/`assets/` continuam reservados para decisão futura conforme o ADR.

### Fase 5 — Distribuição

- [x] Versão `1.0.0` nos metadados (app, Tauri e pacote) e changelog consolidado em `changelog/1.0.0.md`.
- [ ] Concluir a validação da distribuição: instalador NSIS (Windows, português do Brasil, por usuário) e AppImage (Linux), com atualização automática assinada (ver [ADR 0006](../adr/0006-distribuicao-e-atualizacao.md)). O NSIS foi gerado e verificado nesta máquina; o AppImage é gerado pelo mesmo comando no Linux e ainda não foi executado aqui. Publicar o release no GitHub depende de autorização.

## Tentativas que falharam

- Copiar só quando o código de saída é 0 (`GameService.ts`): perde o save quando o jogo fecha com erro ou quando o `.exe` é um launcher intermediário.

## Dúvidas abertas

- Steam Deck (SteamOS) entra junto com o Ubuntu ou depois?

## Retomada e sequência de implementação (2026-11-07)

Estado confirmado: PR #10 integrado na `main` no commit `485c85f`; branch antiga removida. Não procurar nem recriar `feat/fase1-nucleo-sync`. Os testes locais passaram antes da integração: workspace Rust, 14 testes Bun, build e 21 E2E. Isso não comprova conclusão do MVP nem validação de distribuição.

Ao receber “siga implementando”, começar pelo item 1 abaixo. Ler o fluxo indicado, implementar e verificar; atualizar este plano com o resultado e avançar na sequência. As validações humanas abaixo não impedem implementar os itens de código. Não usar a ausência da branch antiga como bloqueio.

### 1. Concluir o acompanhamento de processos no Windows — próximo trabalho

- Estado: árvore de processos e busca pelo executável na pasta existem em `src-tauri/crates/sgs-core/src/process.rs`. Há proteção por data de início contra reutilização de PID. Ainda não há Job Object; polling pode perder filhos de launchers curtos.
- Implementar Job Object com recursos Windows, isolados por `cfg(windows)`. Vincular o processo antes que ele possa iniciar filhos (criação suspensa e retomada após associação), mantendo a busca por pasta para jogos relançados por outro launcher, como Steam.
- Preservar heartbeat, código de saída, diretório de trabalho, argumentos e intervalo de silêncio. Encerrar o HUB não deve matar o jogo por efeito do fechamento do handle. Definir tratamento explícito de falha ao criar/associar o Job Object, sem deixar um processo suspenso abandonado.
- Conferir todos os consumidores: `sgs-cli`, `sgs-service` e comandos Tauri. Testes existentes: `sgs-core/tests/parar_jogo.rs`, `processo_relancado.rs` e testes de launcher em `sgs-cli/tests/cli.rs`.
- Concluído quando: teste Windows com launcher curto e descendente fora da pasta acompanha o jogo até o fim; relançamento por pasta e parada continuam funcionando; teste de falha não deixa processo suspenso; workspace Rust passa. Registrar limites observados.

### 2. Corrigir a parada para seguir o fluxo especificado no P0

- Estado: `process::parar` sinaliza encerramento forçado; `sgs-service::stop_game`, comando `parar_jogo`, `api.pararJogo` e `App.tsx` expõem esse caminho. Falta pedido de fechamento normal e espera de 10 segundos antes de oferecer força.
- Implementar fechamento normal da janela do jogo no Windows; acompanhar o grupo durante a espera. Só disponibilizar encerramento forçado após o prazo e confirmação com aviso sobre saves. Manter a interface responsiva, evitar pedidos duplicados e tratar jogo já encerrado.
- Concluído quando: testes cobrem fechamento normal, jogo que ignora fechamento, confirmação/cancelamento e sync somente após o grupo terminar. Verificar diálogo e foco por E2E e capturas mobile/desktop.

### 3. Descobrir saves por mudança — concluído (2026-11-11)

- Estado: catálogo e seleção manual existem; não há fluxo de captura antes/depois.
- Implementar comparação de arquivos nas pastas definidas na seção “Onde encontrar o save”, com exclusão de caches/logs e erros de acesso tratados. Executar fora da thread da interface. Reutilizar resolução de caminhos lógicos; não sincronizar candidatos automaticamente.
- Expor pelo serviço e Tauri; adicionar ação em `AddGame.tsx`: registrar estado inicial, abrir jogo, aguardar fechamento, listar candidatos com evidência e permitir confirmar/editar. Não salvar caminhos sem confirmação; preservar o formulário em falhas.
- Concluído quando: testes com diretórios temporários cobrem novos/alterados/inalterados, acesso negado e exclusões; E2E cobre confirmação e cancelamento; inspeção mobile/desktop aprovada.
- Resultado: núcleo em `sgs-core/src/discover.rs` (captura/comparação por raízes com profundidade, filtro de ruído, caminhos lógicos com marcadores), serviço `discover_run`/`discover_candidates`, comandos Tauri `descobrir_save`/`candidatos_save` e painel em `AddGame.tsx` com cancelamento e confirmação manual. 8 testes do núcleo, teste do serviço com falso jogo, 3 testes E2E novos (25/25) e capturas 390×844/1280×720 aprovadas. Flake de `service.rs` irmãos com `ATIVOS` compartilhado tratado por retry no teste. Pendente da descrição original: exibir tamanho dos candidatos (evidência mostra contagem de arquivos; `DiscoveredSave.bytes` já carrega o dado no serviço).

### 4. Busca, filtros e ordenação da biblioteca — concluído (2026-11-11)

- Estado: `src/screens/Biblioteca.tsx` mostra grade ordenada por nome; faltam controles de busca/filtros.
- Implementar busca por nome e filtros que os dados atuais permitem, com ordenação e estados vazios distintos. Conferir modelo em `src/api.ts` e `sgs-library` antes de prometer favoritos/ocultos; esses campos exigem persistência e migração se ainda não existirem.
- Preservar consulta/filtros ao abrir e voltar; integrar foco e atalhos em `src/nav/`. E2E deve cobrir resultados, limpar filtros e retorno com contexto; inspecionar mobile/desktop e atualizar miniatura se a tela principal mudar.
- Resultado: busca por nome sem acentos, filtros Todos/Sincronizados/Sem sincronização/Nunca jogados (campos já existentes em `Game`; favoritos/ocultos continuam fora, pois não há persistência) e ordenação por nome, jogado recentemente e tempo de jogo. Estado vazio próprio ("Nenhum jogo com estes filtros" + Limpar filtros) e contagem "N de M jogos". Busca/filtro/ordem ficam no `App.tsx` para sobreviver a Início↔Biblioteca e diálogos. Atalhos `/` (search) e `F` (filter) ligados em `App.tsx`/legenda; nova semente no cenário preview (Sea of Stars nunca jogado). 2 testes E2E novos (27/27); capturas 1280×720 e 390×844 sem rolagem horizontal (chips rolam dentro da própria linha no celular).

### 5. Varredura em lote, menu e central rápida

- Em andamento: o resultado da varredura agora registra erros de leitura e reconhecimento por caminho, permitindo continuar com os demais jogos; testes de catálogo passaram. Falta expor esses erros e os resultados na interface, selecionar e adicionar os jogos, e validar os fluxos.
- Varredura: reutilizar `sgs-service::scan_library_folder` e comando `varrer_pasta`; criar resultado selecionável com confirmação dos candidatos, caminhos de save e erros por jogo. Não adicionar registros sem saves válidos.
- Menu/central: implementar as ações especificadas na seção de interface, usando estados reais do app. Sincronização não pode sobrescrever saves de jogo em execução. Preservar navegação por controle, teclado e retorno de foco.
- Entregar cada fluxo separadamente com regressões E2E, capturas e README/changelog correspondentes.

### 6. Distribuição e fechamento da entrega

- Ícones ainda precisam ser conferidos/substituídos conforme a marca do app; não tratar ícones do modelo Tauri como entrega final.
- Conferir metadados e changelog juntos: `1.0.0` está no código, mas ainda há fragmentos unreleased. Consolidar apenas o escopo efetivamente entregue no próximo fechamento de versão, seguindo as regras do projeto; não afirmar release publicado.
- Gerar e validar NSIS e AppImage nos respectivos sistemas. Configuração de AppImage não equivale a binário gerado/testado. Publicação continua dependendo de autorização específica.

## Validações externas ainda abertas

- Uso real por 1–2 semanas; registrar os resultados fornecidos pelo usuário.
- Controle físico Xbox/PlayStation e revisão de foco na interface real.
- Ouvir o fundo por uma hora e avaliar fadiga/loop.
- Consulta real ao SteamGridDB com credencial disponível, sem registrar segredos.
- Instalação e atualização assinada no Windows e Linux.

## Próximo passo

Implementar o item 5: varredura em lote, menu e central rápida. Itens 1–4 desta sequência já concluídos (Job Object em `a2ab3f9`; parada normal em `b38516b`; descoberta por mudança em `2b50137`; busca/filtros/ordenação em 2026-11-11).

### Acompanhamento de processos no Windows

Implementado Job Object com processo criado suspenso e retomada pela API ResumeThread. Os testes cobrem launcher curto com filho fora da pasta e falha de associação sem deixar processo suspenso; a busca por pasta e a parada existente também foram verificadas. A execução completa dos testes do workspace deve ser confirmada após essa alteração. Próximo: item 2, parada normal com espera e confirmação antes de forçar.

Validação em 2026-11-07: `cargo test --workspace` passou no Windows, incluindo launcher curto, relançamento por pasta e falha de associação. O Job Object está no commit `a2ab3f9`. Limite: os testes automatizados não comprovam o comportamento de todos os launchers reais nem processos que escapem do grupo fora da pasta do jogo. A parada ainda é forçada; o item 2 permanece aberto.

Revisão inicial dos cenários de sync: `sync.rs` cobre as decisões de três vias e agora uma falha após copiar um arquivo; `scenarios.rs` cobre arquivo travado, nuvem parcial, caminho longo, arquivos apagados, save vazio e restauração em outro PC; `engine.rs` cobre trava e saída com erro; `processo_relancado.rs` cobre launcher curto. Ainda falta verificar cobertura de cada condição e testar disco cheio na nuvem e no local: erros genéricos de destino não demonstram ENOSPC. A revisão dos oito itens continua pendente.

Implementação em andamento da parada: núcleo passou a pedir WM_CLOSE às janelas do grupo e exige 10 s antes de aceitar força; o loop continua acompanhando Job Object e processos da pasta até o intervalo de silêncio. Serviço, comando Tauri e diálogo foram ligados. `cargo check --workspace`, `bun run build` e teste de processo sem janela passaram. Falta validar janela real que aceita/ignora WM_CLOSE, sincronização após todos os filhos, cancelamento, E2E e capturas; revisar concorrência e casos de erro antes de considerar seguro ou commitar.

Continuação: o prazo de 10 s agora começa após o envio do pedido; falhas de acompanhamento não disparam sincronização nem soltam a trava. Teste Windows com janela real passou e confirmou espera por um processo filho iniciado no fechamento; teste de processo sem janela confirmou a espera e o bloqueio da força antecipada. Testes do serviço passaram. E2E de cancelar e confirmar força passou isolado; a suíte completa precisa ser repetida em porta livre (havia um Vite antigo na porta 1420). Capturas mobile e desktop revelaram foco recortado; foi aplicada correção que ainda precisa ser reinspecionada. Permanecem testes de janela que ignora WM_CLOSE, sync após todo o grupo, execução completa e revisão do P0.

Revisão de segurança: teste de serviço com launcher que sai e filho que grava o save revelou que dois snapshots da mesma sessão reutilizavam o mesmo ID e sobrescreviam o primeiro. A reserva atômica de diretório e sufixo para colisões foram implementados; o teste do serviço agora passa. O formato e a exibição/ordenação desses IDs precisam ser revisados e documentados antes do commit. A verificação do P0 6 também identificou que `Ctx::after_exit` ainda calcula o índice imediatamente: faltam a espera de 3 s e dois hashes iguais separados por 1 s, além de testes correspondentes. A revisão dos demais itens e a execução completa continuam abertas.

Estabilização implementada no núcleo: espera inicial de 3 s, duas leituras de índice com 1 s de intervalo, nova comparação antes de copiar e erro após 60 leituras sem estabilidade; a trava recebe heartbeat durante a espera. Serviço e CLI preservam a trava quando o acompanhamento falha. O teste de save alterado durante a espera passou na execução anterior do núcleo; a comparação adicional precisa ser testada junto do workspace atual. A captura atualizada da confirmação não mostrou recorte de foco nas larguras 390 e 1280. ADR 0007 documenta o sufixo de IDs e a reserva atômica; revisão completa do P0 e cobertura de disco cheio ainda pendentes.

### Auditoria inicial dos oito itens P0 (2026-11-07)

1. Pasta de sync: seletor e validação em `Configuracoes`/`check_sync_dir`; E2E de primeiro uso e testes de serviço. Confirmar por E2E o acesso às configurações ao tentar jogar sem pasta.
2. Adição: reconhecimento por catálogo e seleção manual de pasta têm testes de catálogo e E2E; faltam seleção manual de arquivos individuais e aviso de save no registro. A detecção por mudança descrita nas decisões do P0 também falta (item 3 acima). Verificar exibição de tamanho dos candidatos.
3. Persistência e edição/remoção: testes `sgs-library/tests/library.rs`, `sgs-service/tests/service.rs` e E2E de edição/remoção; confirmar persistência do último sync e base após reinício.
4. Jogar/parar: Job Object e testes Windows; fechamento normal/forçado e diálogo em andamento. Rever comportamento em falhas de rastreamento e concorrência; rodar suítes após as últimas mudanças.
5. Abertura: `engine.rs`/`sync.rs` cobrem três vias e preservação do lado descartado; E2E cobre conflito e trava. O diálogo de conflito ainda não mostra data e tamanho de cada lado. Confirmar E2E de nuvem indisponível e preservação de backups em falhas.
6. Fechamento: testes de saída com erro, launcher/filho, estabilidade e snapshot; falta validar integralmente os novos caminhos e casos de erro, além de prevenir concorrência entre operações em processos distintos.
7. Interface: E2E de teclado, diálogo, biblioteca e Sobre, testes unitários de controle; falta verificação com controle físico e conferir build real na tela Sobre.
8. Segurança: versões/restauração e trava têm testes; disco cheio foi simulado por erro do sistema injetado nas cópias (não por volume realmente cheio). Não foi encontrado log diário de sync nem botão para abrir sua pasta. Há configuração de NSIS, mas falta evidência de instalador gerado e testado para esta entrega.

Cenários de sync: três vias (`sync.rs`), interrupção após cópia e disco cheio injetado (`snapshot.rs`), arquivo travado e nuvem parcial (`scenarios.rs`), trava (`lock_paths.rs`), caminhos longos e diferenças de caixa (`scenarios.rs`), acentos/espaços (`sync.rs`), pastas/vazio/remoções e outro perfil (`scenarios.rs`), saída com erro (`engine.rs`) e launcher (`processo_relancado.rs`). Falta validar os erros de disco em volume realmente cheio e confirmar que a restauração parcial preserva backup acessível ao usuário em todos os chamadores.

Novos achados de recuperação: `restore_roots` agora recusa reutilizar pasta de backup; serviço impede restauração durante jogo/sync e valida o ID da versão. Ainda é preciso testar concorrência entre instâncias do app/CLI, o caso de operação interrompida com trava própria e a comunicação do caminho de backup quando uma restauração falha após gravá-lo.

Validação adicional: a suíte Rust completa terminou com sucesso antes das últimas mudanças em restauração. Os testes direcionados de snapshot, backup e bloqueio de restauração durante o jogo passaram; é necessário repetir os testes afetados após a mensagem de erro adicionada ao serviço e, antes de commit na `main`, repetir o workspace completo. A simulação de disco cheio usa códigos reais de erro injetados, não um volume físico cheio.
