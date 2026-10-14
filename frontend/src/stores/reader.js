import { defineStore } from 'pinia'
import api from '@/api/client'

// Reader store handles loading article / chapter HTML content.
export const useReaderStore = defineStore('reader', {
    state: () => ({
        kind: null, // 'article' | 'chapter'
        id: null,
        title: '',
        html: '',
        meta: null,
        loading: false,
        error: null,
    }),

    actions: {
        async loadArticle(id) {
            this.reset('article', id)
            this.loading = true
            try {
                const article = await api.getArticle(id)
                this.title = article.title
                this.html = article.content_html
                this.meta = article
            } catch (err) {
                this.error = err.message
            } finally {
                this.loading = false
            }
        },

        async loadChapter(id) {
            this.reset('chapter', id)
            this.loading = true
            try {
                const chapter = await api.getChapter(id)
                this.title = chapter.title || `Chapter ${chapter.chapter_index + 1}`
                this.html = chapter.content_html
                this.meta = chapter
            } catch (err) {
                this.error = err.message
            } finally {
                this.loading = false
            }
        },

        reset(kind = null, id = null) {
            this.kind = kind
            this.id = id
            this.title = ''
            this.html = ''
            this.meta = null
            this.error = null
            this.loading = false
        },
    },
})
