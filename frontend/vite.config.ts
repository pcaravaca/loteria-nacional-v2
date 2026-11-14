import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { fileURLToPath, URL } from 'node:url'

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: { '@': fileURLToPath(new URL('./src', import.meta.url)) },
  },
  server: {
    port: 5173,
    // El proyecto vive en un mount NTFS (D:) con mtimes poco fiables: sin
    // polling, Vite puede servir modulos cacheados despues de una edicion.
    watch: { usePolling: true, interval: 300 },
    proxy: {
      '/api': { target: 'http://localhost:8080', changeOrigin: true },
    },
  },
})
