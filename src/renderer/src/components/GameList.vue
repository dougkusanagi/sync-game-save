<script setup lang="ts">
import steamUrlBuilder from '../../../steam/steamUrlBuilder'
import type { Game } from '../../../types/types'

const games = await window.electron.ipcRenderer.invoke('game-list')

function openGame(game_info: Game) {
  window.electron.ipcRenderer.send('open-game', game_info)
}
</script>

<template>
  <div class="grid grid-cols-4 gap-4">
    <div v-for="game in games" :key="game.id" class="flex flex-col gap-4 border border-gray-700 rounded-lg p-4">
      <img :src="steamUrlBuilder.library(game.id)" alt="">

      <h2 class="text-lg font-bold text-center">{{ game.name }}</h2>

      <div class="flex justify-center">
        <button class="bg-indigo-500 hover:bg-indigo-700 text-white font-bold py-1 px-4 rounded"
          @click="openGame(game)">
          Abrir Jogo
        </button>
      </div>
    </div>
  </div>
</template>
