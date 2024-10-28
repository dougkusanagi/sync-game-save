import { app, shell, BrowserWindow, ipcMain } from 'electron'
import { electronApp, optimizer, is } from '@electron-toolkit/utils'
import icon from '../../resources/icon.png?asset'
import { type ChildProcess, type ExecException, execFile } from 'node:child_process'
import { copyFile, mkdir } from "node:fs/promises";
import { join, dirname, basename } from 'node:path'
import { statSync, existsSync } from 'node:fs'
import { listFiles, safeAwait } from './utils'

function createWindow(): void {
  // Create the browser window.
  const mainWindow = new BrowserWindow({
    width: 900,
    height: 670,
    show: false,
    autoHideMenuBar: true,
    ...(process.platform === 'linux' ? { icon } : {}),
    webPreferences: {
      preload: join(__dirname, '../preload/index.js'),
      sandbox: false
    }
  })

  mainWindow.on('ready-to-show', () => {
    mainWindow.show()
  })

  mainWindow.webContents.setWindowOpenHandler((details) => {
    shell.openExternal(details.url)
    return { action: 'deny' }
  })

  // HMR for renderer base on electron-vite cli.
  // Load the remote URL for development or the local html file for production.
  if (is.dev && process.env.ELECTRON_RENDERER_URL) {
    mainWindow.loadURL(process.env.ELECTRON_RENDERER_URL)
  } else {
    mainWindow.loadFile(join(__dirname, '../renderer/index.html'))
  }
}

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

interface Game {
  exe: string;
  name: string;
  save_files: SavePath[];
}

interface SavePath {
  path: string;
  same_dir: boolean;
  type: "dir" | "file" | "pattern";
}

class GameService {
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
    if (same_dir) {
      return join(dirname(this.game_info.exe), path);
    }
    return path;
  }

  private async listPathsPattern(save_path: SavePath, cwd: string): Promise<string[]> {
    try {
      const [pattern_paths, error] = await safeAwait(
        listFiles(save_path.path, { cwd })
      );

      if (error) {
        console.error('Error listing pattern paths:', error);
        return [];
      }

      return pattern_paths || [];
    } catch (error) {
      console.error('Error in listPathsPattern:', error);
      return [];
    }
  }
}

class GameRepository {
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

// This method will be called when Electron has finished
// initialization and is ready to create browser windows.
// Some APIs can only be used after this event occurs.
app.whenReady().then(() => {
  // Set app user model id for windows
  electronApp.setAppUserModelId('com.electron')

  // Default open or close DevTools by F12 in development
  // and ignore CommandOrControl + R in production.
  // see https://github.com/alex8088/electron-toolkit/tree/master/packages/utils
  app.on('browser-window-created', (_, window) => {
    optimizer.watchWindowShortcuts(window)
  })

  // IPC test
  ipcMain.on('ping', () => console.log('pong'))

  ipcMain.on('open-game', () => {
    console.log('Opening game...')

    const games_repository = new GameRepository();
    const games = games_repository.all();
    const destiny_path = "D:/fake_bkp_save_files";
    const game = new GameService(games[0], destiny_path);

    game.exec();
  })

  createWindow()

  app.on('activate', () => {
    // On macOS it's common to re-create a window in the app when the
    // dock icon is clicked and there are no other windows open.
    if (BrowserWindow.getAllWindows().length === 0) createWindow()
  })
})

// Quit when all windows are closed, except on macOS. There, it's common
// for applications and their menu bar to stay active until the user quits
// explicitly with Cmd + Q.
app.on('window-all-closed', () => {
  if (process.platform !== 'darwin') {
    app.quit()
  }
})

// In this file you can include the rest of your app"s specific main process
// code. You can also put them in separate files and require them here.
