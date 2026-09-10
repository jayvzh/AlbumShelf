/// <reference types="vitest/config" />
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'

// https://vite.dev/config/
export default defineConfig({
  plugins: [vue(), tailwindcss()],
  // Vitest 仅覆盖纯函数单测（utils/*.test.ts）
  test: {
    environment: 'node',
  },
  server: {
    // 监听所有网卡，允许局域网设备（手机/其他电脑）通过本机 IP 访问 dev 服务
    host: true,
    // 开发/自测固定端口 5160（scripts/dev.sh 与本端口保持一致），
    // 端口被占用时直接报错，避免 Vite 自动顺延导致脚本/文档端口失效
    port: 5160,
    strictPort: true,
    // dev 模式将 /api 代理到 Go 后端（http://localhost:8080）
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
})
