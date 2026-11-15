Alterado: a migração para Wails prepara diálogos, eventos e acesso às artes sem apresentar dados simulados como biblioteca real.
Corrigido: a versão Wails em migração reconhece o WebView real e não apresenta dados de demonstração como biblioteca real.
Adicionado: listagem e inclusão de jogos no backend Wails, usando a biblioteca local compatível.
Adicionado: primeiro uso e configuração da pasta de sincronização na versão Wails em desenvolvimento.
Adicionado: cadastro manual de jogos sem catálogo no Wails; corrigida a rolagem horizontal no diálogo de cadastro.
Adicionado: edição e remoção de jogos no Wails, preservando os saves e invalidando a base ao trocar pastas.
Alterado: o aplicativo passa a ser feito em Go com Wails (no lugar de Rust com Tauri); a biblioteca, o catálogo, os saves e os temas continuam os mesmos e são lidos sem conversão.
Alterado: a atualização do app usa um novo canal com assinatura própria; quem usa a versão 1.x receberá a versão Go por uma versão de transição.
Alterado: o instalador do Windows continua por usuário e na mesma pasta, e agora vem em português.
