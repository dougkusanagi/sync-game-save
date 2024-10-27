class GameManager {
  private childProcess: ChildProcess | null = null;

  async createGame(name: string, exePath: string, backupPath: string, savePaths: { path: string; type: 'dir' | 'file' }[]) {
    try {
      const game = await prisma.game.create({
        data: {
          name,
          exePath,
          backupPath,
          saveFiles: {
            create: savePaths.map(save => ({
              path: save.path,
              type: save.type
            }))
          }
        },
        include: {
          saveFiles: true
        }
      })
      return game
    } catch (error) {
      console.error('Error creating game:', error)
      throw error
    }
  }

  async executeGame(gameId: number) {
    try {
      const game = await prisma.game.findUnique({
        where: { id: gameId },
        include: { saveFiles: true }
      })

      if (!game) {
        throw new Error('Game not found')
      }

      this.childProcess = execFile(game.exePath)

      this.childProcess.on("error", (error: ExecException) => {
        console.error(`Error executing game: ${error.message}`)
        this.logBackupHistory(gameId, '', false, error.message)
      })

      this.childProcess.on("close", async (code: number) => {
        console.log(`Game process exited with code ${code}`)
        await this.backupSaveFiles(game)
      })
    } catch (error) {
      console.error('Error executing game:', error)
      throw error
    }
  }

  private async backupSaveFiles(game: Game) {
    for (const saveFile of game.saveFiles) {
      try {
        const backupPath = join(game.backupPath, `${saveFile.id}_${Date.now()}`)

        if (saveFile.type === 'file') {
          await copyFile(saveFile.path, backupPath)
        } else {
          await cp(saveFile.path, backupPath, { recursive: true })
        }

        await this.logBackupHistory(game.id, saveFile.path, true)
      } catch (error) {
        console.error(`Error backing up ${saveFile.path}:`, error)
        await this.logBackupHistory(game.id, saveFile.path, false, error.message)
      }
    }
  }

  private async logBackupHistory(gameId: number, path: string, success: boolean, error?: string) {
    try {
      await prisma.backupHistory.create({
        data: {
          gameId,
          path,
          success,
          error
        }
      })
    } catch (error) {
      console.error('Error logging backup history:', error)
    }
  }

  async getAllGames() {
    return prisma.game.findMany({
      include: {
        saveFiles: true
      }
    })
  }

  async getBackupHistory(gameId: number) {
    return prisma.backupHistory.findMany({
      where: {
        gameId
      },
      orderBy: {
        createdAt: 'desc'
      }
    })
  }
}
