import { defineConfig } from 'astro/config';
import react from '@astrojs/react';

// https://astro.build/config
// Build estático para que Go sirva el frontend desde /frontend/dist
// Uso: npm run build → genera dist/ y Go lo sirve en :8080
export default defineConfig({
  output: 'static',
  integrations: [react()],
  server: {
    port: 4321,
    host: true
  },
  vite: {
    server: {
      proxy: {
        '/api': 'http://localhost:8080'
      }
    }
  }
});
