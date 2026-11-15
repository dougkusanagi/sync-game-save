// Testes E2E dos fluxos principais, com dados simulados (modo navegador do app).
// Uso: bun run e2e    (sobe o servidor do Vite sozinho e fecha no fim)
// Primeira vez na máquina: bunx playwright install chromium
const { chromium } = require("playwright");
const { spawn } = require("node:child_process");
const assert = require("node:assert/strict");
const http = require("node:http");
const path = require("node:path");

const PORTA = Number(process.env.PORTA ?? 1420);
const BASE = `http://localhost:${PORTA}/`;

function servidorNoAr() {
  return new Promise((ok) => {
    http.get(BASE, (r) => (r.resume(), ok(true))).on("error", () => ok(false));
  });
}

async function subirServidor() {
  if (await servidorNoAr()) return null;
  const p = spawn("bunx", ["vite", "--port", String(PORTA), "--strictPort"], { cwd: path.join(__dirname, ".."), shell: true, stdio: "ignore" });
  for (let i = 0; i < 60; i++) {
    if (await servidorNoAr()) return p;
    await new Promise((r) => setTimeout(r, 500));
  }
  p.kill();
  throw new Error("o servidor do Vite não subiu");
}

// Enquanto o jogo roda, Jogar fica travado; ao fechar sem problema, nenhum aviso aparece.
async function jogoTerminouSemAviso(page) {
  const jogar = page.locator(".hero").getByRole("button", { name: /Jogar|Jogando/ });
  await page.waitForFunction(() => document.querySelector(".hero .btn-play")?.disabled === true);
  await page.waitForFunction(() => document.querySelector(".hero .btn-play")?.disabled === false);
  assert.equal(await jogar.count(), 1);
  assert.equal(await page.locator(".toast").count(), 0, "não mostra aviso quando deu tudo certo");
}

const testes = [];
const teste = (nome, fn) => testes.push([nome, fn]);

teste("primeiro uso: escolhe a pasta, nomeia o PC e chega à biblioteca vazia", async (page) => {
  await page.goto(`${BASE}?cenario=primeiro-uso`);
  await page.getByRole("button", { name: "Escolher a pasta" }).click();
  await page.getByRole("button", { name: /Escolher pasta/ }).click();
  await page.getByText("Pasta pronta").waitFor();
  await page.getByRole("button", { name: "Continuar" }).click();
  const nome = page.getByLabel("Nome deste PC");
  await nome.fill("PC-QUARTO");
  await page.getByRole("button", { name: "Concluir" }).click();
  await page.getByText("Sua biblioteca está vazia").waitFor();
});

teste("adicionar jogo: reconhece, marca o save e o jogo aparece na lista", async (page) => {
  await page.goto(`${BASE}?cenario=vazio`);
  await page.getByRole("button", { name: "Adicionar jogo" }).first().click();
  await page.getByRole("button", { name: /Escolher pasta do jogo/ }).click();
  await page.getByText("Reconheci este jogo").waitFor();
  assert.equal(await page.getByRole("checkbox").first().isChecked(), true, "o save encontrado vem marcado");
  assert.equal(await page.getByRole("checkbox").nth(1).isChecked(), false, "o save que não existe vem desmarcado");
  await page.getByRole("button", { name: "Adicionar jogo" }).last().click();
  await page.getByRole("heading", { name: "Rogue Legacy" }).waitFor();
});

teste("varredura: revisa resultados e adiciona apenas jogo com save", async (page) => {
  await page.goto(BASE + "?cenario=vazio");
  await page.getByRole("button", { name: "Adicionar jogo" }).first().click();
  await page.getByRole("button", { name: /Procurar jogos em uma pasta/ }).click();
  await page.getByRole("button", { name: /Escolher pasta e procurar/ }).click();
  await page.getByText("Acesso negado").waitFor();
  await page.getByRole("button", { name: "Usar Outro candidato" }).click();
  await page.getByText("<base>/Saves").waitFor();
  await page.getByRole("button", { name: "Usar Jogo encontrado" }).click();
  await page.getByRole("checkbox", { name: "Adicionar Jogo encontrado" }).waitFor();
  assert.equal(await page.getByRole("checkbox", { name: "Adicionar Outro jogo" }).isDisabled(), true);
  const outro = page.locator("fieldset").filter({ hasText: "C:/Games/Outro" });
  await outro.getByRole("button", { name: /Adicionar pasta de save/ }).click();
  await page.waitForFunction(() => [...document.querySelectorAll("input[type=checkbox]")].some((el) => el.closest("label")?.textContent.includes("Adicionar Outro jogo") && !el.disabled));
  await page.getByRole("checkbox", { name: "Adicionar Jogo encontrado" }).check();
  await page.getByRole("button", { name: "Adicionar selecionados" }).click();
  await page.getByText(/Adicionado .* biblioteca/).waitFor();
  await page.getByRole("button", { name: "Concluir" }).click();
  await page.getByRole("heading", { name: "Jogo encontrado" }).waitFor();
});

teste("varredura: erro parcial preserva os jogos adicionados e permite corrigir", async (page) => {
  await page.goto(BASE + "?cenario=vazio");
  await page.getByRole("button", { name: "Adicionar jogo" }).first().click();
  await page.getByRole("button", { name: /Procurar jogos em uma pasta/ }).click();
  await page.getByRole("button", { name: /Escolher pasta e procurar/ }).click();
  const outro = page.locator("fieldset").filter({ hasText: "C:/Games/Outro" });
  await outro.getByRole("button", { name: /Adicionar pasta de save/ }).click();
  await page.waitForFunction(() => [...document.querySelectorAll("input[type=checkbox]")].some((el) => el.closest("label")?.textContent.includes("Adicionar Outro jogo") && !el.disabled));
  await page.getByRole("checkbox", { name: "Adicionar Jogo encontrado" }).check();
  await outro.getByLabel("Nome do jogo").fill("Jogo encontrado");
  await outro.getByRole("checkbox", { name: "Adicionar Jogo encontrado" }).check();
  await page.getByRole("button", { name: "Adicionar selecionados" }).click();
  await page.getByText("Já existe um jogo com este nome.").waitFor();
  assert.equal(await page.getByText(/Adicionado .* biblioteca/).count(), 1);
  await outro.getByLabel("Nome do jogo").fill("Outro jogo");
  await page.getByRole("button", { name: "Adicionar selecionados" }).click();
  await page.waitForFunction(() => [...document.querySelectorAll("p")].filter((el) => el.textContent.includes("Adicionado à biblioteca")).length === 2);
  await page.getByRole("button", { name: "Concluir" }).click();
  await page.getByRole("button", { name: /Biblioteca/ }).first().click();
  await page.getByRole("button", { name: /Jogo encontrado/ }).first().waitFor();
  await page.getByRole("button", { name: /Outro jogo/ }).first().waitFor();
});

teste("varredura: primeiro backup informa o resultado de cada jogo sem abrir nada", async (page) => {
  await page.goto(BASE + "?cenario=vazio");
  await page.getByRole("button", { name: "Adicionar jogo" }).first().click();
  await page.getByRole("button", { name: /Procurar jogos em uma pasta/ }).click();
  await page.getByRole("button", { name: /Escolher pasta e procurar/ }).click();
  const outro = page.locator("fieldset").filter({ hasText: "C:/Games/Outro" });
  await outro.getByRole("button", { name: /Adicionar pasta de save/ }).click();
  await page.waitForFunction(() => [...document.querySelectorAll("input[type=checkbox]")].some((el) => el.closest("label")?.textContent.includes("Adicionar Outro jogo") && !el.disabled));
  await page.getByRole("checkbox", { name: "Adicionar Jogo encontrado" }).check();
  await page.getByRole("checkbox", { name: "Adicionar Outro jogo" }).check();
  await page.getByRole("button", { name: "Adicionar selecionados" }).click();
  await page.getByText("Primeira cópia do save guardada na pasta de sincronização.").waitFor();
  await page.getByText(/já tem outra versão deste save. Nada foi alterado/).waitFor();
  assert.equal(await page.getByRole("button", { name: "Tentar o backup de novo" }).count(), 0);
});

teste("varredura: backup recusado por outro PC pode ser repetido sem recriar o jogo", async (page) => {
  await page.goto(BASE + "?cenario=backup-ocupado");
  await page.getByRole("button", { name: "Adicionar jogo" }).first().click();
  await page.getByRole("button", { name: /Procurar jogos em uma pasta/ }).click();
  await page.getByRole("button", { name: /Escolher pasta e procurar/ }).click();
  await page.getByRole("checkbox", { name: "Adicionar Jogo encontrado" }).check();
  await page.getByRole("button", { name: "Adicionar selecionados" }).click();
  await page.getByText(/PC-QUARTO está jogando agora/).waitFor();
  await page.getByRole("button", { name: "Tentar o backup de novo" }).click();
  await page.getByText("Primeira cópia do save guardada na pasta de sincronização.").waitFor();
  assert.equal(await page.getByRole("button", { name: "Tentar o backup de novo" }).count(), 0);
  assert.equal(await page.getByText("Adicionado à biblioteca.").count(), 1);
});

teste("central rápida: F1 mostra o estado, envia os saves e devolve o foco", async (page) => {
  await page.goto(BASE);
  await page.getByRole("button", { name: /Jogar/ }).first().waitFor();
  await page.keyboard.press("F1");
  const dialogo = page.getByRole("dialog", { name: "Central rápida" });
  await dialogo.waitFor();
  await dialogo.getByText(/Sincronizado|Ainda não sincronizado/).first().waitFor();
  await dialogo.getByRole("button", { name: "Sincronizar tudo agora" }).click();
  await dialogo.getByText("Save enviado.").first().waitFor();
  await dialogo.getByRole("checkbox", { name: /Música ambiente/ }).waitFor();
  await page.keyboard.press("Escape");
  await dialogo.waitFor({ state: "detached" });
});

teste("central rápida: no celular não rola para os lados", async (page) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto(BASE);
  await page.getByRole("button", { name: /Jogar/ }).first().waitFor();
  await page.keyboard.press("F1");
  await page.getByRole("dialog", { name: "Central rápida" }).waitFor();
  assert.equal(await page.evaluate(() => document.documentElement.scrollWidth > innerWidth), false);
});

teste("adicionar jogo: nome vazio bloqueia o envio", async (page) => {
  await page.goto(`${BASE}?cenario=vazio`);
  await page.getByRole("button", { name: "Adicionar jogo" }).first().click();
  await page.getByRole("button", { name: /Escolher pasta do jogo/ }).click();
  await page.getByLabel("Nome do jogo").fill("   ");
  assert.equal(await page.getByRole("button", { name: "Adicionar jogo" }).last().isDisabled(), true);
});

teste("jogar: trava o botão enquanto o jogo roda e não mostra aviso quando dá tudo certo", async (page) => {
  await page.goto(`${BASE}?cenario=jogos`);
  await page.getByRole("button", { name: "Jogar" }).first().click();
  await jogoTerminouSemAviso(page);
});

teste("conflito: o usuário escolhe e o jogo segue", async (page) => {
  await page.goto(`${BASE}?cenario=conflito`);
  await page.getByRole("button", { name: "Jogar" }).first().click();
  await page.getByRole("dialog", { name: "Dois saves diferentes" }).waitFor();
  await page.getByRole("button", { name: "Usar o save deste PC" }).click();
  await jogoTerminouSemAviso(page);
});

teste("jogo aberto em outro PC: pede confirmação antes de abrir", async (page) => {
  await page.goto(`${BASE}?cenario=jogos`);
  await page.getByRole("button", { name: /^Hollow Knight/ }).click();
  await page.getByRole("dialog", { name: "Jogo aberto em outro PC" }).getByText("PC-QUARTO").waitFor();
  await page.getByRole("button", { name: "Abrir mesmo assim" }).click();
  await jogoTerminouSemAviso(page);
});

teste("versões: restaurar pede confirmação e informa onde ficou o backup", async (page) => {
  await page.goto(`${BASE}?cenario=jogos`);
  await page.getByRole("button", { name: "Versões" }).first().click();
  await page.getByRole("button", { name: "Restaurar" }).first().click();
  await page.getByRole("button", { name: "Confirmar" }).click();
  await page.getByText(/foi guardado em/).waitFor();
});

teste("conquistas: mostra o progresso, filtra e revela as ocultas", async (page) => {
  await page.goto(`${BASE}?cenario=jogos`);
  await page.getByRole("button", { name: /^Rogue Legacy./ }).hover();
  await page.getByRole("button", { name: "Conquistas" }).first().click();
  const dlg = page.getByRole("dialog", { name: "Conquistas de Rogue Legacy" });
  await dlg.getByRole("heading", { name: "2 de 3 conquistas · 66%" }).waitFor();
  await dlg.locator(".filtros").getByRole("button", { name: "Bloqueadas", exact: true }).click();
  await dlg.getByText("Conquista oculta").waitFor();
  await dlg.getByRole("button", { name: "Mostrar" }).click();
  await dlg.getByText("Segredo da família").waitFor();
});

teste("conquistas do HUB: mostra marcos e o resumo do progresso", async (page) => {
  await page.goto(`${BASE}?cenario=jogos`);
  await page.getByRole("button", { name: "Meu progresso" }).click();
  const dlg = page.getByRole("dialog", { name: "Conquistas do HUB" });
  await dlg.getByRole("heading", { name: "Faltam 7 marcos para completar." }).waitFor();
  await dlg.getByText("Ritmo constante").waitFor();
  await dlg.getByText("Jogos completos").waitFor();
});

teste("configurações: sem chave, usa a conta da Steam deste PC e permite trocar", async (page) => {
  await page.goto(`${BASE}?cenario=jogos`);
  await page.getByRole("button", { name: "Configurações" }).click();
  const dlg = page.getByRole("dialog", { name: "Configurações" });
  await dlg.getByText("Conta da Steam encontrada (SteamID 76561198000000000).").waitFor();
  assert.equal(await dlg.getByLabel(/chave/i).count(), 0, "não pede nenhuma chave");
  await dlg.getByText("Usar outra conta").click();
  const usar = dlg.getByRole("button", { name: "Usar esta conta" });
  await dlg.getByLabel("SteamID64").fill("123");
  assert.equal(await usar.isDisabled(), true, "SteamID incompleto não é aceito");
  await dlg.getByLabel("SteamID64").fill("76561198263490373");
  await usar.click();
  await dlg.getByText("Conta da Steam encontrada (SteamID 76561198263490373).").waitFor();
  await dlg.getByRole("button", { name: "Usar a conta deste PC" }).click();
  await dlg.getByText("Conta da Steam encontrada (SteamID 76561198000000000).").waitFor();
});

teste("tema: troca para o claro e a escolha fica guardada", async (page) => {
  await page.goto(`${BASE}?cenario=jogos`);
  await page.getByRole("button", { name: "Configurações" }).click();
  const dlg = page.getByRole("dialog", { name: "Configurações" });
  const aparencia = dlg.getByLabel("Aparência");
  await aparencia.selectOption({ label: "Claro" });
  await page.waitForFunction(() => getComputedStyle(document.documentElement).getPropertyValue("--sgs-bg").trim() === "#f2f5fb");
  assert.equal(await page.evaluate(() => getComputedStyle(document.body).backgroundColor), "rgb(242, 245, 251)");
  await page.waitForFunction(() => localStorage.getItem("sgs.mock.tema") === "claro");

  await page.reload();
  await page.waitForFunction(() => getComputedStyle(document.documentElement).getPropertyValue("--sgs-bg").trim() === "#f2f5fb");

  await page.getByRole("button", { name: "Configurações" }).click();
  await page.getByRole("dialog", { name: "Configurações" }).getByLabel("Aparência").selectOption({ label: "Escuro (padrão)" });
  await page.waitForFunction(() => document.documentElement.style.getPropertyValue("--sgs-bg") === "");
});

teste("remover: confirma, o jogo some e os saves não são apagados", async (page) => {
  await page.goto(`${BASE}?cenario=jogos`);
  await page.getByRole("button", { name: /^Rogue Legacy./ }).hover();
  await page.getByRole("button", { name: "Mais opções" }).click();
  await page.getByRole("button", { name: "Remover da biblioteca" }).click();
  await page.getByText(/Nada é apagado/).waitFor();
  await page.getByRole("button", { name: "Remover" }).last().click();
  await page.getByText(/continuam onde estavam/).waitFor();
  await page.getByRole("heading", { name: "Rogue Legacy" }).waitFor({ state: "detached" });
});

teste("som: desligar e mudar o volume fica guardado depois de recarregar", async (page) => {
  await page.goto(`${BASE}?cenario=jogos`);
  await page.getByRole("button", { name: "Configurações" }).click();
  const musica = page.getByRole("checkbox", { name: "Música ambiente" });
  assert.equal(await musica.isChecked(), true, "música ligada por padrão, em volume baixo");
  await musica.uncheck();
  // Com a música desligada, os botões de volume dela ficam desativados.
  assert.equal(await page.getByRole("button", { name: "Aumentar volume da música ambiente" }).isDisabled(), true);
  await page.getByRole("button", { name: "Aumentar volume dos sons da interface" }).click();
  const salvo = await page.evaluate(() => JSON.parse(localStorage.getItem("sgs.som")));
  assert.equal(salvo.fundo, false);
  assert.ok(salvo.volEfeitos > 0.3 && salvo.volEfeitos < 0.4, `volume dos efeitos: ${salvo.volEfeitos}`);
  await page.reload();
  await page.getByRole("button", { name: "Configurações" }).click();
  assert.equal(await page.getByRole("checkbox", { name: "Música ambiente" }).isChecked(), false);
});

teste("teclado: Esc fecha o diálogo e devolve o foco ao botão de origem", async (page) => {
  await page.goto(`${BASE}?cenario=jogos`);
  const botao = page.getByRole("button", { name: "Configurações" });
  await botao.focus();
  await botao.press("Enter");
  await page.getByRole("dialog", { name: "Configurações" }).waitFor();
  await page.keyboard.press("Escape");
  await page.getByRole("dialog").waitFor({ state: "detached" });
  assert.equal(await page.evaluate(() => document.activeElement?.textContent), "Configurações");
});

teste("teclado: setas movem o foco entre os botões", async (page) => {
  await page.goto(`${BASE}?cenario=jogos`);
  await page.waitForSelector(".cap");
  const antes = await page.evaluate(() => document.activeElement?.textContent);
  await page.keyboard.press("ArrowRight");
  const depois = await page.evaluate(() => document.activeElement?.textContent);
  assert.notEqual(antes, depois);
});

teste("início: Q e E trocam a seção; L abre a biblioteca e Esc volta", async (page) => {
  await page.goto(`${BASE}?cenario=jogos`);
  await page.waitForSelector(".cap.is-focused");
  await page.keyboard.press("e");
  await page.getByRole("heading", { name: "Adicionados" }).waitFor();
  await page.keyboard.press("q");
  await page.getByRole("heading", { name: "Jogados recentemente" }).waitFor();
  await page.keyboard.press("l");
  await page.getByRole("heading", { name: "Biblioteca" }).waitFor();
  assert.equal(await page.locator(".card").count(), 4);
  await page.keyboard.press("Escape");
  await page.getByRole("heading", { name: "Jogados recentemente" }).waitFor();
});

teste("sobre: mostra a versão e procura atualização", async (page) => {
  await page.goto(`${BASE}?cenario=jogos`);
  await page.getByRole("button", { name: "Configurações" }).click();
  await page.getByRole("dialog", { name: "Configurações" }).getByRole("button", { name: "Sobre" }).click();
  const dlg = page.getByRole("dialog", { name: "Sobre" });
  await dlg.getByText(/Versão 1.1.0/).waitFor();
  await dlg.getByText("Você está na versão mais recente.").waitFor();
});

teste("layout: sem rolagem horizontal no celular, com nomes longos", async (page) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto(`${BASE}?cenario=jogos`);
  await page.waitForSelector(".cap");
  const [larg, vis] = await page.evaluate(() => [document.documentElement.scrollWidth, window.innerWidth]);
  assert.ok(larg <= vis, `rolagem horizontal: ${larg} > ${vis}`);
});

teste("parada: cancelar mantém o jogo e força exige confirmação", async (page) => {
  await page.goto(`${BASE}?cenario=parada`);
  await page.getByRole("button", { name: "Jogar" }).first().click();
  const parar = page.getByRole("button", { name: "Parar jogo" });
  await parar.click();
  let dlg = page.getByRole("dialog", { name: /^Parar / });
  await dlg.getByText(/pode corromper ou perder o save/).waitFor();
  await page.screenshot({ path: path.join(require("node:os").tmpdir(), "sgs-parada-desktop.png") });
  await page.setViewportSize({ width: 390, height: 844 });
  await page.screenshot({ path: path.join(require("node:os").tmpdir(), "sgs-parada-mobile.png") });
  const dimensoes = await page.evaluate(() => [document.documentElement.scrollWidth, innerWidth]);
  assert.ok(dimensoes[0] <= dimensoes[1], `rolagem horizontal: ${dimensoes}`);
  await dlg.getByRole("button", { name: "Continuar aguardando" }).click();
  await dlg.waitFor({ state: "detached" });
  assert.equal(await parar.isVisible(), true);
  await parar.click();
  dlg = page.getByRole("dialog", { name: /^Parar / });
  await dlg.getByRole("button", { name: "Forçar fechamento" }).click();
  await dlg.getByRole("button", { name: "Entendi" }).click();
  await page.getByRole("button", { name: "Jogar" }).first().waitFor();
});

teste("Wails usa estado real e encaminha diálogos, eventos e jogos", async (page) => {
  await page.addInitScript(() => { window._wails = { environment: { OS: "windows", Arch: "amd64", Debug: true } }; });
  const chamadas = [];
  let testePonte = false;
  await page.route("**/wails/runtime", async (route) => {
    const chamada = route.request().postDataJSON();
    if (testePonte) chamadas.push(chamada);
    const metodo = chamada.args?.methodName;
    const resposta = metodo === "main.App.Estado" ? { pasta_sync: null, nome_pc: "PC", catalogo_pronto: false, operacoes_interrompidas: [], steam_configurada: false, steam_id: null, steamgriddb_configurada: false }
      : metodo === "main.App.AdicionarJogo" ? 42
      : metodo === "main.App.VerificarAtualizacao" ? { versao: "2.0.0", atual: "1.0.0", notas: null }
      : metodo === "main.App.ListarJogos" ? (testePonte ? [{ id: 42, name: "Jogo", save_paths: [] }] : [])
      : chamada.object === 0 ? "1.0.0" : "C:/Jogos/Teste";
    await route.fulfill({ json: resposta });
  });
  await page.goto(`${BASE}?cenario=jogos`);
  await page.getByRole("heading", { name: "Vamos guardar seus saves com segurança" }).waitFor();
  assert.equal(await page.getByRole("heading", { name: "Rogue Legacy" }).count(), 0);
  testePonte = true;
  const resultado = await page.evaluate(async () => {
    const ponte = await import("/src/api.ts");
    const eventos = [];
    const cancelar = await ponte.aoMudarEstadoDoJogo((id, estado) => eventos.push([id, estado]));
    window._wails.dispatchWailsEvent({ name: "jogo-estado", data: { id: 7, estado: "jogando" } });
    cancelar();
    window._wails.dispatchWailsEvent({ name: "jogo-estado", data: { id: 7, estado: "salvando" } });
    const atualizacao = await ponte.verificarAtualizacao();
    const jogo = { name: "Jogo", exe: "jogo.exe", install_dir: null, catalog_id: null, save_paths: [] };
    const id = await ponte.api.adicionarJogo(jogo);
    const jogos = await ponte.api.listarJogos();
    return { eventos, pasta: await ponte.escolherPasta("Escolher pasta"), sobre: await ponte.api.sobre(), atualizacao, id, jogos };
  });
  assert.deepEqual(resultado.eventos, [[7, "jogando"]]);
  assert.equal(resultado.pasta, "C:/Jogos/Teste");
  assert.deepEqual(resultado.atualizacao, { versao: "2.0.0", atual: "1.0.0", notas: null });
  assert.deepEqual(resultado.sobre, { nome: "Sync Game Save", versao: "1.0.0", build: "" });
  assert.equal(resultado.id, 42);
  assert.equal(resultado.jogos[0].name, "Jogo");
  assert.equal(chamadas.length, 5);
  assert.equal(chamadas[0].args.methodName, "main.App.VerificarAtualizacao");
  assert.equal(chamadas[1].args.methodName, "main.App.AdicionarJogo");
  assert.equal(chamadas[1].args.args[0].exe, "jogo.exe");
  assert.equal(chamadas[2].args.methodName, "main.App.ListarJogos");
  assert.equal(chamadas[3].args.CanChooseDirectories, true);
  assert.equal(chamadas[3].args.CanChooseFiles, false);
});
(async () => {
  const servidor = await subirServidor();
  const browser = await chromium.launch();
  let falhas = 0;
  for (const [nome, fn] of testes) {
    if (process.env.E2E_FILTRO && !nome.includes(process.env.E2E_FILTRO)) continue;
    const ctx = await browser.newContext({ viewport: { width: 1280, height: 720 } });
    const page = await ctx.newPage();
    await page.route("https://cdn.cloudflare.steamstatic.com/**", (route) => route.abort());
    page.setDefaultTimeout(6000);
    const erros = [];
    page.on("pageerror", (e) => erros.push(String(e)));
    try {
      await fn(page);
      assert.equal(erros.length, 0, `erros na página: ${erros.join(" | ")}`);
      console.log(`ok   ${nome}`);
    } catch (e) {
      falhas++;
      console.log(`FALHOU ${nome}\n     ${e.stack ?? e}`);
    }
    await ctx.close();
  }
  await browser.close();
  servidor?.kill();
  if (servidor) spawn("taskkill", ["/F", "/T", "/PID", String(servidor.pid)], { shell: true, stdio: "ignore" });
  console.log(falhas ? `\n${falhas} teste(s) falharam` : `\n${testes.length} testes passaram`);
  process.exit(falhas ? 1 : 0);
})();
teste("tema rapido fica disponivel na barra superior", async (page) => {
  await page.goto(`${BASE}?cenario=jogos`);
  const seletor = page.getByLabel("Tema visual");
  await seletor.waitFor();
  await seletor.selectOption("claro");
  await page.waitForFunction(() => localStorage.getItem("sgs.mock.tema") === "claro");
  await seletor.selectOption("escuro");
});
teste("descobrir save: compara as pastas depois do jogo e marca a sugestão", async (page) => {
  await page.goto(`${BASE}?cenario=vazio`);
  await page.getByRole("button", { name: "Adicionar jogo" }).first().click();
  await page.getByRole("button", { name: /Escolher pasta do jogo/ }).click();
  await page.getByText("Reconheci este jogo").waitFor();
  const dlg = page.getByRole("dialog", { name: "Adicionar jogo" });
  await dlg.getByRole("button", { name: /Descobrir onde o jogo salva/ }).click();
  const opcao = dlg.locator("label.opcao", { hasText: "Rogue Legacy/Saves" });
  await opcao.waitFor();
  assert.equal(await opcao.locator("input").isChecked(), true, "o candidato mais provável vem marcado para o usuário confirmar");
  // Confirmar é manual: nada entra na biblioteca sem o clique em Adicionar.
  await dlg.getByRole("button", { name: "Adicionar jogo", exact: true }).click();
  await page.getByRole("heading", { name: "Rogue Legacy" }).waitFor();
});

teste("descobrir save: sem mudanças avisa com o que fazer e mantém o formulário", async (page) => {
  await page.goto(`${BASE}?cenario=sem-descoberta`);
  await page.getByRole("button", { name: "Adicionar jogo" }).first().click();
  await page.getByRole("button", { name: /Escolher pasta do jogo/ }).click();
  await page.getByText("Reconheci este jogo").waitFor();
  const dlg = page.getByRole("dialog", { name: "Adicionar jogo" });
  await dlg.getByRole("button", { name: /Descobrir onde o jogo salva/ }).click();
  await dlg.getByText(/Nenhum arquivo mudou/).waitFor();
  assert.equal(await dlg.getByRole("button", { name: "Adicionar jogo", exact: true }).isEnabled(), true, "o formulário continua completo");
  await dlg.getByLabel("Nome do jogo").fill("Nome preservado");
  assert.equal(await dlg.getByLabel("Nome do jogo").inputValue(), "Nome preservado");
});

teste("descobrir save: cancelar durante o jogo devolve o formulário", async (page) => {
  await page.goto(`${BASE}?cenario=descoberta-enrolada`);
  await page.getByRole("button", { name: "Adicionar jogo" }).first().click();
  await page.getByRole("button", { name: /Escolher pasta do jogo/ }).click();
  await page.getByText("Reconheci este jogo").waitFor();
  const dlg = page.getByRole("dialog", { name: "Adicionar jogo" });
  await dlg.getByRole("button", { name: /Descobrir onde o jogo salva/ }).click();
  await dlg.getByRole("status").getByRole("button", { name: "Cancelar descoberta" }).click();
  await dlg.getByRole("button", { name: /Descobrir onde o jogo salva/ }).waitFor();
  assert.equal(await dlg.getByRole("button", { name: "Adicionar jogo", exact: true }).isEnabled(), true, "o formulário volta a funcionar após cancelar");
});

teste("adicionar jogo aceita selecionar diretamente o executavel", async (page) => {
  await page.goto(`${BASE}?cenario=vazio`);
  await page.getByRole("button", { name: "Adicionar jogo" }).first().click();
  await page.getByRole("dialog", { name: "Adicionar jogo" }).getByRole("button", { name: /Escolher execut/ }).click();
  await page.getByRole("textbox", { name: "Nome do jogo" }).waitFor();
  await page.getByRole("textbox", { name: "Nome do jogo" }).fill("Rogue Legacy");
  await page.getByRole("dialog", { name: "Adicionar jogo" }).getByRole("button", { name: "Adicionar jogo", exact: true }).click();
  await page.getByRole("button", { name: /^Rogue Legacy/ }).waitFor();
});
teste("biblioteca: busca acha mesmo com acento e filtro sem resultado limpa", async (page) => {
  await page.goto(`${BASE}?cenario=jogos`);
  await page.waitForSelector(".cap");
  await page.keyboard.press("l");
  await page.getByRole("heading", { name: "Biblioteca" }).waitFor();
  assert.equal(await page.locator(".card").count(), 4);
  await page.getByLabel("Buscar jogo pelo nome").fill("céleste");
  assert.equal(await page.locator(".card").count(), 1, "a busca sem acento acha o jogo");
  await page.getByRole("button", { name: "Sincronizados" }).click();
  await page.getByRole("heading", { name: "Nenhum jogo com estes filtros" }).waitFor();
  assert.ok(await page.getByRole("button", { name: "Limpar filtros" }).isVisible());
  await page.getByRole("button", { name: "Limpar filtros" }).click();
  assert.equal(await page.locator(".card").count(), 4, "limpar filtros devolve todos os jogos");
  assert.equal(await page.getByLabel("Buscar jogo pelo nome").inputValue(), "");
  await page.getByText("4 jogos", { exact: true }).waitFor();
});

teste("biblioteca: filtros contam, ordena e devolve os filtros ao voltar", async (page) => {
  await page.goto(`${BASE}?cenario=jogos`);
  await page.waitForSelector(".cap");
  await page.keyboard.press("l");
  await page.getByRole("heading", { name: "Biblioteca" }).waitFor();
  await page.getByRole("button", { name: "Sem sincronização" }).click();
  await page.getByText("1 de 4 jogos").waitFor();
  assert.equal(await page.locator(".card-name").first().textContent(), "Celeste");
  await page.getByRole("button", { name: "Todos" }).click();
  await page.getByLabel("Ordenar por").selectOption({ label: "Mais tempo de jogo" });
  assert.equal(await page.locator(".card-name").first().textContent(), "Rogue Legacy");
  await page.getByRole("button", { name: "Sem sincronização" }).click();
  await page.keyboard.press("Escape");
  await page.getByRole("heading", { name: "Jogados recentemente" }).waitFor();
  await page.keyboard.press("l");
  await page.getByRole("heading", { name: "Biblioteca" }).waitFor();
  assert.equal(await page.getByRole("button", { name: "Sem sincronização" }).getAttribute("aria-pressed"), "true", "o filtro continua marcado ao voltar");
  assert.equal(await page.locator(".card").count(), 1);
});

teste("editar jogo salva nome e caminhos de save", async (page) => {
  await page.goto(`${BASE}?cenario=jogos`);
  await page.getByRole("button", { name: /^Rogue Legacy/ }).hover();
  await page.getByRole("button", { name: "Mais opções" }).click();
  await page.getByRole("button", { name: "Editar jogo" }).click();
  await page.getByLabel("Nome do jogo").fill("Rogue Legacy Editado");
  await page.getByRole("button", { name: "Salvar alterações" }).click();
  await page.getByRole("heading", { name: "Rogue Legacy Editado" }).waitFor();
});
