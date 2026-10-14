import { defineStore } from 'pinia'
import api from '@/api/client'

export const useLibraryStore = defineStore('library', {
    state: () => ({
        articles: [],
        books: [],
        loading: false,
        error: null,
    }),

    actions: {
        async fetchAll() {
            this.loading = true
            this.error = null
            try {
                const [articles, books] = await Promise.all([api.listArticles(), api.listBooks()])
                this.articles = articles || []
                this.books = books || []
            } catch (err) {
                this.error = err.message
            } finally {
                this.loading = false
            }
        },

        async uploadEPUB(file) {
            const book = await api.uploadEPUB(file)
            await this.fetchAll()
            return book
        },
    },
})
