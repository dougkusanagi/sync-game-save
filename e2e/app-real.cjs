// Abre o app Tauri de verdade (dados em pasta temporária) e controla a janela pelo WebView2 (CDP).
// Uso: (cd src-tauri && cargo build) && bun run e2e:app   (só Windows; precisa de `bunx playwright install chromium`)
const { chromium } = require("playwright");
const { spawn, execSync } = require("node:child_process");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const http = require("node:http");

const PROJ = path.join(__dirname, "..");
const OUT = path.join(os.tmpdir(), "sgs-app-real-capturas");
fs.mkdirSync(OUT, { recursive: true });
const tmp = fs.mkdtempSync(path.join(os.tmpdir(), "sgs-real-"));
const nuvem = path.join(tmp, "nuvem");
const saves = path.join(tmp, "saves", "Jogo");
fs.mkdirSync(nuvem, { recursive: true });
fs.mkdirSync(saves, { recursive: true });
fs.writeFileSync(path.join(saves, "slot1.sav"), "progresso");

const noAr = (url) => new Promise((ok) => http.get(url, (r) => (r.resume(), ok(true))).on("error", () => ok(false)));
async function esperar(url, n = 80) {
  for (let i = 0; i < n; i++) {
    if (await noAr(url)) return true;
    await new Promise((r) => setTimeout(r, 500));
  }
  return false;
}

(async () => {
  const resultado = [];
  const ok = (m) => (resultado.push("ok   " + m), console.log("ok   " + m));
  const falha = (m) => (resultado.push("FALHOU " + m), console.log("FALHOU " + m));

  const vite = spawn("bunx", ["vite", "--port", "1420", "--strictPort"], { cwd: PROJ, shell: true, stdio: "ignore" });
  if (!(await esperar("http://localhost:1420/"))) throw new Error("vite não subiu");

  const app = spawn(path.join(PROJ, "src-tauri/target/debug/sync-game-save.exe"), [], {
    env: { ...process.env, SGS_DATA_DIR: path.join(tmp, "dados"), WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS: "--remote-debugging-port=9222", WEBVIEW2_USER_DATA_FOLDER: path.join(tmp, "wv2") },
    stdio: "ignore",
  });
  try {
    if (!(await esperar("http://127.0.0.1:9222/json/version"))) throw new Error("o depurador do WebView2 não abriu");
    const browser = await chromium.connectOverCDP("http://127.0.0.1:9222");
    const page = browser.contexts()[0].pages()[0];
    page.setDefaultTimeout(8000);
    await page.waitForLoadState("domcontentloaded");
    const invoke = (cmd, args) => page.evaluate(([c, a]) => window.__TAURI_INTERNALS__.invoke(c, a), [cmd, args]);

    // 1. Primeiro uso (banco novo, sem pasta de sync).
    await page.getByText("Vamos guardar seus saves com segurança").waitFor({ timeout: 30000 }).catch(async (e) => {
      const texto = await page.locator("body").innerText().catch(() => "");
      throw new Error(`${e.message.split(/\r?\n/)[0]} (página: ${page.url()}, texto: ${texto.slice(0, 200)})`);
    });
    await page.screenshot({ path: path.join(OUT, "1-primeiro-uso.png") });
    ok("app real abre no assistente de primeiro uso com o banco novo");

    // 2. Comandos reais do Rust: validar e definir a pasta.
    const verif = await invoke("verificar_pasta_sync", { pasta: nuvem });
    verif.writable && !verif.has_data ? ok("verificar_pasta_sync (pasta gravável, sem dados de outro PC)") : falha("verificar_pasta_sync " + JSON.stringify(verif));
    const ruim = await invoke("definir_pasta_sync", { pasta: path.join(tmp, "nao-existe") }).then(() => null, (e) => String(e.message));
    typeof ruim === "string" && ruim.includes("não existe") ? ok("pasta inexistente recusada com mensagem clara") : falha("pasta inexistente: " + ruim);
    await invoke("definir_pasta_sync", { pasta: nuvem });
    await invoke("definir_nome_pc", { nome: "PC-TESTE" });
    fs.existsSync(path.join(nuvem, "SyncGameSave", "devices.json")) ? ok("devices.json criado na pasta de sync") : falha("devices.json ausente");

    // 3. Reconhecer o jogo real sem catálogo (cai no modo manual).
    const rec = await invoke("reconhecer_pasta", { pasta: "E:\\Games\\RogueLegacy" });
    rec.name === "RogueLegacy" && rec.exe && rec.exe.toLowerCase().endsWith("roguelegacy.exe") ? ok("reconhecer_pasta sem catálogo acha nome e executável do Rogue Legacy real") : falha("reconhecer_pasta " + JSON.stringify(rec));

    // 4. Adicionar um jogo de teste (executável que sai na hora) e jogar pela interface.
    const id = await invoke("adicionar_jogo", { jogo: { name: "Jogo Teste", exe: "cmd", install_dir: null, catalog_id: null, save_paths: [saves.replaceAll("\\", "/")] } });
    await page.reload();
    await page.getByRole("heading", { name: "Jogo Teste" }).waitFor();
    await page.screenshot({ path: path.join(OUT, "2-biblioteca.png") });
    ok("jogo adicionado pelo backend aparece na biblioteca");

    await page.getByRole("button", { name: "Jogar" }).first().click();
    // O executável de teste termina rápido: esperamos o snapshot persistido, que prova o sync.
    const snaps = path.join(nuvem, "SyncGameSave", "games", "Jogo Teste", "snapshots");
    for (let i = 0; i < 80 && (!fs.existsSync(snaps) || fs.readdirSync(snaps).length === 0); i++) await new Promise((r) => setTimeout(r, 500));
    await new Promise((r) => setTimeout(r, 20000));
    await page.screenshot({ path: path.join(OUT, "3-depois-de-jogar.png") });
    fs.existsSync(snaps) && fs.readdirSync(snaps).length === 1 ? ok("jogar pela interface criou 1 snapshot na pasta de sync") : falha("snapshot ausente em " + snaps);

    // 5. Versões e tempo de jogo.
    await page.getByRole("button", { name: "Versões" }).first().click();
    await page.getByRole("button", { name: "Restaurar" }).waitFor();
    await page.screenshot({ path: path.join(OUT, "4-versoes.png") });
    ok("diálogo de versões lista o snapshot do backend real");
    await page.keyboard.press("Escape");

    // 6. Arte de um jogo sem catálogo: busca na loja da Steam pelo nome e a janela consegue mostrar o arquivo local.
    const rl = await invoke("adicionar_jogo", { jogo: { name: "RogueLegacy", exe: "cmd", install_dir: null, catalog_id: null, save_paths: [] } });
    const hero = await invoke("arte_do_jogo", { id: rl, tipo: "hero" });
    hero ? ok("arte de fundo baixada pela busca na loja da Steam") : falha("arte_do_jogo hero voltou vazia");
    await page.reload();
    await page.getByRole("button", { name: /^RogueLegacy./ }).hover();
    const carregou = await page
      .waitForFunction(() => [...document.querySelectorAll(".cap-art img")].some((i) => i.complete && i.naturalWidth > 0) && getComputedStyle(document.querySelector(".fundo-arte")).backgroundImage.includes("asset"), null, { timeout: 15000 })
      .then(() => true, () => false);
    await page.screenshot({ path: path.join(OUT, "5-arte.png") });
    carregou ? ok("a janela mostra a capa e o fundo vindos do cache local") : falha("as imagens locais não carregaram na janela (protocolo asset)");

    await browser.close();
  } finally {
    app.kill();
    try { execSync("taskkill /F /T /PID " + vite.pid, { stdio: "ignore" }); } catch {}
    try { execSync("taskkill /F /IM sync-game-save.exe", { stdio: "ignore" }); } catch {}
  }
  console.log("\n" + (resultado.some((r) => r.startsWith("FALHOU")) ? "HÁ FALHAS" : "tudo certo") + "  (dados em " + tmp + ")");
})().catch((e) => {
  console.error("ERRO:", e.message);
  try { execSync("taskkill /F /IM sync-game-save.exe", { stdio: "ignore" }); } catch {}
  process.exit(1);
});
