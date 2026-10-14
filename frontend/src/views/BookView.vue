<script setup>
import { onMounted, ref, watch } from 'vue'
import { RouterLink } from 'vue-router'
import api from '@/api/client'

const props = defineProps({ id: { type: String, required: true } })

const chapters = ref([])
const loading = ref(false)
const error = ref(null)

async function load(id) {
  loading.value = true
  error.value = null
  try {
    chapters.value = (await api.listChapters(id)) || []
  } catch (err) {
    error.value = err.message
  } finally {
    loading.value = false
  }
}

onMounted(() => load(props.id))
watch(() => props.id, (id) => load(id))
</script>

<template>
  <section class="toc">
    <h1>Table of Contents</h1>

    <div v-if="loading" class="state">Loading chapters…</div>
    <div v-else-if="error" class="state state-error">{{ error }}</div>
    <p v-else-if="!chapters.length" class="state">This book has no chapters.</p>

    <ol v-else class="toc-list">
      <li v-for="chapter in chapters" :key="chapter.id">
        <RouterLink :to="{ name: 'chapter', params: { id: chapter.id } }">
          <span class="toc-index">{{ chapter.chapter_index + 1 }}</span>
          <span class="toc-title">{{ chapter.title || `Chapter ${chapter.chapter_index + 1}` }}</span>
        </RouterLink>
      </li>
    </ol>
  </section>
</template>
