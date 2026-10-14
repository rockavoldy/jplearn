<script setup>
import { computed, onBeforeUnmount, ref, watch, nextTick } from 'vue'
import { computePosition, offset, flip, shift, autoUpdate } from '@floating-ui/dom'
import { useDictionaryStore } from '@/stores/dictionary'

const dict = useDictionaryStore()
const tooltipEl = ref(null)

// Virtual element positioned at the clicked word's coordinates.
const virtualAnchor = computed(() => ({
  getBoundingClientRect() {
    const a = dict.anchor
    if (!a) return { x: 0, y: 0, top: 0, left: 0, width: 0, height: 0, right: 0, bottom: 0 }
    return {
      x: a.x,
      y: a.y,
      top: a.y,
      left: a.x,
      width: 0,
      height: 0,
      right: a.x,
      bottom: a.y,
    }
  },
}))

let cleanup = null

function teardown() {
  if (cleanup) {
    cleanup()
    cleanup = null
  }
}

async function position() {
  if (!dict.open) return
  await nextTick()
  if (!tooltipEl.value) return
  teardown()
  cleanup = autoUpdate(virtualAnchor.value, tooltipEl.value, () => {
    computePosition(virtualAnchor.value, tooltipEl.value, {
      // The tooltip is `position: fixed`, so coordinates must be resolved in
      // viewport space. Without this, Floating UI defaults to `absolute` and
      // adds the page scroll offset, making the popup drift up as you scroll.
      strategy: 'fixed',
      placement: 'top',
      middleware: [offset(10), flip({ padding: 8 }), shift({ padding: 8 })],
    }).then(({ x, y }) => {
      if (tooltipEl.value) {
        tooltipEl.value.style.left = `${x}px`
        tooltipEl.value.style.top = `${y}px`
      }
    })
  })
}

watch(
  () => [dict.open, dict.anchor],
  ([open]) => {
    if (open) {
      position()
    } else {
      teardown()
    }
  },
  { deep: true },
)

onBeforeUnmount(teardown)
</script>

<template>
  <div v-if="dict.open" ref="tooltipEl" class="dict-tooltip" role="dialog">
    <button class="dict-close" aria-label="Close" @click="dict.close()">×</button>

    <header class="dict-head">
      <span class="dict-word">{{ dict.surface || dict.lemma }}</span>
      <span v-if="dict.reading" class="dict-reading">{{ dict.reading }}</span>
    </header>

    <div v-if="dict.loading" class="dict-loading">Searching…</div>

    <div v-else-if="dict.error" class="dict-error">{{ dict.error }}</div>

    <div v-else-if="dict.entries.length === 0" class="dict-empty">
      No dictionary entry found for「{{ dict.lemma }}」.
    </div>

    <ol v-else class="dict-entries">
      <li v-for="entry in dict.entries" :key="entry.ent_seq" class="dict-entry">
        <div class="dict-entry-forms">
          <span v-if="entry.kanji && entry.kanji.length">{{ entry.kanji.join('、') }}</span>
          <span v-if="entry.readings && entry.readings.length" class="dict-entry-readings">
            {{ entry.readings.join('、') }}
          </span>
        </div>
        <ol class="dict-senses">
          <li v-for="(sense, i) in entry.meanings" :key="i" class="dict-sense">
            <span v-if="sense.pos && sense.pos.length" class="dict-pos">
              {{ sense.pos.join(', ') }}
            </span>
            <span class="dict-gloss">{{ sense.glosses.join('; ') }}</span>
          </li>
        </ol>
      </li>
    </ol>
  </div>
</template>
