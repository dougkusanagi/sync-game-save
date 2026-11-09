# 0003. Formato da pasta da nuvem: snapshots imutáveis e `head.json`

- Status: Substituído por 0007
- Data: 2026-11-05

## Contexto

Os saves ficam numa pasta sincronizada por OneDrive, Google Drive, Dropbox ou Syncthing. Esses serviços copiam arquivos de forma assíncrona e podem entregar uma pasta pela metade. Dois PCs usam a mesma pasta, e o usuário precisa recuperar um save só com o Explorer. O formato fica nos arquivos do usuário e outros PCs dependem dele; mudar depois é difícil.

## Decisão

Layout em `<pasta de sync>/SyncGameSave/games/<nome do jogo>/`:

- `snapshots/<segundos desde 1970>_<PC>/index.json` e `files/<caminho lógico>/...`: arquivos do save em texto claro e um índice com tamanho e SHA-256 de cada um.
- `head.json`: aponta o snapshot atual (`snapshot`, `index_digest`, `device`). É gravado por último, em arquivo temporário seguido de renomeação.
- `lock.json`: PC, início e heartbeat. Heartbeat com menos de 10 minutos de um PC diferente gera aviso.

Regras:

- Um snapshot nunca é alterado depois de escrito. Um snapshot sem `index.json` é incompleto e ignorado.
- Antes de usar um snapshot, os hashes dos arquivos são conferidos. Se não batem, a nuvem ainda está sincronizando: nada local é alterado.
- A decisão ao abrir compara três estados: base (último estado sincronizado neste PC, guardado fora da nuvem), local e `head`. Se os dois lados mudaram, é conflito e o usuário escolhe; o lado descartado fica em snapshot ou em backup local.
- Antes de sobrescrever o save local, ele é copiado para `%LOCALAPPDATA%/SyncGameSave/.../backups/`. Se o backup falha, nada é sobrescrito.
- O banco da biblioteca e a base ficam fora da nuvem, para o sincronizador não corromper o SQLite.
- Retenção: o mínimo de versões a manter é 1, e o snapshot do `head` nunca é apagado. O MVP não apaga nada sozinho.

## Alternativas

- Copiar sobre a pasta de destino (como o app Electron antigo): uma queda no meio deixa save misturado e perde o estado anterior.
- Um arquivo compactado por versão: menor, mas o usuário não recupera um arquivo só pelo Explorer.
- Banco de dados na nuvem: arrisca corrupção com sincronizadores de arquivo.

## Consequências

- Usa mais espaço (uma cópia completa por versão), por isso existe a retenção.
- Dois PCs gravando ao mesmo tempo podem gerar duas versões de `head.json` do sincronizador (cópia de conflito do serviço); a trava reduz o risco, e a detecção dessas cópias ainda está pendente.
- Os caminhos dos arquivos ficam relativos à pasta de save do jogo; a ligação com pastas reais de cada PC (caminhos lógicos com marcadores) é guardada à parte na biblioteca.

