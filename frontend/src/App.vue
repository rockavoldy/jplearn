<script setup>
import { RouterLink, RouterView } from 'vue-router'
import DictionaryTooltip from '@/components/DictionaryTooltip.vue'
</script>

<template>
  <div class="app-shell">
    <header class="app-header">
      <RouterLink to="/" class="brand">
        <span class="brand-mark">読</span>
        <span class="brand-name">JP Reader</span>
      </RouterLink>
      <nav class="app-nav">
        <RouterLink to="/">Library</RouterLink>
        <RouterLink to="/upload">Upload EPUB</RouterLink>
      </nav>
    </header>

    <main class="app-main">
      <RouterView />
    </main>

    <!-- Global dictionary tooltip, positioned via Floating UI. -->
    <DictionaryTooltip />
  </div>
</template>
