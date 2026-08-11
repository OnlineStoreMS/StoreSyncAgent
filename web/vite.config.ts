import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  base: process.env.VITE_BASE || '/',
  plugins: [vue(),
    {
      name: 'runtime-config-base',
      transformIndexHtml(html) {
        const baseUrl = process.env.VITE_BASE || '/'
        const tag = `<script src="${baseUrl}runtime-config.js"></script>`
        const cleaned = html.replace(/\s*<script src=["'][^"']*runtime-config\.js["']><\/script>/g, '')
        if (cleaned.includes('<head>')) {
          return cleaned.replace('<head>', `<head>\n    ${tag}`)
        }
        return `${tag}\n${cleaned}`
      },
    },
],
  server: {
    port: 5178,
    proxy: {
      '/api': {
        target: 'http://localhost:8097',
        changeOrigin: true,
      },
      '/iam': {
        target: 'http://localhost:8091',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/iam/, '/api/v1'),
      },
    },
  },
})
