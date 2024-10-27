<script setup lang="ts">
// const ipcHandle = () => window.electron.ipcRenderer.send('ping')
function openGame() {
  window.electron.ipcRenderer.send('open-game')
}
</script>

<template>
  <button @click="openGame">
    Abrir Jogo
  </button>
</template>
