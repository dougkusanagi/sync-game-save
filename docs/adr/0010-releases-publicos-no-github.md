# 0010. Repositório e releases públicos no GitHub

- Status: Aceito
- Data: 2026-11-15

## Contexto

O atualizador do app (ADR 0006 para o Tauri e ADR 0009 para o Wails) lê um manifesto e baixa o instalador direto do GitHub, sem servidor próprio e sem login: `https://github.com/dougkusanagi/sync-game-save/releases/latest/download/<arquivo>`. Com o repositório privado, essa URL responde 404 a qualquer cliente sem credenciais, então nenhuma instalação conseguia se atualizar sozinha. Isso apareceu ao publicar a versão 2.0.0, quando o `curl` nos manifestos devolveu 404.

## Decisão

Tornar o repositório `dougkusanagi/sync-game-save` público. Os releases, os manifestos (`latest-wails.json` para o app Go e `latest.json` para as instalações da v1.x) e os instaladores ficam acessíveis sem login. Antes de publicar, o histórico completo (287 commits) foi procurado por chaves privadas e tokens; não havia nenhum. A chave pública do atualizador fica em `build/updater-public.pem`, e as privadas ficam fora do repositório (`MEMORY.md`).

## Alternativas

- Manter o repositório privado e hospedar só os manifestos e os arquivos em outro lugar público (um repositório só de releases, um bucket ou um domínio próprio): preserva o código fechado, mas cria outro serviço para manter e outro lugar a publicar a cada release.
- Autenticar o atualizador com um token de leitura embutido no app: qualquer pessoa extrairia o token do executável.

## Consequências

- O código-fonte, o histórico e o plano de trabalho passam a ser públicos. O manifesto do Ludusavi (dados de terceiros sob licença MIT) já está no histórico e continua a ser usado.
- O atualizador alcança o release sem configuração extra. Foi conferido de ponta a ponta com o provedor real do Wails (consulta, download, resumo e assinatura Ed25519) e com um Tauri 1.1.0 instalado de verdade, que viu a 2.0.1, baixou o instalador, aceitou a assinatura minisign e atualizou em silêncio.
- Voltar a tornar o repositório privado quebraria o atualizador de todas as instalações existentes; é preciso migrar os manifestos antes.
