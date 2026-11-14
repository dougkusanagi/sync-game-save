// Smoke test do WebView Wails real enquanto os serviços Go são portados (Windows).
const { chromium } = require("playwright");
const { spawn } = require("node:child_process");
const fs = require("node:fs");
const net = require("node:net");
const os = require("node:os");
const path = require("node:path");

async function portaLivre() {
  const servidor = net.createServer();
  await new Promise((resolve, reject) => servidor.once("error", reject).listen(0, "127.0.0.1", resolve));
  const porta = servidor.address().port;
  await new Promise((resolve, reject) => servidor.close((err) => err ? reject(err) : resolve()));
  return porta;
}

async function aguardarDepurador(url, app) {
  for (let i = 0; i < 100; i++) {
    if (app.exitCode !== null) throw new Error(`O aplicativo encerrou com código ${app.exitCode}.`);
    try { if ((await fetch(url)).ok) return; } catch {}
    await new Promise((resolve) => setTimeout(resolve, 200));
  }
  throw new Error("O depurador do WebView2 não iniciou.");
}

(async () => {
  if (process.platform !== "win32") throw new Error("Este teste requer Windows e WebView2.");
  const pasta = fs.mkdtempSync(path.join(os.tmpdir(), "sgs-wails-e2e-"));
  const nuvem = path.join(pasta, "nuvem");
  fs.mkdirSync(nuvem);
  const porta = await portaLivre();
  const url = `http://127.0.0.1:${porta}`;
  const app = spawn(path.join(__dirname, "../bin/sync-game-save-e2e.exe"), [], {
    env: { ...process.env, SGS_DATA_DIR: path.join(pasta, "dados"), SGS_STEAM_DIR: path.join(pasta, "steam"), SGS_E2E_PORT: String(porta) },
    windowsHide: true,
    stdio: "ignore",
  });
  let browser;
  try {
    await aguardarDepurador(`${url}/json/version`, app);
    browser = await chromium.connectOverCDP(url);
    const context = browser.contexts()[0];
    let page = context.pages()[0];
    if (!page) page = await context.waitForEvent("page", { timeout: 10000 });
    await page.getByRole("heading", { name: "Vamos guardar seus saves com segurança" }).waitFor({ timeout: 15000 });
    for (const [nome, width, height] of [["mobile", 390, 844], ["desktop", 1280, 720]]) {
      await page.setViewportSize({ width, height });
      if (await page.evaluate(() => document.documentElement.scrollWidth > innerWidth)) throw new Error(`Rolagem horizontal inesperada: ${nome}`);
      await page.screenshot({ path: path.join(pasta, `${nome}.png`) });
    }
    await page.route("**/wails/runtime", async (route) => {
      const request = route.request().postDataJSON();
      if (request.object === 5 && request.method === 4) await route.fulfill({ json: nuvem });
      else await route.continue();
    });
    await page.getByRole("button", { name: "Escolher a pasta" }).click();
    await page.getByRole("button", { name: "Escolher pasta…" }).click();
    await page.getByText("Pasta pronta").waitFor();
    await page.getByRole("button", { name: "Continuar" }).click();
    await page.getByLabel("Nome deste PC").fill("PC-TESTE");
    await page.getByRole("button", { name: "Concluir" }).click();
    await page.getByText("Sua biblioteca está vazia").waitFor({ timeout: 15000 });
    if (!fs.existsSync(path.join(nuvem, "SyncGameSave", "devices.json"))) throw new Error("A configuração não registrou o PC na nuvem.");
    console.log("ok   primeiro uso configura uma pasta temporária no WebView Wails real");
    const info = await page.evaluate(async () => {
      const runtime = await import("/wails/runtime.js");
      const evento = new Promise((resolve, reject) => {
        const timeout = setTimeout(() => reject(new Error("Evento Wails não chegou.")), 5000);
        const cancelar = runtime.Events.On("jogo-estado", (e) => { clearTimeout(timeout); cancelar(); resolve(e.data); });
        runtime.Events.Emit("jogo-estado", { id: 17, estado: "jogando" }).catch(reject);
      });
      const id = await runtime.Call.ByName("main.App.AdicionarJogo", { name: "Jogo de teste", exe: "jogo.exe", install_dir: null, catalog_id: null, save_paths: ["<base>/save"] });
      const jogos = await runtime.Call.ByName("main.App.ListarJogos");
      return { ambiente: window._wails?.environment?.OS, versao: await runtime.Call.ByName("main.App.Versao"), evento: await evento, id, jogos };
    });
    if (info.ambiente !== "windows" || info.versao !== "1.0.0") throw new Error(`Binding ou ambiente inesperado: ${JSON.stringify(info)}`);
    console.log("ok   WebView Wails real reconhece o ambiente e chama o binding de versão");
    if (info.evento.id !== 17 || info.evento.estado !== "jogando") throw new Error(`Evento inesperado: ${JSON.stringify(info.evento)}`);
    if (info.jogos.length !== 1 || info.jogos[0].id !== info.id || info.jogos[0].name !== "Jogo de teste" || info.jogos[0].save_paths[0] !== "<base>/save") throw new Error(`Biblioteca inesperada: ${JSON.stringify(info.jogos)}`);
    console.log("ok   o evento de jogo trafega pelo runtime Wails real");
    console.log("ok   adicionar e listar jogos usam a biblioteca SQLite temporária");
    await page.reload();
    await page.getByRole("heading", { name: "Jogo de teste" }).waitFor({ timeout: 15000 });
    console.log("ok   jogo salvo aparece na interface após recarregar");

    console.log(`Capturas: ${pasta}`);
  } finally {
    if (browser) await browser.close();
    app.kill();
  }
})().catch((err) => { console.error(err); process.exitCode = 1; });
