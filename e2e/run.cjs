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

teste("remover: confirma, o jogo some e os saves não são apagados", async (page) => {
  await page.goto(`${BASE}?cenario=jogos`);
  await page.getByRole("button", { name: "Remover" }).first().click();
  await page.getByText(/Nada é apagado/).waitFor();
  await page.getByRole("button", { name: "Remover" }).last().click();
  await page.getByText(/continuam onde estavam/).waitFor();
  await page.getByRole("heading", { name: "Rogue Legacy" }).waitFor({ state: "detached" });
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
