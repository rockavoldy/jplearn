// Thin fetch wrapper around the Go backend REST API.
const BASE_URL = import.meta.env.VITE_API_BASE_URL || ''

async function request(path, options = {}) {
    const res = await fetch(`${BASE_URL}${path}`, {
        headers: {
            Accept: 'application/json',
            ...(options.body && !(options.body instanceof FormData)
                ? { 'Content-Type': 'application/json' }
                : {}),
            ...(options.headers || {}),
        },
        ...options,
    })

    if (!res.ok) {
        let detail = res.statusText
        try {
            const body = await res.json()
            if (body && body.error) detail = body.error
        } catch {
            /* ignore non-JSON error bodies */
        }
        throw new Error(`API ${res.status}: ${detail}`)
    }

    if (res.status === 204) return null
    const contentType = res.headers.get('content-type') || ''
    return contentType.includes('application/json') ? res.json() : res.text()
}

export const api = {
    // Articles
    listArticles: (limit = 50) => request(`/api/articles?limit=${limit}`),
    getArticle: (id) => request(`/api/articles/${id}`),
    createArticle: (payload) =>
        request('/api/articles', { method: 'POST', body: JSON.stringify(payload) }),

    // Books & chapters
    listBooks: () => request('/api/books'),
    listChapters: (bookId) => request(`/api/books/${bookId}/chapters`),
    getChapter: (id) => request(`/api/chapters/${id}`),
    uploadEPUB: (file) => {
        const form = new FormData()
        form.append('file', file)
        return request('/api/epubs/upload', { method: 'POST', body: form })
    },

    // Dictionary
    lookup: (lemma) => request(`/api/dictionary?lemma=${encodeURIComponent(lemma)}`),

    // Images (proxied)
    imageURL: (id) => `${BASE_URL}/api/images/${id}`,
}

export default api
