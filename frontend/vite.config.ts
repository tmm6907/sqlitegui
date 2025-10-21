import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'
import tailwindcss from '@tailwindcss/vite'

import path from 'path';

export default defineConfig({
  plugins: [svelte(), tailwindcss()],
  resolve: {
    alias: {
      'src': path.resolve(__dirname, './src')
    }
  },
  server: {
    watch: {
      usePolling: true,
    }
  },
  css: {
    postcss: "./postcss.config.cjs"
  }
})