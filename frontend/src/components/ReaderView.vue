<script setup>
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { useDictionaryStore } from '@/stores/dictionary'

const props = defineProps({
  html: { type: String, default: '' },
  title: { type: String, default: '' },
})

const dict = useDictionaryStore()
const wrapper = ref(null)

// Single delegated click listener on the reader wrapper. We read the lemma /
// reading from the clicked .vocab-word element's data attributes.
function onClick(event) {
  const target = event.target.closest('.vocab-word')
  if (!target) {
    return
  }
  const rect = target.getBoundingClientRect()
  dict.lookup(
    {
      lemma: target.dataset.lemma,
      reading: target.dataset.reading,
      surface: target.dataset.surface || target.textContent,
    },
    { x: rect.left + rect.width / 2, y: rect.top },
  )
}

// Close the tooltip when clicking outside any vocab word.
function onDocumentClick(event) {
  if (!event.target.closest('.vocab-word') && !event.target.closest('.dict-tooltip')) {
    dict.close()
  }
}

function onKeydown(event) {
  if (event.key === 'Escape') dict.close()
}

onMounted(() => {
  wrapper.value?.addEventListener('click', onClick)
  document.addEventListener('click', onDocumentClick)
  document.addEventListener('keydown', onKeydown)
})

onBeforeUnmount(() => {
  wrapper.value?.removeEventListener('click', onClick)
  document.removeEventListener('click', onDocumentClick)
  document.removeEventListener('keydown', onKeydown)
})
</script>

<template>
  <article class="reader">
    <h1 v-if="title" class="reader-title">{{ title }}</h1>
    <div ref="wrapper" class="reader-content" v-html="html"></div>
  </article>
</template>
