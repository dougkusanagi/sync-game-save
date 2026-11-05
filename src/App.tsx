import { useEffect, useState } from "react";
import { invoke } from "@tauri-apps/api/core";

type Sobre = { nome: string; versao: string };

export default function App() {
  const [sobre, setSobre] = useState<Sobre | null>(null);
  const [erro, setErro] = useState<string | null>(null);

  useEffect(() => {
    invoke<Sobre>("sobre")
      .then(setSobre)
      .catch(() => setErro("Não foi possível ler os dados do aplicativo."));
  }, []);

  return (
    <main className="sobre">
      <h1>Sync Game Save</h1>
      <p>Launcher de jogos que guarda seus saves com segurança na nuvem.</p>
      {sobre && (
        <p>
          {sobre.nome} · versão {sobre.versao}
        </p>
      )}
      {erro && <p role="alert">{erro}</p>}
    </main>
  );
}
