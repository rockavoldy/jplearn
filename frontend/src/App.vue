<script setup>
import { RouterLink, RouterView } from 'vue-router'
import DictionaryTooltip from '@/components/DictionaryTooltip.vue'

// EPUB upload is turned off for now; the nav link and route are hidden unless
// explicitly enabled at build time via VITE_ENABLE_EPUB_UPLOAD=true.
const epubUploadEnabled = import.meta.env.VITE_ENABLE_EPUB_UPLOAD === 'true'
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
        <RouterLink v-if="epubUploadEnabled" to="/upload">Upload EPUB</RouterLink>
      </nav>
    </header>

    <main class="app-main">
      <RouterView />
    </main>

    <!-- Global dictionary tooltip, positioned via Floating UI. -->
    <DictionaryTooltip />
  </div>
</template>
