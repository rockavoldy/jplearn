<script setup>
import { onMounted } from 'vue'
import { RouterLink } from 'vue-router'
import { useLibraryStore } from '@/stores/library'

const library = useLibraryStore()

onMounted(() => library.fetchAll())

// Cards are a scannable summary, so show a compact hostname (e.g. "example.com")
// rather than the full URL. Falls back to the raw string if parsing fails.
function formatHost(url) {
  if (!url) return ''
  try {
    return new URL(url).hostname.replace(/^www\./, '')
  } catch {
    return url
  }
}

function formatDate(value) {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  return date.toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' })
}
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
              <span v-if="book.author" class="card-meta">
                <span class="card-meta-item">{{ book.author }}</span>
              </span>
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
            <RouterLink
              :to="{ name: 'article', params: { id: article.id } }"
              class="card-link"
              :title="article.url"
            >
              <span class="card-badge card-badge-web">WEB</span>
              <span class="card-title">{{ article.title }}</span>
              <span class="card-meta">
                <span class="card-meta-item" :title="article.url">
                  <svg
                    class="card-meta-icon"
                    viewBox="0 0 16 16"
                    aria-hidden="true"
                    focusable="false"
                  >
                    <path
                      fill="currentColor"
                      d="M8 0a8 8 0 1 0 0 16A8 8 0 0 0 8 0Zm5.9 5h-2.2a12.6 12.6 0 0 0-1-2.3A6.5 6.5 0 0 1 13.9 5ZM8 1.6c.6.9 1.1 2 1.4 3.4H6.6C6.9 3.6 7.4 2.5 8 1.6ZM1.6 8c0-.8.1-1.5.4-2.2h2.5a13 13 0 0 0 0 4.4H2c-.3-.7-.4-1.4-.4-2.2Zm.6 3.4h2.2c.3.8.6 1.6 1 2.3a6.5 6.5 0 0 1-3.2-2.3Zm2.2-6.8H2.2A6.5 6.5 0 0 1 5.4 2.3c-.4.7-.7 1.5-1 2.3ZM8 14.4c-.6-.9-1.1-2-1.4-3.4h2.8c-.3 1.4-.8 2.5-1.4 3.4Zm1.6-5H6.4a11.6 11.6 0 0 1 0-2.8h3.2a11.6 11.6 0 0 1 0 2.8Zm.6 4.9c.4-.7.7-1.5 1-2.3h2.2a6.5 6.5 0 0 1-3.2 2.3Zm1.4-3.9a13 13 0 0 0 0-4.4h2.5c.3.7.4 1.4.4 2.2s-.1 1.5-.4 2.2h-2.5Z"
                    />
                  </svg>
                  {{ formatHost(article.url) }}
                </span>
                <span class="card-meta-item">{{ formatDate(article.created_at) }}</span>
              </span>
            </RouterLink>
          </li>
        </ul>
      </div>
    </template>
  </section>
</template>
