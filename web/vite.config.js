import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

// Port the Go backend listens on. Must match config.DefaultPort (internal/config).
const BACKEND_PORT = 6970;

// Vite + React 19 stack. The dev server proxies /api to the Go backend so the
// SPA and API share one origin in development (same as in production, where the
// Go binary serves both the API and ./web/dist).
export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: `http://localhost:${BACKEND_PORT}`,
        changeOrigin: true,
      },
      // Imaging films are fetched through an authenticated API route, but keep
      // the raw upload path proxied too for future direct-asset access.
      '/uploads': {
        target: `http://localhost:${BACKEND_PORT}`,
        changeOrigin: true,
      },
    },
  },
  build: {
    outDir: 'dist',
    // Clean the previous build so stale assets never linger in the bundle.
    emptyOutDir: true,
    // Production bundles ship without sourcemaps (smaller + no source exposure).
    sourcemap: false,
    // Raise the warning threshold: vendor chunks are intentionally large.
    chunkSizeWarningLimit: 800,
    rollupOptions: {
      output: {
        // Split vendor libraries into separate long-lived chunks so application
        // code changes don't invalidate expensive third-party caches.
        manualChunks(id) {
          if (!id.includes('node_modules')) return;
          if (id.includes('react')) return 'vendor-react';
          if (id.includes('i18next')) return 'vendor-i18n';
          if (id.includes('axios')) return 'vendor-http';
          return 'vendor-misc';
        },
      },
    },
  },
});
