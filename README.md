# sync-game-save

Launcher/HUB de jogos para PC que sincroniza o save com uma pasta de nuvem (OneDrive, Google Drive etc.) ao abrir e ao fechar cada jogo, sem perder saves.

![Início do Sync Game Save, com a arte do jogo em destaque, os botões Jogar, Versões e Conquistas e a faixa de jogos jogados recentemente](.github/preview.png)

## Estado

Em desenvolvimento com Tauri 2 + React (ver [plano](docs/plans/launcher-hub.md) e [ADR 0001](docs/adr/0001-stack-tauri-react.md)). Estão prontos o núcleo de sync em Rust (`sgs-core`), o catálogo (`sgs-catalog`), a biblioteca local (`sgs-library`), o serviço (`sgs-service`), a CLI `sgs` e as telas: primeiro uso, Início (arte do jogo em destaque e faixa de jogos por seção), Biblioteca em grade de capas, adicionar jogo, conquistas do jogo e do HUB, versões, conflito, configurações e Sobre. Os sons da interface são sintetizados em `src/som.ts`.

As artes de cada jogo (capa, fundo, logo e capa vertical) vêm da Steam CDN. Quando o catálogo não conhece o jogo, o app procura o nome na loja da Steam. Sem arte, o app mostra as iniciais do jogo sobre uma cor. As conquistas da Steam usam a conta da Steam instalada neste PC e o perfil público, sem chave de API; o perfil precisa mostrar os detalhes dos jogos como públicos. Os temas trocam as cores e medidas da interface; o app traz o escuro (padrão) e o claro. Falta o assistente de varredura de pastas.

Na tela de adicionar jogo, quando o catálogo não indica onde o jogo salva, o botão “Descobrir onde o jogo salva” abre o jogo, espera ele fechar e lista as pastas com arquivos novos ou alterados, ignorando caches e logs. O caminho só entra no jogo após a sua confirmação; nada é salvo por conta própria.

No Windows, ao parar um jogo o app pede que ele feche normalmente. Se continuar aberto após 10 segundos, é possível confirmar o fechamento forçado, que pode danificar o save. O sync espera o grupo de processos terminar e verifica se os arquivos pararam de mudar antes de criar uma nova versão.

## Conquistas do HUB

Em **Meu progresso**, o app mostra marcos válidos para qualquer jogo: horas jogadas, jogos na biblioteca, sequência de sincronizações concluídas sem erro e jogos com todas as conquistas da Steam desbloqueadas. O progresso é calculado da biblioteca local e do cache de conquistas; a sequência atual zera quando uma sincronização falha, e o melhor resultado fica guardado.

## Capas

As capas são baixadas uma vez e ficam no cache local (`art` na pasta de dados do app). A Steam é a primeira fonte. Para os demais jogos, o SteamGridDB é consultado pelo nome quando há uma chave configurada em Configurações → Capas dos jogos. A chave fica só neste PC, no banco local, e nunca volta para a tela.

## Temas

Para instalar um tema, crie `themes/<id>/theme.json` na pasta de dados do app (`%APPDATA%\com.syncgamesave.app\themes` no Windows; `SGS_DATA_DIR` troca a pasta) e escolha o tema em Configurações → Tema:

```json
{
  "name": "Meu tema",
  "author": "Seu nome",
  "version": "1.0.0",
  "min_app_version": "0.1.0",
  "tokens": {
    "--sgs-bg": "#101820",
    "--sgs-text": "#f5f7fa"
  }
}
```

Só os tokens `--sgs-*` (listados em `src/tokens.css`) são aplicados. Nenhum CSS ou JavaScript do tema é executado, e valores com `url()`, `@` ou `;` são recusados. Temas com `min_app_version` maior que a versão do app são ignorados.

## Distribuição

O app é distribuído em [releases no GitHub](https://github.com/dougkusanagi/sync-game-save/releases): instalador NSIS para Windows (português do Brasil) e AppImage para Linux. Ao abrir, o app procura uma versão nova e mostra o aviso; a instalação fica em Configurações → Sobre.

```bash
bun run tauri build --bundles nsis   # Windows: src-tauri/target/release/bundle/nsis/
bun run tauri build --bundles appimage   # Linux
```

O build gera os pacotes de atualização assinados quando `TAURI_SIGNING_PRIVATE_KEY` aponta para a chave privada (fora do repositório; a chave pública está em `src-tauri/tauri.conf.json`). O canal de atualização lê o `latest.json` do último release.

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

`SGS_DATA_DIR` troca a pasta de dados do app (útil para testes sem mexer na sua biblioteca). Na primeira vez, instale o navegador dos testes E2E com `bunx playwright install chromium`. A interface também abre no navegador (`bun run dev`, `http://localhost:1420/?cenario=jogos`) com dados simulados; os cenários são `jogos`, `vazio`, `primeiro-uso`, `conflito` e `preview` (com arte, usado na miniatura do README).

## CLI `sgs`

```bash
cd src-tauri
cargo run -p sgs-cli -- help
cargo run -p sgs-cli -- run --sync-dir <pasta da nuvem> --game "Nome" --save-dir <pasta do save> -- <jogo.exe>
```

O `run` sincroniza antes de abrir, abre o programa, espera ele fechar e envia o save. Uma queda ou um erro do jogo não perde o save: o formato da pasta da nuvem está no [ADR 0003](docs/adr/0003-formato-da-pasta-da-nuvem.md).
