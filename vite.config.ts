import { writeFileSync } from 'node:fs';
import { defineConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';
export default defineConfig({
  root: 'src/web',
  plugins: [
    svelte(),
    {
      name: 'keep-embed-directory',
      closeBundle() {
        writeFileSync(
          'src/internal/web/dist/.gitkeep',
          'Run make build to generate the embedded dashboard assets.\n',
        );
      },
    },
  ],
  build: {
    // Relative to the frontend root, src/web/.
    outDir: '../internal/web/dist',
    emptyOutDir: true,
  },
  server: {
    proxy: { '/api': 'http://127.0.0.1:8787' },
  },
});
