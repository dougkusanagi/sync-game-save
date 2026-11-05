# 0001. Stack do app: Tauri 2 + React + TypeScript

- Status: Aceito
- Data: 2026-11-05

## Contexto

O app é um launcher/HUB que fica aberto (na bandeja) enquanto o jogo roda e copia saves entre o PC e uma pasta de nuvem. A primeira versão usava Electron + Vue (electron-vite), com cerca de 300 linhas úteis, sem restauração de save e sem testes. O usuário quer algo mais leve que o Electron, interface animada em HTML/CSS, controle por gamepad e suporte a Windows e Ubuntu.

## Decisão

Reescrever o app com Tauri 2: núcleo em Rust (sync, catálogo, processos, controle, bandeja) e interface em React + TypeScript com Vite, usando bun. O código Electron é descartado.

## Alternativas

- Manter Electron: termina mais rápido, mas gasta mais memória (Chromium próprio) e gera instaladores muito maiores, o que pesa num app que fica aberto durante o jogo.
- Tauri + Vue: o usuário prefere React porque as ferramentas de IA geram e revisam React com mais qualidade.
- Tauri com sidecar Node: mantém TypeScript no backend, mas soma um runtime ao instalador e perde o ganho de leveza.
- Usar o Ludusavi (CLI) como motor: já faz backup e restauração, mas limita o fluxo seguro (snapshots, conflitos, travas) e a integração com a interface.
- App nativo (WinUI, Qt, Flutter): mais trabalho para a interface animada e para temas em CSS.

## Consequências

- Menos memória e instaladores menores; leitura do controle no Rust funciona com a janela sem foco.
- A parte crítica (sync) fica numa biblioteca Rust testável sem interface (`sgs-core`).
- O time precisa trabalhar com Rust no backend.
- No Linux o Tauri usa WebKitGTK, mais lento com blur e animações pesadas: a interface precisa de modo sem transparência (já previsto nos tokens) e testes no Ubuntu.
- O protótipo em HTML/CSS/JS (`prototype/`) é portado para componentes React.
