import { glob } from 'glob'
import { join } from 'node:path'

export async function listFiles(
  pattern: string,
  options: { cwd?: string; absolute?: boolean; ignoreCase?: boolean } = {}
): Promise<string[]> {
    const {
        cwd = process.cwd(),
        absolute = true,
        ignoreCase = false
    } = options;

    try {
        // Normalize backslashes to forward slashes for glob
        const normalizedPattern = pattern.replace(/\\/g, '/');

        const files = await glob(normalizedPattern, {
            cwd,
            nodir: false, // Only match files, not directories
            absolute,
            nocase: ignoreCase,
            windowsPathsNoEscape: true // Handles Windows paths better
        });

        // If absolute paths weren't requested but cwd was provided,
        // prepend the cwd to make paths relative to it
        if (!absolute && cwd !== process.cwd()) {
            return files.map(file => join(cwd, file));
        }

        return files;
    } catch (error: unknown) {
        throw new Error(`Error listing files: ${error instanceof Error ? error.message : String(error)}`);
    }
}

export async function safeAwait<T, E=Error>(promise: Promise<T>): Promise<[T, null] | [null, E]>
{
    try {
        const result = await promise;
        return [result, null]
    } catch (error) {
        return [null, error as E]
    }
}
