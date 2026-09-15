import { defineConfig } from 'vite';

// Development only. Production must route these paths to Go on the same origin.
const target = process.env.NEXUS_API_TARGET || 'http://127.0.0.1:8080';
export default defineConfig({
  server: {
    proxy: {
      '/health': { target, changeOrigin: true },
      '/telemetry': { target, changeOrigin: true },
      '/api/v1/subscribers': { target, changeOrigin: true },
      '/api/v1/devices': { target, changeOrigin: true },
      '/api/v1/sessions': { target, changeOrigin: true },
      '/api/v1/events/recent': { target, changeOrigin: true },
      '/api/v1/network/ip-pool': { target, changeOrigin: true },
    },
  },
});
