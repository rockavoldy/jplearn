<script setup>
import { onMounted } from 'vue'
import { RouterLink } from 'vue-router'
import { useLibraryStore } from '@/stores/library'

const library = useLibraryStore()

onMounted(() => library.fetchAll())
</script>

<template>
  <section class="library">
    <div v-if="library.loading" class="state">Loading library…</div>
    <div v-else-if="library.error" class="state state-error">{{ library.error }}</div>

    <template v-else>
      <div class="library-section">
        <h2>Books</h2>
        <p v-if="!library.books.length" class="state">
          No books yet. <RouterLink to="/upload">Upload an EPUB</RouterLink> to get started.
        </p>
        <ul v-else class="card-grid">
          <li v-for="book in library.books" :key="book.id" class="card">
            <RouterLink :to="{ name: 'book', params: { id: book.id } }" class="card-link">
              <span class="card-badge">EPUB</span>
              <span class="card-title">{{ book.title }}</span>
              <span v-if="book.author" class="card-sub">{{ book.author }}</span>
            </RouterLink>
          </li>
        </ul>
      </div>

      <div class="library-section">
        <h2>Saved Articles</h2>
        <p v-if="!library.articles.length" class="state">
          No articles yet. Use the browser extension to save a Japanese web page.
        </p>
        <ul v-else class="card-grid">
          <li v-for="article in library.articles" :key="article.id" class="card">
            <RouterLink :to="{ name: 'article', params: { id: article.id } }" class="card-link">
              <span class="card-badge card-badge-web">WEB</span>
              <span class="card-title">{{ article.title }}</span>
              <span class="card-sub">{{ article.url }}</span>
            </RouterLink>
          </li>
        </ul>
      </div>
    </template>
  </section>
</template>
