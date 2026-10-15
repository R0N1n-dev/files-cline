import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    target: 'es2020',
    chunkSizeWarningLimit: 900
  },
  test: {
    environment: 'jsdom',
    include: ['tests/**/*.test.js'],
    globals: true
  },
  server: {
    port: 5173,
    proxy: {
      // Browser dev mode: proxy the JSON API to the headless Go server.
      '/api': 'http://127.0.0.1:34115'
    }
  }
})
