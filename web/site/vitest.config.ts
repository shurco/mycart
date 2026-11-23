import { configDefaults, defineConfig } from 'vitest/config'
import { sveltekit } from '@sveltejs/kit/vite'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  plugins: [sveltekit(), tailwindcss()],
  test: {
    environment: 'jsdom',
    globals: true,
    // Browser-mode tests need a real browser session; they run via vitest.browser.config.ts
    exclude: [...configDefaults.exclude, '**/*.browser.test.ts'],
    setupFiles: ['./vitest.setup.ts'],
    coverage: {
      provider: 'v8',
      reporter: ['text', 'json', 'html'],
      exclude: [
        'node_modules/',
        '.svelte-kit/',
        'build/',
        '**/*.test.ts',
        '**/*.spec.ts',
        '**/+page.server.ts',
        '**/+layout.server.ts'
      ],
      lines: 80,
      functions: 80,
      branches: 75,
      statements: 80
    }
  },
  resolve: {
    // Without the browser condition Svelte resolves to its server build and
    // component tests cannot mount anything.
    conditions: ['browser'],
    alias: {
      $lib: '/src/lib',
      '$lib/*': '/src/lib/*'
    }
  }
})
