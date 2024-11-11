<script setup lang="ts">
import { ref } from 'vue';
import GameList from './components/GameList.vue'

const destiny_path = ref("D:/fake_game_files/bkp_save_files")

async function chooseDestinyPath() {
  const path = await window.electron.ipcRenderer.invoke('dialog-directory:destiny-path');
  console.log(path)
  destiny_path.value = path
}
</script>

<template>
  <div class="h-screen bg-black text-gray-400 pt-12">
    <div class="container mx-auto">
      <div class="flex">
        <input class="bg-gray-900 text-white py-1 px-2 rounded w-full border border-gray-700" v-model="destiny_path"
          placeholder="Escolha a pasta de destino" />

        <button class="bg-indigo-500 hover:bg-indigo-700 text-white font-bold py-2 px-4 rounded ml-2 w-48"
          @click="chooseDestinyPath">
          Escolher Pasta
        </button>
      </div>

      <div class="mt-8">
        <Suspense>
          <!-- component with nested async dependencies -->
          <GameList />
          <!-- loading state via #fallback slot -->
          <template #fallback>
            Loading...
          </template>
        </Suspense>
      </div>
    </div>
  </div>
</template>
