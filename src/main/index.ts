import { app, shell, BrowserWindow, ipcMain } from 'electron'
import { join } from 'node:path'
import { electronApp, optimizer, is } from '@electron-toolkit/utils'
import icon from '../../resources/icon.png?asset'
import { type ChildProcess, type ExecException, execFile } from 'node:child_process'
import { cp, copyFile } from "node:fs/promises";
import { PrismaClient } from '@prisma/client'
import { glob } from 'glob'
import { listFiles, safeAwait } from './utils'

const prisma = new PrismaClient()

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
      console.log(`child process exited with code ${code}`);

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

  async backupFile(save_path: SavePath) {
    if (save_path.type === "pattern") {
      console.log(`save_path.path=${save_path.path}`);
      console.log(`this.destiny_path=${this.destiny_path}`);

      // const path_pattern = join(this.destiny_path, save_path.path);
      // const [files, error] = await safeAwait(listFiles(save_path.path, { cwd: this.destiny_path }));
      const [files, error] = await glob(save_path.path, {
          cwd: this.destiny_path,
          nodir: false, // Only match files, not directories
          absolute: true,
          nocase: true,
          windowsPathsNoEscape: true // Handles Windows paths better
      });

      if (error) {
        console.error(error);
      } else {
        console.log(`files=${files}`);
      }

      console.log("============================================");
    }

    // if (save_path.type === "file") {
    //   await copyFile(save_path.path, save_path.path + ".bak");
    // } else if (save_path.type === "dir") {
    //   await cp(save_path.path, save_path.path + ".bak");
    // }
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
    const backup_dir = "C:/Users/dl_ag/Desktop/backup_saves";
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
