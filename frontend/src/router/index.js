import { createRouter, createWebHistory } from 'vue-router'

// EPUB upload is disabled by default; the route is only registered when
// explicitly enabled at build time via VITE_ENABLE_EPUB_UPLOAD=true.
const epubUploadEnabled = import.meta.env.VITE_ENABLE_EPUB_UPLOAD === 'true'

const routes = [
    {
        path: '/',
        name: 'library',
        component: () => import('@/views/LibraryView.vue'),
    },
    ...(epubUploadEnabled
        ? [
            {
                path: '/upload',
                name: 'upload',
                component: () => import('@/views/UploadView.vue'),
            },
        ]
        : []),
    {
        path: '/articles/:id',
        name: 'article',
        component: () => import('@/views/ArticleView.vue'),
        props: true,
    },
    {
        path: '/books/:id',
        name: 'book',
        component: () => import('@/views/BookView.vue'),
        props: true,
    },
    {
        path: '/chapters/:id',
        name: 'chapter',
        component: () => import('@/views/ChapterView.vue'),
        props: true,
    },
]

const router = createRouter({
    history: createWebHistory(),
    routes,
    scrollBehavior() {
        return { top: 0 }
    },
})

export default router
