import { type ChildProcess, type ExecException, execFile } from 'node:child_process'
import { copyFile, mkdir } from "node:fs/promises";
import { statSync, existsSync } from 'node:fs'
import { listFiles, safeAwait } from './utils'
import { join, dirname, basename } from 'node:path'

async function copyDir(src: string, dest: string): Promise<void> {
  await mkdir(dest, { recursive: true });
  const entries = await import('node:fs').then(fs => fs.promises.readdir(src, { withFileTypes: true }));

  for (const entry of entries) {
    const srcPath = join(src, entry.name);
    const destPath = join(dest, entry.name);

    if (entry.isDirectory()) {
      await copyDir(srcPath, destPath);
    } else {
      await copyFile(srcPath, destPath);
    }
  }
}

export class GameService {
  private game_info: Game;
  private destiny_path: string;
  private child_process: ChildProcess | null;

  constructor(game_info: Game, destiny_path: string) {
    this.game_info = game_info;
    this.destiny_path = destiny_path;
    this.child_process = null;

    // Ensure destiny_path exists
    if (!existsSync(this.destiny_path)) {
      mkdir(this.destiny_path, { recursive: true })
        .catch(err => console.error('Failed to create destiny path:', err));
    }
  }

  exec(): void {
    try {
      this.child_process = execFile(this.game_info.exe);

      this.child_process.on("error", (error: ExecException) => {
        console.error(`Failed to start game: ${error.message}`);
      });

      this.child_process.on("close", async (code: number) => {
        console.log(`Game process ended with code ${code}`);
        if (code === 0) {
          await this.backupSaveFiles();
        }
      });
    } catch (error) {
      console.error('Failed to execute game:', error);
    }
  }

  private async backupSaveFiles(): Promise<void> {
    try {
      for (const save_file of this.game_info.save_files) {
        await this.backupFile(save_file);
      }
      console.log("Save files backup completed");
    } catch (error) {
      console.error("Failed to backup save files:", error);
    }
  }

  private async backupFile(save_path: SavePath): Promise<void> {
    try {
      if (!this.destiny_path) {
        throw new Error('Destination path is not set');
      }

      const paths: string[] = await this.resolvePaths(save_path);

      for (const path of paths) {
        const sourcePath = this.resolveSourcePath(path, save_path.same_dir);

        if (!existsSync(sourcePath)) {
          console.warn(`Source path does not exist: ${sourcePath}`);
          continue;
        }

        const filename = basename(path);
        const destPath = join(this.destiny_path, filename);

        const stats = statSync(sourcePath);

        if (stats.isDirectory()) {
          const [, error] = await safeAwait(
            copyDir(sourcePath, destPath)
          );
          if (error) {
            console.error(`Failed to copy directory: ${error.message}`);
          }
        } else {
          const [, error] = await safeAwait(
            copyFile(sourcePath, destPath)
          );
          if (error) {
            console.error(`Failed to copy file: ${error.message}`);
          }
        }
      }
    } catch (error) {
      console.error('Error in backupFile:', error);
    }
  }

  private async resolvePaths(save_path: SavePath): Promise<string[]> {
    if (save_path.type === "pattern") {
      const basePath = save_path.same_dir ? dirname(this.game_info.exe) : dirname(save_path.path);
      const pattern = save_path.same_dir ? save_path.path : basename(save_path.path);
      return this.listPathsPattern({ ...save_path, path: pattern }, basePath);
    }
    return [save_path.path];
  }

  private resolveSourcePath(path: string, same_dir: boolean): string {
    return same_dir ?
      join(dirname(this.game_info.exe), path)
      : path;
  }

  private async listPathsPattern(save_path: SavePath, cwd: string): Promise<string[]> {
    const [pattern_paths, error] = await safeAwait(
      listFiles(save_path.path, { cwd })
    );

    if (error) {
      console.error('Error listing pattern paths:', error);
      return [];
    }

    return pattern_paths || [];
  }
}

