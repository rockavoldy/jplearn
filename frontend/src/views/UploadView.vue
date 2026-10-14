<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useLibraryStore } from '@/stores/library'

const MAX_BYTES = 50 * 1024 * 1024
const library = useLibraryStore()
const router = useRouter()

const file = ref(null)
const uploading = ref(false)
const error = ref(null)
const progressLabel = ref('')

function onFileChange(event) {
  error.value = null
  const selected = event.target.files?.[0] || null
  if (selected && selected.size > MAX_BYTES) {
    error.value = 'File exceeds the 50MB EPUB limit.'
    file.value = null
    return
  }
  file.value = selected
}

async function submit() {
  if (!file.value) return
  uploading.value = true
  error.value = null
  progressLabel.value = 'Uploading and processing… this may take a moment.'
  try {
    const book = await library.uploadEPUB(file.value)
    router.push({ name: 'book', params: { id: book.id } })
  } catch (err) {
    error.value = err.message
  } finally {
    uploading.value = false
    progressLabel.value = ''
  }
}
</script>

<template>
  <section class="upload">
    <h1>Upload an EPUB</h1>
    <p class="upload-hint">
      Maximum file size: 50MB. Images are offloaded to object storage and text is
      annotated with furigana automatically.
    </p>

    <label class="dropzone">
      <input type="file" accept=".epub,application/epub+zip" @change="onFileChange" />
      <span v-if="!file">Choose an .epub file</span>
      <span v-else>{{ file.name }} ({{ (file.size / 1024 / 1024).toFixed(2) }} MB)</span>
    </label>

    <p v-if="error" class="state-error">{{ error }}</p>

    <button class="btn" :disabled="!file || uploading" @click="submit">
      {{ uploading ? 'Processing…' : 'Upload & Process' }}
    </button>
    <p v-if="progressLabel" class="state">{{ progressLabel }}</p>
  </section>
</template>
