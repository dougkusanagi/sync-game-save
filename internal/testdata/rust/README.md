# Dados gerados pela versão Rust

`library.db`, `catalog.db` e `captura-<chave>.json` foram gerados pela versão Rust (Tauri) do app e servem de fixture para os testes de compatibilidade do porte em Go (`internal/library`, `internal/catalog`, `internal/core`). A fixture do formato da nuvem fica em `internal/core/testdata/rust`.

Para regerar, recupere o exemplo do histórico do git (ele saiu do código com a remoção do Tauri):

```bash
git show ce3ca1d:src-tauri/crates/sgs-service/examples/gerar_fixtures.rs
```

Coloque o arquivo em `src-tauri/crates/sgs-service/examples/` de um checkout do `main` (que ainda tem o código Rust) e rode `cargo run -p sgs-service --example gerar_fixtures -- <pasta de saída>`.

Regerar muda os bytes dos bancos; confira os testes depois. Não edite os arquivos à mão.
