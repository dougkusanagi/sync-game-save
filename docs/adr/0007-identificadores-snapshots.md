# 0007. Identificadores únicos para snapshots

- Status: Aceito
- Data: 2026-11-07

## Contexto

O mesmo jogo pode criar um snapshot ao abrir e outro ao fechar. Ambos usavam o instante capturado no início da sessão e, portanto, o mesmo nome. Isso permitia sobrescrever o primeiro snapshot, contrariando a imutabilidade prevista na ADR 0003.

## Decisão

Preservar o formato e as demais regras da ADR 0003. Quando o identificador `<segundos>_<PC>` já existe, acrescentar `~sgs-<sequência>` ao nome; reservar a pasta com criação atômica e recusar uma colisão concorrente. O `head.json` continua apontando o nome completo. A lista mostra primeiro o head e omite o sufixo técnico na apresentação do PC.

## Alternativas

- Sobrescrever a pasta: descartado porque destrói uma versão anterior.
- Usar apenas relógio com maior precisão: não garante unicidade entre chamadas concorrentes ou relógios iguais.

## Consequências

Snapshots existentes continuam legíveis e nenhum precisa ser migrado. Versões antigas do app não reconhecem o sufixo na apresentação e podem mostrar o nome do PC com o sufixo. O marcador final `~sgs-` seguido de dígitos fica reservado para a sequência; um nome de PC antigo que termine assim pode ser apresentado sem essa parte. Uma colisão concorrente falha sem alterar o snapshot existente e exige nova tentativa.
