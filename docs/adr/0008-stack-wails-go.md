# 0008. Stack do app: Wails v3 + Go + React

- Status: Proposto (branch `feat/migracao-mygo`; passa a Aceito e substitui a ADR 0001 quando a migração entrar no `main`)
- Data: 2026-11-14

## Contexto

O app usa Tauri 2 com o backend em Rust (ADR 0001). Os arquivos de build do Rust chegaram a 31,87 GiB no checkout principal, e o build de release (LTO completo, `codegen-units = 1`) leva cerca de 32 minutos. O usuário decidiu trocar o backend por Go para reduzir espaço e tempo de compilação, mantendo o frontend React.

Medição de uma prova de conceito igual nos dois frameworks Go (janela com o React atual, binding, evento, bandeja, fechar para a bandeja, capas locais), uma rodada no Windows, em 2026-11-14:

| | Tauri (Rust) | MyGo 0.9.0 | Wails 3.1.0 |
|---|---|---|---|
| Espaço do build | 4,8–7,8 GB | 262 MB | 134 MB |
| Release | ~32 min | ~2,5 min | ~3 min (build completo) |
| Recompilar após 1 linha (só compilador) | 13 s | 22 s | 21 s |
| Instalador NSIS | não medido | 3,5 MB | 6,1 MB |

## Decisão

Reescrever o backend em Go com o Wails v3 (v3.1.0). O frontend React + TypeScript + Vite continua, com bun. O plano está em `docs/plans/migracao-wails.md`.

## Alternativas

- Manter Tauri e ajustar os perfis de build (menos símbolos de depuração, LTO só na distribuição, limpeza do `target`): menor esforço, mas o usuário preferiu a troca.
- MyGo: mesmo patamar de espaço e tempo, Go puro sem cgo e NSIS baixado pelo CLI. Recusado por ser pré-1.0 (quebras de API na 0.8 e na 0.9), não gerar AppImage e não ter mobile.
- Wails v2: estável e mais antigo, mas sem mobile e com menos recursos de bandeja e várias janelas que a v3.
- Manter os crates Rust e trocar só a camada do app: não resolve o espaço nem o tempo de compilação.

## Consequências

- Build bem menor e release muito mais rápido; a recompilação do dia a dia fica um pouco mais lenta que no Rust (a confirmar com o núcleo portado).
- Todo o backend (~5.200 linhas de Rust e seus testes) precisa ser reescrito em Go, com compatibilidade byte a byte dos dados do usuário (ADR 0003, 0007).
- O atualizador muda de formato; os usuários da v1.x precisam de uma versão-ponte do Tauri.
- O instalador NSIS precisa do `makensis` instalado na máquina de build.
- Android e iOS no Wails são experimentais; o Tauri tinha mobile estável, que o projeto não usava.
- O nome do branch (`feat/migracao-mygo`) ficou da primeira escolha.
