import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import path from 'path'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    port: 3000,
    host: true, // Required for Docker
    proxy: {
      '/api': {
        // Use 'api' service name when in Docker, localhost otherwise
        target: process.env.VITE_API_URL || 'http://localhost:8080',
        changeOrigin: true,
      },
    },
    watch: {
      // Use polling for Docker volume mounts (more reliable than inotify)
      usePolling: true,
      interval: 1000,
    },
  },
})
