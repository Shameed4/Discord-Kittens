import { defineConfig } from 'vitest/config';
import react from '@vitejs/plugin-react';
import tailwindcss from '@tailwindcss/vite';

export default defineConfig({
  // Load env vars (e.g. VITE_DISCORD_CLIENT_ID) from the repo-root .env, which is
  // shared with the Go backend. Without this, Vite only looks in client/ and the
  // client_id ends up undefined — the Discord SDK handshake then hangs on ready().
  envDir: '..',
  plugins: [react(), tailwindcss()],
  test: {
    environment: 'jsdom',
  },
  server: {
    allowedHosts: true,
    proxy: {
      // Single /api entry (ws: true handles the /api/ws upgrade too). No rewrite:
      // the backend serves under /api, matching the Vercel rewrite and Discord
      // URL mapping, which also forward the prefix unchanged.
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        ws: true,
      },
    },
  },
});
