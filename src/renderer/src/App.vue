<script setup lang="ts">
import GameList from './components/GameList.vue'
</script>

<template>
  <div class="h-screen bg-black text-gray-400 pt-8">
    <div class="container mx-auto">
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
</template>
