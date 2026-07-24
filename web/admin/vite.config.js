import { sveltekit } from '@sveltejs/kit/vite'
import { defineConfig } from 'vite'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  plugins: [
    tailwindcss(),
    sveltekit()
  ],
  server: {
    proxy: {
      '/api': {
        target: 'http://localhost:8080/'
      },
      '/uploads': {
        target: 'http://localhost:8080/'
      }
    }
  },
  build: {
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (id.includes('node_modules')) {
            const packageName = id.toString().split('node_modules/')[1].split('/')[0].toString()
            // Exclude devalue from manual chunks as it may create empty chunks in static mode
            if (packageName === 'devalue') {
              return null
            }
            return packageName
          }
        }
      }
    },
    chunkSizeWarningLimit: 1000
  }
})
