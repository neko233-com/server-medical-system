import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

// Vite + React 19 stack ("vite-plus"). Dev server proxies /api to the Go backend.
export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
  build: {
    outDir: 'dist',
  },
});
