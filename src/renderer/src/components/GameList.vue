<script setup lang="ts">
import type { Game } from '../../../types/types'

const games = await window.electron.ipcRenderer.invoke('game-list')
console.log(games)
function openGame(game: Game) {
  window.electron.ipcRenderer.send('open-game', game)
}
</script>

<template>
  <div class="grid grid-cols-3 gap-4">
    <div v-for="game in games" :key="game.id" class="flex flex-col gap-4 border border-gray-700 rounded-lg p-4">
      <h2 class="text-2xl font-bold text-center">{{ game.name }}</h2>

      <div class="flex justify-center">
        <button class="bg-blue-500 hover:bg-blue-700 text-white font-bold py-1 px-4 rounded" @click="openGame(game)">
          Abrir Jogo
        </button>
      </div>
    </div>
  </div>
</template>
