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

## Estado atual (2026-11-05)

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

## Temas (futuro)

- Toda cor, raio, fonte, sombra, blur e duração de animação vem de **tokens CSS** (`--sgs-*`) desde o protótipo. Isso torna os temas possíveis sem refazer a interface.
- Pacote de tema: `theme.json` (nome, autor, versão, versão mínima do app, tokens) + `theme.css` opcional + `assets/` (fundos, sons, fontes).
- Segurança: temas não executam JS; o CSS é carregado com CSP restrita e sem acesso a URLs externas.
- Tema padrão e um tema claro oficiais servem de exemplo e de teste.

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
- [x] Caminhos lógicos: resolver/converter marcadores (`paths`). Falta: preencher `Env` com as pastas reais do Windows/Linux.
- [x] Índice de arquivos com SHA-256; comparação em três vias (`sgs-core`: `index`, `sync`).
- [x] Snapshots imutáveis, commit por `head.json`, verificação de hash e restauração com backup. Falta: retenção (`snapshot::prune`).
- [x] Backups locais (`engine`) e journal de operações (`journal`).
- [x] `lock.json` com heartbeat (`lock`).
- [x] CLI `sgs run|backup|restore|status|list` com saída JSON opcional (`sgs-cli`).
- [x] Cenários de teste do sync passando com `cargo test`, inclusive jogo aberto por launcher intermediário (a CLI acompanha o programa e os processos descendentes; `--quiet-secs`, padrão 5). O Job Object do Windows e a busca por pasta de instalação ficam para a Fase 2.
- [ ] Uso real por 1–2 semanas pela CLI com atalhos (Steam, Epic ou outro), nos jogos atuais. Primeira sessão real feita em 2026-11-05 com o Rogue Legacy (Epic, sem save na nuvem). **Em andamento, em paralelo à Fase 2** (decisão do usuário em 2026-11-05).

### Fase 2 — App e biblioteca

- [x] SQLite local (`library.db`) com migrações (`sgs-library`: jogos, caminhos de save, estado do sync, configurações).
- [ ] Assistente de primeiro uso (pasta da nuvem) bloqueando o uso até ser concluído.
- [x] Reconhecimento pela pasta do jogo e varredura de pastas (`sgs-catalog::recognize`; testado com o Rogue Legacy real). Falta: a tela do assistente de adicionar jogo (Fase 3) e a detecção de save por mudança.
- [ ] Abrir jogo com rastreio de processos (Job Object) e tempo de jogo.
- [ ] Integração do fluxo de abrir/fechar com o `sgs-core` e eventos para a interface.
- [ ] Arte (Steam CDN + SteamGridDB) com cache.
- [ ] Bandeja e notificações.

### Fase 3 — Interface final

- [ ] Portar o protótipo para componentes React, mantendo os tokens e o motor de foco.
- [ ] Controle lido no Rust (`gilrs`) + motor de foco.
- [ ] Telas de histórico, conflito, configurações, primeiro uso e "Sobre" (nome, versão, build).
- [ ] Testes E2E dos fluxos principais (abrir jogo, conflito, restaurar versão) e capturas em várias resoluções.
- [ ] `.github/preview.png` e README atualizados.

### Fase 4 — Som, conquistas e temas (P1/P2)

- [ ] Fundo ambiente e efeitos sonoros próprios (seção "Som"), com volume separado e teste de 1 hora.

- [ ] Spike de fontes de conquistas; implementar `AchievementProvider` com Steam Web API e conquistas do HUB.
- [ ] Formato de pacote de tema, carregamento seguro e tema claro de exemplo.

### Fase 5 — Distribuição

- [ ] Versão `0.1.0` → 1.0 conforme SemVer; changelog por fragmentos.
- [ ] Instalador NSIS (Windows) e AppImage (Linux), atualização automática.

## Tentativas que falharam

- Copiar só quando o código de saída é 0 (`GameService.ts`): perde o save quando o jogo fecha com erro ou quando o `.exe` é um launcher intermediário.

## Dúvidas abertas

- ADR 0002 (catálogo em SQLite FTS5) está de acordo?
- Usar a chave da SteamGridDB do usuário, ou só a arte da Steam e imagens próprias?
- Steam Deck (SteamOS) entra junto com o Ubuntu ou depois?
- Seletor de tema claro/escuro: o protótipo é só escuro (a arte do jogo é o fundo). O plano deixa o tema claro para a fase de temas. Confirmar.

## Próximo passo

Fase 1 concluída no código (branch `feat/fase1-nucleo-sync`): `sgs-core`, `sgs-catalog`, `sgs-cli` e o projeto Tauri + React compilam e os testes passam com `cargo test --workspace`. Faltam só duas coisas que dependem do usuário: o uso real por 1–2 semanas pela CLI. Próximo: Fase 2 (SQLite `library.db`, assistente de primeiro uso, adicionar jogo, rastreio de processos). Ainda vale o texto abaixo para a revisão do protótipo.

Revisar o protótipo (`cd prototype && bun install && bun run dev`) com controle e teclado, anotar ajustes e confirmar o ADR 0002. Depois, iniciar a Fase 1 removendo o código Electron e criando o projeto Tauri + React.
