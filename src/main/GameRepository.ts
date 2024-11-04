export class GameRepository {
  all(): Game[] {
    const games: Game[] = [
      {
        name: "Aggelos",
        exe: "C:/Games/Aggelos.v1.065/LAUNCHER_x64.exe",
        save_files: [
          { path: "curseur.ini", same_dir: true, type: "file" },
          { path: "sauvegarde*.ini", same_dir: true, type: "pattern" },
        ],
      }
    ]

    return games;
  }
}
