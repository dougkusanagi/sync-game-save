import { glob } from "glob";
import { join } from "node:path";

export async function listFiles(
	_pattern: string,
	options: { cwd?: string; absolute?: boolean; nocase?: boolean } = {},
): Promise<string[]> {
	const { cwd = process.cwd(), absolute = true, nocase = false } = options;

	const userHome =
		process.platform === "win32" ? process.env.USERPROFILE : process.env.HOME;

	const pattern = userHome
		? _pattern.replace("%userprofile%", userHome)
		: _pattern;

	try {
		// Normalize backslashes to forward slashes for glob
		const normalizedPattern = pattern.replace(/\\/g, "/");

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

async function testGlob() {
	const pattern = "%userprofile%/AppData/LocalLow/Billy Basso/Animal Well/*";

	const files = await listFiles(pattern);
	console.log(files);
}

testGlob();
