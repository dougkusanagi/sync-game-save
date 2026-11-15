# 0009. Atualizador do app Go: manifesto assinado com Ed25519

- Status: Proposto (branch `feat/migracao-mygo`; passa a Aceito junto com a ADR 0008)
- Data: 2026-11-14

## Contexto

O atualizador do Tauri (ADR 0006) usa `tauri-plugin-updater`, assinatura minisign e um `latest.json` no último release do GitHub. Com a troca para o Wails v3 (ADR 0008), esse plugin deixa de existir. O Wails v3.1 traz um atualizador próprio (`app.Updater`) com provedores de GitHub Releases, de manifesto por URL (`endpoint`), de appcast e de Keygen. Ele confere o SHA-256 do arquivo e, se houver chave pública configurada, torna obrigatória uma assinatura Ed25519 válida, recusando o arquivo se faltar. O provedor do GitHub só confere o SHA-256 de um arquivo de somas publicado no mesmo release: não dá autenticidade se o repositório for comprometido.

## Decisão

- Usar o provedor `endpoint` com um manifesto estático `latest-wails.json`, publicado como arquivo do último release: `https://github.com/dougkusanagi/sync-game-save/releases/latest/download/latest-wails.json`. O nome difere do `latest.json` do Tauri para as duas gerações conviverem durante a transição.
- O artefato de atualização é o executável do app (`sync-game-save.exe`), que o atualizador troca no lugar. O instalador NSIS continua só para a primeira instalação.
- Assinatura: Ed25519 sobre o SHA-256 do executável, gravada no manifesto. A chave pública fica em `build/updater-public.pem` e entra no binário (`go:embed`). A privada fica fora do repositório, em `%USERPROFILE%\.sgs\updater-ed25519.pem` (PKCS#8, sem senha). Sem a chave pública embutida, o atualizador fica desligado.
- O manifesto e o par de chaves saem do comando `cmd/assinar` (`assinar chaves` e `assinar manifesto`), que se recusa a sobrescrever uma chave existente.
- A janela do atualizador fica desligada (`WindowNone`): a interface do app mostra o andamento. A instalação recusa jogo aberto ou em sincronização e só reinicia depois de baixar e conferir o arquivo.

## Alternativas

- Provedor do GitHub com arquivo de somas: mais simples de publicar, mas sem assinatura, então quem controla o release controla o que é instalado.
- Reaproveitar a chave minisign do Tauri: o atualizador do Wails não verifica minisign, e misturar as chaves ligaria as duas gerações.
- Substituir o app pelo instalador NSIS a cada atualização: o Wails 3.1 troca o executável por padrão, e o instalador exigiria elevar permissões e fechar o app por outro caminho.

## Consequências

- Cada release precisa gerar o executável, rodar `assinar manifesto` com a chave privada e publicar o `latest-wails.json` junto com o executável. Sem a chave privada não há atualização válida.
- Perder a chave privada impede atualizar instalações existentes pelo app: seria preciso uma nova instalação com uma nova chave pública.
- Quem usa a v1.x (Tauri) não recebe esse manifesto. A passagem depende de uma última versão do Tauri cujo `latest.json`, assinado com a chave minisign atual, entregue o instalador da versão Go. Essa ponte ainda não foi construída nem testada.
- A troca do executável em uso e o reinício dependem do auxiliar do Wails; só foi testada a verificação (manifesto, resumo e assinatura), não a troca real num Windows com a v1.x instalada.
