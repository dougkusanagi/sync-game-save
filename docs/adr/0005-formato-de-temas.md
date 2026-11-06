# 0005. Temas como tokens JSON, sem CSS ou JavaScript

- Status: Aceito
- Data: 2026-11-06

## Contexto

O plano previa temas da comunidade desde o protótipo: todas as cores, raios, fontes e durações vêm de tokens `--sgs-*`. A primeira versão precisava definir um formato instalável sem abrir uma superfície de execução de código na interface, que roda com acesso aos saves do usuário.

## Decisão

- Um tema é uma pasta `themes/<id>/` na pasta de dados do app, com `theme.json`: `name`, `author`, `version`, `min_app_version` opcional e `tokens`.
- Só tokens com prefixo `--sgs-` são aceitos. Os valores são aplicados como propriedades no elemento raiz; nenhum CSS, JavaScript, `@import` ou `url()` do tema é executado.
- Valores com `@`, `url(`, `expression(`, `;`, chaves, sinais de menor/maior ou caracteres de controle são recusados. Pacotes inválidos ou com `min_app_version` maior que a do app são ignorados.
- O app traz dois temas embutidos: `escuro` (padrão, sem trocas) e `claro`, que também serve de exemplo.
- A escolha fica em `settings` no `library.db` e é aplicada ao abrir o app.
- Arquivos `theme.css` e `assets/` ainda não são lidos; ficam reservados para uma decisão futura.

## Alternativas

- CSS arbitrário do tema com CSP restrita: permitiria layouts completos, mas amplia o risco de sobreposição da interface, phishing e carregamento externo.
- Um arquivo CSS com só `:root { --sgs-... }`: equivalente aos tokens JSON, porém exige analisar CSS e cria ambiguidade sobre quais regras valem.

## Consequências

- Temas não conseguem mudar estrutura, textos, imagens de fundo ou sons.
- O formato é estável e fácil de validar; qualquer token novo em `src/tokens.css` passa a estar disponível sem mudar o carregador.
- Quem quiser um tema mais completo depende de uma nova decisão que amplie o formato.
