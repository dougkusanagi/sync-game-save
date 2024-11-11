import fs from "node:fs";
import yaml from "js-yaml";
import type { GameYaml } from "../types/types";

// Load the YAML file and parse it into a JavaScript object
export function loadSteamYamlFile(filePath) {
	try {
		const fileContent = fs.readFileSync(filePath, "utf8");
		return yaml.load(fileContent);
	} catch (error) {
		console.error("Error reading YAML file:", error);
		return null;
	}
}

// Function to search for game IDs by game name
export function searchGameIdsByName(yamlData, gameName) {
	const gameIds: string[] = [];

	for (const [gameId, gameData] of Object.entries(yamlData) as [
		string,
		GameYaml,
	][]) {
		if (gameData.installDir) {
			const gameNames = Object.keys(gameData.installDir);

			if (gameNames.includes(gameName)) {
				gameIds.push(gameId);
			}
		}
	}

	return gameIds;
}

// Function to get save files by game ID
export function getSaveFilesByGameId(yamlData, gameId) {
	const gameData = yamlData[gameId];
	const saveFiles: string[] = [];

	if (gameData?.files) {
		for (const [filePath, fileData] of Object.entries(gameData.files) as [
			string,
			{ tags?: string[] },
		][]) {
			if (fileData.tags?.includes("save")) {
				saveFiles.push(filePath);
			}
		}
	}

	return saveFiles;
}

export function getGameInfoById(
	yamlData: Record<string, GameYaml>,
	gameId: string,
): GameYaml | null {
	const gameData = yamlData[gameId];
	return gameData ? gameData : null;
}
