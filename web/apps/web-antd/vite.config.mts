import { defineConfig } from '@vben/vite-config';

export default defineConfig(async () => {
  return {
    application: {},
    vite: {
      server: {
        proxy: {
          '/api': {
            changeOrigin: true,
            rewrite: (path) => path.replace(/^\/api/, ''),
            // mock代理目标地址
            // target: 'http://localhost:5320/api',
            target: 'http://127.0.0.1:8080/api',
            ws: true,
          },
          '/v1/chat': {
            changeOrigin: true,
            rewrite: (path) => path.replace(/^\/v1\/chat/, ''),
            // mock代理目标地址
            // target: 'http://localhost:5320/api',
            target: 'http://127.0.0.1:8080/v1/chat',
            ws: true,
          },
        },
      },
    },
  };
});
