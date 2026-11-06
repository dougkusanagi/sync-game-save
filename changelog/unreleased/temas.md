# Temas

## Adicionado

- Temas embutidos escuro (padrão) e claro, com a escolha guardada no banco local.
- Pacotes de tema em `themes/<id>/theme.json`, aplicados só como tokens `--sgs-*`, sem executar CSS ou JavaScript.
- Seletor de tema em Configurações, com autor, versão e indicação de tema instalado.

## Segurança

- Temas não podem carregar URLs externas nem injetar regras; valores com `url()`, `@`, `;` ou caracteres de regra são recusados.
