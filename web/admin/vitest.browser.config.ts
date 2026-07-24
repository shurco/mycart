import { defineConfig } from 'vitest/config'
import { sveltekit } from '@sveltejs/kit/vite'
import tailwindcss from '@tailwindcss/vite'
import svg from 'vite-plugin-svelte-svg'
import { webdriverio } from '@vitest/browser-webdriverio'

export default defineConfig({
  plugins: [
    tailwindcss(),
    sveltekit(),
    svg({
      svgoConfig: {
        plugins: [
          {
            name: 'preset-default',
            params: {
              overrides: {
                removeViewBox: false
              }
            }
          }
        ]
      }
    })
  ],
  test: {
    browser: {
      enabled: true,
      provider: webdriverio({
        capabilities: {
          'goog:chromeOptions': {
            binary: '/usr/local/bin/chrome',
            args: [
              '--headless',
              '--no-sandbox',
              '--disable-setuid-sandbox',
              '--disable-dev-shm-usage',
              '--disable-gpu'
            ]
          },
          'wdio:chromedriverOptions': {
            binary: '/usr/local/bin/chromedriver'
          }
        }
      }),
      instances: [
        {
          browser: 'chrome'
        }
      ],
      headless: true
    },
    // Browser tests run in real browsers, no need for jsdom
    include: ['src/**/*.browser.test.ts'],
    testTimeout: 30000
  },
  resolve: {
    alias: {
      $lib: '/src/lib',
      '$lib/*': '/src/lib/*'
    }
  }
})
