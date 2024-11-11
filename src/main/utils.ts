import { glob } from "glob";
import { basename, dirname, join } from "node:path";
import { copyFile, mkdir, lstat } from "node:fs/promises";
import type { Game, SavePath } from "../types/types";

export async function listPathsPattern(
	save_path: SavePath,
	cwd?: string,
): Promise<string[]> {
	const [pattern_paths, error] = await safeAwait(
		listFiles(save_path.path, { cwd }),
	);

	if (error) {
		console.error("Error listing pattern paths:", error);
		return [];
	}

	return pattern_paths || [];
}

async function isDir(path: string): Promise<boolean> {
	return await lstat(path).then((stats) => stats.isDirectory());
}

async function isFile(path: string): Promise<boolean> {
	return await lstat(path).then((stats) => stats.isFile());
}

// async function isDir(path: string): Promise<boolean> {
// 	return new Promise((resolve, reject) => {
// 		lstat(path, (err, stats) => {
// 			if (err) {
// 				reject(err);
// 			} else {
// 				resolve(stats.isDirectory());
// 			}
// 		});
// 	});
// }

// async function isFile(path: string): Promise<boolean> {
// 	return new Promise((resolve, reject) => {
// 		lstat(path, (err, stats) => {
// 			if (err) {
// 				reject(err);
// 			} else {
// 				resolve(stats.isFile());
// 			}
// 		});
// 	});
// }

export async function backupFiles(game_info: Game, destiny_path: string) {
	for (const save_path of game_info.save_files) {
		const [paths, error] = await safeAwait(
			listFiles(save_path.path, { cwd: dirname(game_info.exe) }),
		);

		if (error) {
			console.error("Error listing pattern paths:", error);
		}

		if (!paths) {
			console.error("No paths found for", save_path.path);
			continue;
		}

		await mkdir(destiny_path, { recursive: true });

		for (const path of paths) {
			if (await isDir(path)) {
				await copyDir(path, join(destiny_path, basename(path)));
				continue;
			}

			if (!(await isFile(path))) {
				continue;
			}

			const pathfile = basename(path);
			await copyFile(path, join(destiny_path, pathfile));
		}
	}
}

export async function copyDir(src: string, dest: string): Promise<void> {
	await mkdir(dest, { recursive: true });

	const entries = await import("node:fs").then((fs) =>
		fs.promises.readdir(src, { withFileTypes: true }),
	);

	for (const entry of entries) {
		const srcPath = join(src, entry.name);
		const destPath = join(dest, entry.name);

		if (entry.isDirectory()) {
			await copyDir(srcPath, join(dest, entry.name));
		} else {
			await copyFile(srcPath, destPath);
		}
	}
}

export function replaceEnvVariables(_pattern: string): string {
	const envMap: { [key: string]: string | undefined } = {
		"%userprofile%": process.env.USERPROFILE,
		"%appdata%": process.env.APPDATA,
		"%localappdata%": process.env.LOCALAPPDATA,
		"%programfiles%": process.env.PROGRAMFILES,
		"%temp%": process.env.TEMP,
	};

	let pattern = _pattern;

	for (const key in envMap) {
		if (envMap[key]) {
			pattern = pattern.replace(key, envMap[key]);
		}
	}

	return pattern;
}

export async function listFiles(
	pattern: string,
	options: { cwd?: string; absolute?: boolean; nocase?: boolean } = {},
): Promise<string[]> {
	const { cwd = process.cwd(), absolute = true, nocase = false } = options;

	try {
		// Normalize backslashes to forward slashes for glob
		const normalizedPattern = replaceEnvVariables(pattern).replace(/\\/g, "/");

		const files = await glob(normalizedPattern, {
			cwd,
			nodir: false, // Only match files, not directories
			absolute,
			nocase,
			windowsPathsNoEscape: true, // Handles Windows paths better
		});

		// If absolute paths weren't requested but cwd was provided,
		// prepend the cwd to make paths relative to it
		if (!absolute && cwd !== process.cwd()) {
			return files.map((file) => join(cwd, file));
		}

		return files;
	} catch (error: unknown) {
		throw new Error(
			`Error listing files: ${error instanceof Error ? error.message : String(error)}`,
		);
	}
}

export async function safeAwait<T, E = Error>(
	promise: Promise<T>,
): Promise<[T, null] | [null, E]> {
	try {
		const result = await promise;
		return [result, null];
	} catch (error) {
		return [null, error as E];
	}
}
