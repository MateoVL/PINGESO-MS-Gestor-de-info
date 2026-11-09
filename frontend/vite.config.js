import { fileURLToPath } from 'node:url'
import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  test: {
    coverage: {
      include: ['src/**'],
      reporter: ['text', ['lcovonly', { projectRoot: fileURLToPath(new URL('..', import.meta.url)) }]]
    }
  },
  server: {
    host: '0.0.0.0',
    port: 3000,
    strictPort: true,
    watch: {
      usePolling: true
    }
  },
  preview: {
    host: '0.0.0.0',
    port: 3000
  }
})
