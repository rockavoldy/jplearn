import { fileURLToPath, URL } from 'node:url'
import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'

// https://vitejs.dev/config/
export default defineConfig(({ mode }) => {
    // Read the shared .env from the repository root so the backend and frontend
    // use a single source of configuration.
    const envDir = fileURLToPath(new URL('..', import.meta.url))
    const env = loadEnv(mode, envDir, '')
    const apiTarget = env.VITE_API_BASE_URL || 'http://localhost:8080'

    return {
        plugins: [vue()],
        envDir,
        resolve: {
            alias: {
                '@': fileURLToPath(new URL('./src', import.meta.url)),
            },
        },
        server: {
            port: 5173,
            proxy: {
                '/api': {
                    target: apiTarget,
                    changeOrigin: true,
                },
            },
        },
    }
})
