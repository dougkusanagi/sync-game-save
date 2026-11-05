# Protótipo da interface

Protótipo navegável do HUB (HTML + CSS + JS puro, servidor Vite+). Serve para validar visual, navegação por controle e teclado e os fluxos de sync antes de portar para Tauri + React. Os dados e os fluxos são simulados.

## Como rodar

```bash
cd prototype
bun install
bun run dev
```

Abra `http://localhost:5199`. Use teclado, mouse ou um controle (Xbox ou PlayStation) conectado.

## O que testar

- Início: navegue pela faixa de jogos; o fundo e o logo mudam com o foco. `Q`/`E` (LB/RB) trocam a seção.
- Jogar: Hollow Knight tem conflito de save; Celeste baixa da nuvem; Hades envia antes de abrir; Sea of Stars espera a nuvem terminar de baixar.
- Biblioteca (`L`), filtros (`F`), busca (`/`), menu (`M`), central rápida (`F1`), atalhos (`?`), adicionar jogo (`N`), configurações (`Ctrl + ,`).
- Central rápida → "Ferramentas do protótipo": simular nuvem indisponível e biblioteca vazia.

As artes vêm da CDN pública da Steam só para dar realismo ao protótipo.
