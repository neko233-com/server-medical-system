import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import { mkdirSync, writeFileSync, existsSync } from 'node:fs';
import { resolve } from 'node:path';

// Port the Go backend listens on. Must match config.DefaultPort (internal/config).
const BACKEND_PORT = 6970;

/**
 * Keeps a placeholder file inside the build output directory.
 *
 * The Go binary embeds `web/dist` with `go:embed`, and an embed directive that
 * matches no files is a compile error. Vite empties the output directory on
 * every build, which would delete the placeholder and break `go build` on a
 * machine where the frontend was never built.
 *
 * Recreating the placeholder after each build keeps `go build ./...` working
 * in every situation: fresh clone, frontend rebuilt, or backend-only changes.
 */
function keepDistPlaceholder() {
  return {
    name: 'keep-dist-placeholder',
    closeBundle() {
      const outDir = resolve(process.cwd(), 'dist');
      if (!existsSync(outDir)) mkdirSync(outDir, { recursive: true });
      const keep = resolve(outDir, '.gitkeep');
      if (!existsSync(keep)) writeFileSync(keep, '');
    },
  };
}

// Vite + React 19 stack. The dev server proxies /api to the Go backend so the
// SPA and API share one origin in development (same as in production, where
// the Go binary serves both the API and the embedded SPA).
export default defineConfig({
  plugins: [react(), keepDistPlaceholder()],
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
