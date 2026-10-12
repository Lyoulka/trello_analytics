/// <reference types="vitest/config" />
import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  const proxyTarget =
    process.env.VITE_API_PROXY_TARGET || env.VITE_API_PROXY_TARGET || 'http://backend:8080'

  return {
    plugins: [vue()],
    server: {
      host: '0.0.0.0',
      port: 5173,
      // Docker Desktop on Windows: file events often need polling
      watch: {
        usePolling: true,
        interval: 300,
      },
      proxy: {
        '/api': {
          target: proxyTarget,
          changeOrigin: true,
        },
        '/health': {
          target: proxyTarget,
          changeOrigin: true,
        },
        '/ready': {
          target: proxyTarget,
          changeOrigin: true,
        },
      },
    },
    test: {
      environment: 'jsdom',
      globals: true,
    },
  }
})
