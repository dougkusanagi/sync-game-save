import { app, shell, BrowserWindow, ipcMain } from 'electron'
import { join, dirname, basename } from 'node:path'
import { statSync } from 'node:fs'
import { electronApp, optimizer, is } from '@electron-toolkit/utils'
import icon from '../../resources/icon.png?asset'
import { type ChildProcess, type ExecException, execFile } from 'node:child_process'
import { cp, copyFile } from "node:fs/promises";
import { glob } from 'glob'
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

async function copyDirectory(source, destination) {
  try {
    await cp(source, destination, { recursive: true });
    console.log('Directory copied successfully!');
  } catch (error) {
    console.error('Error copying directory:', error);
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
  game_info: Game;
  destiny_path: string;
  child_process: ChildProcess | null;

  constructor(game_info: Game, destiny_path: string) {
    this.game_info = game_info;
    this.destiny_path = destiny_path;
    this.child_process = null;
  }

  exec() {
    this.child_process = execFile(this.game_info.exe);

    // this.child_process.stdout?.on("data", (data: string) => {
    //   console.log(data);
    // });

    // this.child_process.stderr?.on("data", (data: string) => {
    //   console.log(`stderr: ${data}`);
    // });

    // Handle errors
    this.child_process.on("error", (error: ExecException) => {
      console.log(`error: ${error.message}`);
    });

    // Handle process completion
    this.child_process.on("close", (code: number) => {
      // console.log(`child process exited with code ${code}`);

      this.backupSaveFiles();

      if (code === 0) {
        // console.log('success');
      } else {
        console.log(`Process exited with code ${code}`);
      }
    });
  }

  async backupSaveFiles() {
    for (const save_file of this.game_info.save_files) {
      await this.backupFile(save_file);
    }

    console.log("Save files backed up");
  }

  // async backupFile(save_path: SavePath) {
  //   let paths = [];

  //   if (save_path.type === "pattern") {
  //     paths.push(...this.listPathsPattern(save_path))
  //   } else {
  //     paths.push(save_path.path)
  //   }

  //   for (const path of paths) {
  //     if (save_path.type === "file") {
  //       await copyFile(save_path.path, this.destiny_path);
  //     } else if (save_path.type === "dir") {
  //       await cp(save_path.path, this.destiny_path);
  //     }
  //   }
  // }

  async backupFile(save_path: SavePath): Promise<void> {
    if (!this.destiny_path) {
        throw new Error('Destination path is not set');
    }

    const paths: string[] = [];
    
    if (save_path.type === "pattern") {
        const patternPaths = await this.listPathsPattern(save_path);

        if (!patternPaths.length) {
            console.warn('No files found matching pattern:', save_path.path);
            return;
        }

        paths.push(...patternPaths);
    } else {
        paths.push(save_path.path);
    }

    // console.log("\ntype")
    // console.log(save_path.type);
    // console.log("\npaths");
    // console.log(paths);
    // console.log();
    // return;

    for (const path of paths) {
      // switch (save_path.type) {
      //     case "file":
      //         await copyFile(path, this.destiny_path);
      //         break;
      //     case "dir":
      //         await cp(path, this.destiny_path, { recursive: true });
      //         break;
      //     default:
      //         throw new Error(`Unsupported save path type: ${save_path.type}`);
      // }

      const filename = basename(path)

      if (statSync(path).isDirectory()) {
        const [res, error] = await safeAwait(
          cp(
            path,
            join(this.destiny_path, filename),
            { recursive: true })
        );

        if (error) {
          console.warn(error);
        }

        return;
      }

      const [res, error] = await safeAwait(copyFile(path, this.destiny_path));

      if (error) {
        console.warn(error);
      }
    }
}

  async listPathsPattern(save_path: SavePath) {
    const paths: string[] = [];
    const relative_path = dirname(save_path.path)
    const cwd = save_path.same_dir ? dirname(save_path.exe) : relative_path;
    const [pattern_paths, error] = await safeAwait(listFiles(save_path.path, { cwd }));

    if (error) {
        console.error(error);
    } else if (pattern_paths) {
        paths.push(...pattern_paths);
    }

    return paths;
  }

  // async uploadFile(path: string) {
  //   const file = await uploadFile(path);
  //   console.log(file);
  // }

  // async restoreSaveFiles() {
  //   const save_files = this.save_files;
  //   for (const save_file of save_files) {
  //     const file = await uploadFile(save_file.path);
  //     console.log(file);
  //   }
  // }
}

class GameRepository {
  all(): Game[] {
    const games: Game[] = [
      {
        name: "Aggelos",
        exe: "C:/Windows/System32/notepad.exe",
        save_files: [
          { path: "D:/Douglas/dev/fake_save_files/curseur.ini", same_dir: false, type: "file" },
          { path: "D:/Douglas/dev/fake_save_files/de", same_dir: false, type: "dir" },
          { path: "D:/Douglas/dev/fake_save_files/sauvegarde*.ini", same_dir: false, type: "pattern" },
          { path: "D:/Douglas/dev/fake_save_files/ab*", same_dir: false, type: "pattern" },
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
    const backup_dir = "D:/Douglas/dev/fake_bkp_save";
    const game = new GameService(games[0], backup_dir);

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
