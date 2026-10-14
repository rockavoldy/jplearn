import { defineStore } from 'pinia'
import api from '@/api/client'

// Cache is keyed by lemma so repeated clicks are instant.
const cache = new Map()

export const useDictionaryStore = defineStore('dictionary', {
    state: () => ({
        open: false,
        loading: false,
        lemma: '',
        reading: '',
        surface: '',
        entries: [],
        error: null,
        anchor: null, // { x, y } viewport coordinates of the clicked word
    }),

    actions: {
        /**
         * Look up a word and open the tooltip anchored at the given point.
         * @param {{lemma: string, reading?: string, surface?: string}} word
         * @param {{x: number, y: number}} point
         */
        async lookup(word, point) {
            const lemma = word.lemma || word.surface
            if (!lemma) return

            this.anchor = point
            this.lemma = lemma
            this.reading = word.reading || ''
            this.surface = word.surface || lemma
            this.error = null
            this.open = true

            if (cache.has(lemma)) {
                this.entries = cache.get(lemma)
                return
            }

            this.loading = true
            this.entries = []
            try {
                const data = await api.lookup(lemma)
                const entries = data.entries || []
                cache.set(lemma, entries)
                this.entries = entries
            } catch (err) {
                this.error = err.message
            } finally {
                this.loading = false
            }
        },

        close() {
            this.open = false
            this.anchor = null
        },
    },
})
