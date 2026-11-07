# 0002. Catálogo de jogos: manifesto do Ludusavi indexado em SQLite FTS5

- Status: Aceito
- Data: 2026-11-05

## Contexto

O app precisa saber, para milhares de jogos, o nome, as pastas de instalação e onde cada um guarda o save. A busca precisa responder enquanto o usuário digita, sem travar a interface. A versão Electron guardava o manifesto do Ludusavi (YAML, ~30 MB) no repositório e o lia inteiro em memória.

## Decisão

- Fonte principal: manifesto do Ludusavi (licença MIT), baixado em tempo de execução com `ETag`, nunca versionado. Regras personalizadas do usuário ficam em `library.db` e têm prioridade.
- Persistência: o Rust converte o manifesto num arquivo `catalog.db` (SQLite) separado da biblioteca do usuário. O arquivo é montado numa única transação num arquivo temporário e trocado de forma atômica.
- Busca: tabela virtual FTS5 com `tokenize='trigram'` sobre nome, aliases e pastas; `LIKE 'ab%'` no nome normalizado para buscas com menos de 3 caracteres. Índice B-tree em `install_dirs` para identificar um jogo pela pasta do executável.
- As consultas rodam no Rust fora da thread da interface; a interface usa debounce e descarta respostas antigas.

## Alternativas

- YAML em memória: lento para abrir e alto uso de RAM.
- Índice em memória com busca difusa: rápido, mas gasta RAM e precisa ser remontado a cada início.
- Tantivy: motor de busca completo, maior que o necessário para o volume.
- Servidor de busca externo: contraria um app local e leve.

## Consequências

- Busca rápida e memória baixa; o catálogo pode ser apagado e remontado sem perder dados do usuário.
- O manifesto não traz descrição, desenvolvedora nem ano: esses dados vêm de outras fontes (Steam Store, IGDB opcional) e ficam em cache em `library.db`.
- Medido em 2026-11-05 com o manifesto real (45.612 jogos, 352.819 regras de arquivo, 29 MB de YAML), em build release: montagem em 2,3 s, `catalog.db` com 35 MB, busca em cerca de 1 ms (4 ms para prefixo de 2 letras) e busca por pasta em 0,2 ms.
- O manifesto real traz caracteres de controle (ex.: DEL) que o parser YAML recusa; a montagem os troca por espaço antes de ler.
- A tela "Sobre" precisa citar o Ludusavi e sua licença.
