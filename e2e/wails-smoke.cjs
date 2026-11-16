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
  const jogoDir = path.join(pasta, "Jogo_Exemplo");
  const saveDir = path.join(pasta, "saves");
  fs.mkdirSync(jogoDir);
  fs.mkdirSync(saveDir);
  const save = path.join(saveDir, "slot.sav");
  fs.writeFileSync(save, "progresso preservado");
  const executavel = path.join(jogoDir, "jogo.exe");
  fs.writeFileSync(executavel, "executável de teste");
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
    await page.emulateMedia({ reducedMotion: "reduce" });
    await page.getByRole("heading", { name: "Vamos guardar seus saves com segurança" }).waitFor({ timeout: 15000 });
    for (const [nome, width, height] of [["mobile", 390, 844], ["desktop", 1280, 720]]) {
      await page.setViewportSize({ width, height });
      if (await page.evaluate(() => document.documentElement.scrollWidth > innerWidth)) throw new Error(`Rolagem horizontal inesperada: ${nome}`);
      await page.screenshot({ path: path.join(pasta, `${nome}.png`) });
    }
    const chamadasDialogo = [];
    await page.route("**/wails/runtime", async (route) => {
      const request = route.request().postDataJSON();
      if (request.object === 5) chamadasDialogo.push(request);
      if (request.object === 5 && request.method === 4) await route.fulfill({ json: request.args.Title.includes("executável") ? executavel : request.args.Title.includes("onde o jogo guarda o save") ? saveDir : nuvem });
      else await route.continue();
    });
    await page.getByRole("button", { name: "Escolher a pasta" }).click();
    await page.getByRole("button", { name: "Escolher pasta…" }).click();
    await page.getByText("Pasta pronta").waitFor({ timeout: 10000 }).catch(async (err) => { throw new Error(`${err.message} Tela: ${(await page.locator("body").innerText()).slice(0, 500)}; diálogos: ${JSON.stringify(chamadasDialogo)}`); });
    await page.getByRole("button", { name: "Continuar" }).click();
    await page.getByLabel("Nome deste PC").fill("PC-TESTE");
    await page.getByRole("button", { name: "Concluir" }).click();
    await page.getByText("Sua biblioteca está vazia").waitFor({ timeout: 15000 });
    if (!fs.existsSync(path.join(nuvem, "SyncGameSave", "devices.json"))) throw new Error("A configuração não registrou o PC na nuvem.");
    console.log("ok   primeiro uso configura uma pasta temporária no WebView Wails real");
    await page.getByRole("button", { name: "Adicionar jogo" }).first().click();
    const dialogo = page.getByRole("dialog", { name: "Adicionar jogo" });
    await dialogo.getByRole("button", { name: "Escolher executável…" }).click();
    await dialogo.getByLabel("Nome do jogo").waitFor();
    await dialogo.getByRole("button", { name: "Adicionar outra pasta de save…" }).click();
    for (const [nome, width, height] of [["cadastro-mobile", 390, 844], ["cadastro-intermediario", 768, 800], ["cadastro-desktop", 1280, 720]]) {
      await page.setViewportSize({ width, height });
      if (await page.evaluate(() => document.documentElement.scrollWidth > innerWidth || document.querySelector(".dialogo-corpo").scrollWidth > document.querySelector(".dialogo-corpo").clientWidth)) throw new Error(`Rolagem horizontal inesperada: ${nome}`);
      await page.screenshot({ path: path.join(pasta, `${nome}.png`) });
    }
    await dialogo.getByRole("button", { name: "Adicionar jogo", exact: true }).click();
    await page.getByRole("heading", { name: "Jogo Exemplo" }).waitFor({ timeout: 15000 });
    console.log("ok   cadastro manual escolhe executável e save e adiciona o jogo pela interface");
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
    if (info.ambiente !== "windows" || info.versao !== "2.1.0") throw new Error(`Binding ou ambiente inesperado: ${JSON.stringify(info)}`);
    console.log("ok   WebView Wails real reconhece o ambiente e chama o binding de versão");
    if (info.evento.id !== 17 || info.evento.estado !== "jogando") throw new Error(`Evento inesperado: ${JSON.stringify(info.evento)}`);
    if (info.jogos.length !== 2 || !info.jogos.some((j) => j.id === info.id && j.name === "Jogo de teste" && j.save_paths[0] === "<base>/save") || !info.jogos.some((j) => j.name === "Jogo Exemplo" && j.save_paths.length === 1)) throw new Error(`Biblioteca inesperada: ${JSON.stringify(info.jogos)}`);
    console.log("ok   o evento de jogo trafega pelo runtime Wails real");
    console.log("ok   adicionar e listar jogos usam a biblioteca SQLite temporária");
    await page.reload();
    await page.getByRole("heading", { name: "Jogo de teste" }).waitFor({ timeout: 15000 });
    console.log("ok   jogo salvo aparece na interface após recarregar");
    await page.getByRole("button", { name: /^Jogo Exemplo/ }).hover();
    await page.getByRole("button", { name: "Mais opções" }).click();
    await page.getByRole("button", { name: "Editar jogo" }).click();
    const editar = page.getByRole("dialog", { name: "Editar Jogo Exemplo" });
    await editar.getByLabel("Nome do jogo").fill("Jogo Editado");
    for (const [nome, width, height] of [["editar-mobile", 390, 844], ["editar-desktop", 1280, 720]]) {
      await page.setViewportSize({ width, height });
      await page.waitForTimeout(500);
      if (await page.evaluate(() => document.querySelector(".dialogo-corpo").scrollWidth > document.querySelector(".dialogo-corpo").clientWidth)) throw new Error(`Rolagem horizontal inesperada: ${nome}`);
      await page.screenshot({ path: path.join(pasta, `${nome}.png`) });
    }
    await editar.getByRole("button", { name: "Salvar alterações" }).click();
    await page.getByRole("heading", { name: "Jogo Editado" }).waitFor();
    console.log("ok   edição é salva e exibida pela interface Wails");
    await page.getByRole("button", { name: "Mais opções" }).click();
    await page.getByRole("button", { name: "Remover da biblioteca" }).click();
    await page.getByRole("button", { name: "Remover", exact: true }).click();
    await page.getByText(/continuam onde estavam/).waitFor();
    if (fs.readFileSync(save, "utf8") !== "progresso preservado") throw new Error("O save foi alterado ao remover o jogo.");
    console.log("ok   remoção preserva o save fora da biblioteca");

    // Jogar pela interface: o programa de teste (cmd.exe) fecha logo, e o save sobe para a nuvem.
    const cmd = path.join(process.env.SystemRoot || "C:\Windows", "System32", "cmd.exe");
    const idReal = await page.evaluate(async ([exe, saves]) => {
      const runtime = await import("/wails/runtime.js");
      return runtime.Call.ByName("main.App.AdicionarJogo", { name: "Jogo Real", exe, install_dir: null, catalog_id: null, save_paths: [saves] });
    }, [cmd, saveDir.replaceAll("\\", "/")]);
    await page.reload();
    await page.getByRole("heading", { name: "Jogo Real" }).waitFor({ timeout: 15000 });
    await page.getByRole("button", { name: /^Jogo Real/ }).hover();
    await page.getByRole("button", { name: "Jogar" }).first().click();
    const snaps = path.join(nuvem, "SyncGameSave", "games", "Jogo Real", "snapshots");
    for (let i = 0; i < 160 && !(fs.existsSync(snaps) && fs.readdirSync(snaps).length > 0); i++) await new Promise((r) => setTimeout(r, 500));
    if (!fs.existsSync(snaps) || fs.readdirSync(snaps).length === 0) throw new Error("Jogar pela interface não enviou o save para a nuvem.");
    console.log("ok   jogar pela interface envia o save para a pasta de sync");
    // Registro do sync e central rápida no app real.
    const logs = path.join(pasta, "dados", "logs");
    const lerLog = () => (fs.existsSync(logs) ? fs.readdirSync(logs).filter((f) => /^sync-[0-9-]+.log$/.test(f)).map((f) => fs.readFileSync(path.join(logs, f), "utf8")).join("") : "");
    for (let i = 0; i < 120 && !new RegExp(`jogo ${idReal}: finished`).test(lerLog()); i++) await new Promise((r) => setTimeout(r, 500));
    const textoLog = lerLog();
    if (!new RegExp(`jogo ${idReal}: abrindo`).test(textoLog) || !new RegExp(`jogo ${idReal}: finished`).test(textoLog)) throw new Error("O log de sync não registrou o jogo: " + textoLog);
    console.log("ok   jogar grava o resultado do sync no log diário");
    const central = await page.evaluate(async (id) => {
      const runtime = await import("/wails/runtime.js");
      return { avisos: await runtime.Call.ByName("main.App.Avisos"), voltar: await runtime.Call.ByName("main.App.VoltarAoJogo", id) };
    }, idReal);
    if (!Array.isArray(central.avisos) || central.voltar !== false) throw new Error("Central inesperada: " + JSON.stringify(central));
    await page.keyboard.press("F1");
    const dlgCentral = page.getByRole("dialog", { name: "Central rápida" });
    await dlgCentral.getByRole("region", { name: "Avisos recentes" }).waitFor();
    await page.screenshot({ path: path.join(pasta, "central-desktop.png") });
    await page.keyboard.press("Escape");
    console.log("ok   a central rápida abre no app real com os avisos e o foco sem jogo aberto devolve falso");

    await page.getByRole("button", { name: /^Jogo Real/ }).hover();
    await page.getByRole("button", { name: "Versões" }).first().click();
    await page.getByRole("button", { name: "Restaurar" }).waitFor({ timeout: 15000 });
    console.log("ok   o diálogo de versões lista o snapshot criado pelo backend Go");
    await page.keyboard.press("Escape");

    // Capa local: a interface a carrega por /capa/<arquivo>, só da pasta de arte.
    const arte = path.join(pasta, "dados", "art");
    fs.mkdirSync(arte, { recursive: true });
    const png = Buffer.from("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg==", "base64");
    fs.writeFileSync(path.join(arte, idReal + ".jpg"), png);
    const resp = await page.evaluate(async (id) => {
      const ok = await fetch("/capa/" + id + ".jpg");
      const fora = await fetch("/capa/../library.db");
      return { ok: ok.status, tipo: ok.headers.get("content-type"), foraStatus: fora.status, foraTipo: fora.headers.get("content-type") };
    }, idReal);
    if (resp.ok !== 200 || resp.foraTipo?.includes("octet-stream") || resp.foraTipo?.includes("sqlite")) throw new Error("Capa local inesperada: " + JSON.stringify(resp));
    console.log("ok   a capa local é servida de /capa/ e o banco de dados não");

    console.log(`Capturas: ${pasta}`);
  } finally {
    if (browser) await browser.close();
    app.kill();
  }
})().catch((err) => { console.error(err); process.exitCode = 1; });
