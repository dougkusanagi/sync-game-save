import { expect, test } from "bun:test";
import { urlArteWails } from "./arteWails";

test("converte o nome do arquivo de arte local em URL segura", () => {
  expect(urlArteWails("C:\\Users\\alguém\\art\\capa #1.png")).toBe("/capa/capa%20%231.png");
  expect(urlArteWails("/home/pessoa/art/ação.png")).toBe("/capa/a%C3%A7%C3%A3o.png");
  expect(urlArteWails("/art/..")).toBeNull();
  expect(urlArteWails("/art/")).toBeNull();
});
