# 0004. Várias pastas de save por jogo, com prefixo lógico nos snapshots

- Status: Aceito
- Data: 2026-11-05

## Contexto

O [ADR 0003](0003-formato-da-pasta-da-nuvem.md) definiu o snapshot com `files/<caminho>` relativo a uma única pasta de save. Jogos reais guardam progresso em mais de um lugar (por exemplo Documentos e AppData). O catálogo do Ludusavi lista vários caminhos por jogo. O 0003 também dizia que a ligação com as pastas reais de cada PC ficaria na biblioteca; isso mudou aqui.

## Decisão

- Cada jogo tem uma lista de pastas de save, guardadas como caminhos lógicos com marcadores (`<winDocuments>/SavedGames/Jogo`).
- Dentro do snapshot, os arquivos ficam em `files/<caminho lógico sem < > :>/<arquivo>`, por exemplo `files/winDocuments/SavedGames/Jogo/slot1.sav`. O `index.json` usa as mesmas chaves.
- Ao restaurar, cada chave volta para a pasta real do marcador naquele PC (outro usuário, outra pasta de Documentos). Documentos redirecionado para o OneDrive é resolvido pelo Windows, não por `<home>/Documents`.
- Jogo com uma só pasta sem rótulo (usado pela CLI com `--save-dir` comum) mantém as chaves sem prefixo, como no 0003.
- Só pastas são suportadas como raiz; arquivos soltos de save ainda não.

## Alternativas

- Um snapshot por pasta: duplica `head.json`, trava e conflito por pasta, e permite estados que nunca existiram juntos.
- Guardar a ligação marcador → pasta só na biblioteca: o snapshot não seria restaurável por quem lê a pasta da nuvem sem o app.

## Consequências

- Snapshots feitos com o formato do 0003 e rótulo vazio continuam válidos.
- Mudar a lista de pastas de um jogo muda as chaves: arquivos de uma pasta removida da lista aparecem como apagados na comparação seguinte e são restaurados apenas com backup antes.
- Duas pastas com o mesmo caminho lógico depois de tirar `< > :` colidiriam; o catálogo não tem esse caso.
