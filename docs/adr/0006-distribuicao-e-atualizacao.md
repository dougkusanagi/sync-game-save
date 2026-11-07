# 0006. Distribuição NSIS e AppImage com atualização assinada

- Status: Aceito
- Data: 2026-11-06

## Contexto

O plano pede instalador para Windows (NSIS), AppImage para Linux e atualização automática. O app precisa de um canal de atualização que não dependa de um servidor próprio e que impeça a instalação de um pacote adulterado. O repositório já fica no GitHub (`dougkusanagi/sync-game-save`).

## Decisão

- Pacotes: NSIS no Windows, em português do Brasil e instalação por usuário (`currentUser`); AppImage no Linux. O comando é `bun run tauri build` com `bundle.targets` limitado a esses dois formatos.
- Canal de atualização: o plugin `tauri-plugin-updater` lê `https://github.com/dougkusanagi/sync-game-save/releases/latest/download/latest.json`, gerado junto com cada release do GitHub.
- Assinatura: os pacotes de atualização usam uma chave minisign. A chave pública fica em `src-tauri/tauri.conf.json`; a privada fica fora do repositório, em `%USERPROFILE%\.tauri\sync-game-save.key`, informada no build por `TAURI_SIGNING_PRIVATE_KEY`.
- Uso: o app verifica atualizações ao abrir, sem bloquear a interface, e mostra um botão no topo quando há uma versão nova. A instalação acontece em Configurações → Sobre, com download, instalação e reinício explícitos.
- O número do build mostrado em Sobre vem do `build.rs` (atalho do commit e data), não do pacote.

## Alternativas

- Instalar a atualização sem confirmação: reduz cliques, mas reinicia o app enquanto o usuário pode estar configurando um jogo ou vendo um conflito de save.
- Servidor próprio para o manifesto de atualização: dá controle sobre a distribuição, mas adiciona hospedagem e operação sem necessidade nesta fase.
- Atualização sem assinatura: mais simples de publicar, porém permitiria instalar um pacote trocado por terceiros.

## Consequências

- Publicar uma versão exige gerar os pacotes, assinar, enviar os artefatos ao GitHub Releases e publicar o `latest.json` correspondente. Sem a chave privada, o build gera o instalador, mas não um pacote de atualização válido.
- Quem usa uma versão antiga só recebe a atualização se a nova versão estiver em um release publicado e assinado com a mesma chave.
- O AppImage precisa ser gerado em Linux; o NSIS é gerado no Windows. Os dois formatos saem do mesmo código e configuração.
