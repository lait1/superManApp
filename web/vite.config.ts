import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import { fileURLToPath, URL } from 'node:url';

// The frontend calls the API at the RELATIVE path /api/v1 (see src/api/client.ts).
// Both the dev server and the Docker `preview` server proxy /api to the Go backend.
//
// Target is overridable via API_PROXY_TARGET:
//   - host dev (`make dev` + `make web-dev`):  default http://localhost:8080
//   - docker compose (`make up`, preview):     http://backend:8080  (compose service name)
const apiTarget = process.env.API_PROXY_TARGET || 'http://localhost:8080';

const proxy = {
  '/api': {
    target: apiTarget,
    changeOrigin: true,
  },
};

// Allow Telegram Mini App tunnels (cloudflared / ngrok) so the dev/preview server
// doesn't reject the public HTTPS host. See docs/13-running.md §4.
const allowedHosts = ['localhost', '.trycloudflare.com', '.ngrok-free.app', '.ngrok.io'];

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  server: {
    host: true,
    port: 5173,
    proxy,
    allowedHosts,
  },
  preview: {
    host: true,
    port: 5173,
    proxy,
    allowedHosts,
  },
});
