<script setup>
import { onMounted, watch } from 'vue'
import { useReaderStore } from '@/stores/reader'
import ReaderView from '@/components/ReaderView.vue'

const props = defineProps({ id: { type: String, required: true } })
const reader = useReaderStore()

onMounted(() => reader.loadChapter(props.id))
watch(() => props.id, (id) => reader.loadChapter(id))
</script>

<template>
  <section>
    <div v-if="reader.loading" class="state">Loading chapter…</div>
    <div v-else-if="reader.error" class="state state-error">{{ reader.error }}</div>
    <ReaderView v-else :title="reader.title" :html="reader.html" />
  </section>
</template>
