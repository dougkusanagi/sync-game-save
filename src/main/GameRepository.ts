import type { Game } from "../types/types";

export default class GameRepository {
	all(): Game[] {
		const games: Game[] = [
			{
				id: 1,
				name: "Animal Well",
				exe: "C:/Games/ANIMAL WELL/Animal Well.exe",
				save_files: [
					{
						path: "%userprofile%/AppData/LocalLow/Billy Basso/Animal Well/*",
						same_dir: false,
					},
				],
			},
			{
				id: 2,
				name: "Aggelos",
				exe: "C:/Games/Aggelos/LAUNCHER_x64.exe",
				save_files: [
					{
						path: "*.ini",
						same_dir: true,
					},
				],
			},
		];

		return games;
	}
}
