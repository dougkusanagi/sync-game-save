// Testes E2E dos fluxos principais, com dados simulados (modo navegador do app).
// Uso: bun run e2e    (sobe o servidor do Vite sozinho e fecha no fim)
// Primeira vez na máquina: bunx playwright install chromium
const { chromium } = require("playwright");
const { spawn } = require("node:child_process");
const assert = require("node:assert/strict");
const http = require("node:http");
const path = require("node:path");

const PORTA = 1420;
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

teste("adicionar jogo: nome vazio bloqueia o envio", async (page) => {
  await page.goto(`${BASE}?cenario=vazio`);
  await page.getByRole("button", { name: "Adicionar jogo" }).first().click();
  await page.getByRole("button", { name: /Escolher pasta do jogo/ }).click();
  await page.getByLabel("Nome do jogo").fill("   ");
  assert.equal(await page.getByRole("button", { name: "Adicionar jogo" }).last().isDisabled(), true);
});

teste("jogar: sincroniza e avisa o resultado em linguagem simples", async (page) => {
  await page.goto(`${BASE}?cenario=jogos`);
  await page.getByRole("button", { name: "Jogar" }).first().click();
  await page.getByText("Save enviado para a pasta de sincronização").waitFor();
});

teste("conflito: o usuário escolhe e o jogo segue", async (page) => {
  await page.goto(`${BASE}?cenario=conflito`);
  await page.getByRole("button", { name: "Jogar" }).first().click();
  await page.getByRole("dialog", { name: "Dois saves diferentes" }).waitFor();
  await page.getByRole("button", { name: "Usar o save deste PC" }).click();
  await page.getByText("Save enviado para a pasta de sincronização").waitFor();
});

teste("jogo aberto em outro PC: pede confirmação antes de abrir", async (page) => {
  await page.goto(`${BASE}?cenario=jogos`);
  await page.getByRole("button", { name: "Jogar" }).nth(1).click();
  await page.getByRole("dialog", { name: "Jogo aberto em outro PC" }).getByText("PC-QUARTO").waitFor();
  await page.getByRole("button", { name: "Abrir mesmo assim" }).click();
  await page.getByText("Save enviado para a pasta de sincronização").waitFor();
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
  await page.getByRole("button", { name: "Conquistas" }).first().click();
  const dlg = page.getByRole("dialog", { name: "Conquistas de Rogue Legacy" });
  await dlg.getByRole("heading", { name: "2 de 3 conquistas · 66%" }).waitFor();
  await dlg.locator(".filtros").getByRole("button", { name: "Bloqueadas", exact: true }).click();
  await dlg.getByText("Conquista oculta").waitFor();
  await dlg.getByRole("button", { name: "Mostrar" }).click();
  await dlg.getByText("Segredo da família").waitFor();
});

teste("configurações: guarda e remove as credenciais da Steam", async (page) => {
  await page.goto(`${BASE}?cenario=jogos`);
  await page.getByRole("button", { name: "Configurações" }).click();
  const dlg = page.getByRole("dialog", { name: "Configurações" });
  const salvar = dlg.getByRole("button", { name: "Salvar credenciais" });
  assert.equal(await salvar.isDisabled(), true, "não salva sem a chave e o SteamID");
  await dlg.getByLabel("Chave de API").fill("CHAVE-DE-TESTE");
  await dlg.getByLabel("SteamID64").fill("76561198000000000");
  await salvar.click();
  await dlg.getByText("Credenciais da Steam guardadas neste PC.").waitFor();
  await dlg.getByRole("button", { name: "Remover credenciais" }).click();
  await dlg.getByText(/Credenciais da Steam removidas/).waitFor();
});

teste("remover: confirma, o jogo some e os saves não são apagados", async (page) => {
  await page.goto(`${BASE}?cenario=jogos`);
  await page.getByRole("button", { name: "Remover" }).first().click();
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
  await page.waitForSelector(".cartao");
  const antes = await page.evaluate(() => document.activeElement?.textContent);
  await page.keyboard.press("ArrowRight");
  const depois = await page.evaluate(() => document.activeElement?.textContent);
  assert.notEqual(antes, depois);
});

teste("layout: sem rolagem horizontal no celular, com nomes longos", async (page) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto(`${BASE}?cenario=jogos`);
  await page.waitForSelector(".cartao");
  const [larg, vis] = await page.evaluate(() => [document.documentElement.scrollWidth, window.innerWidth]);
  assert.ok(larg <= vis, `rolagem horizontal: ${larg} > ${vis}`);
});

(async () => {
  const servidor = await subirServidor();
  const browser = await chromium.launch();
  let falhas = 0;
  for (const [nome, fn] of testes) {
    const ctx = await browser.newContext({ viewport: { width: 1280, height: 720 } });
    const page = await ctx.newPage();
    page.setDefaultTimeout(6000);
    const erros = [];
    page.on("pageerror", (e) => erros.push(String(e)));
    try {
      await fn(page);
      assert.equal(erros.length, 0, `erros na página: ${erros.join(" | ")}`);
      console.log(`ok   ${nome}`);
    } catch (e) {
      falhas++;
      console.log(`FALHOU ${nome}\n     ${String(e.message).split("\n")[0]}`);
    }
    await ctx.close();
  }
  await browser.close();
  servidor?.kill();
  if (servidor) spawn("taskkill", ["/F", "/T", "/PID", String(servidor.pid)], { shell: true, stdio: "ignore" });
  console.log(falhas ? `\n${falhas} teste(s) falharam` : `\n${testes.length} testes passaram`);
  process.exit(falhas ? 1 : 0);
})();
