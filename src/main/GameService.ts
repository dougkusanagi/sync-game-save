import {
	type ChildProcess,
	type ExecException,
	execFile,
} from "node:child_process";
import { backupFiles } from "./utils";
import type { Game } from "../types/types";
import { join } from "node:path";

export class GameService {
	private game_info: Game;
	private destiny_path: string;
	private child_process: ChildProcess | null;

	constructor(game_info: Game, destiny_path: string) {
		this.game_info = game_info;
		this.destiny_path = destiny_path;
		this.child_process = null;
	}

	exec(): void {
		try {
			this.child_process = execFile(this.game_info.exe);

			this.child_process.on("error", (error: ExecException) => {
				console.error(`Failed to start game: ${error.message}`);
			});

			this.child_process.on("close", async (code: number) => {
				if (code === 0) {
					await backupFiles(
						this.game_info,
						join(this.destiny_path, this.game_info.name),
					);
				}
			});
		} catch (error) {
			console.error("Failed to execute game:", error);
		}
	}
}
