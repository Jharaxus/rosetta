import { defineConfig, loadEnv } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd())
  // In Docker dev: VITE_BACKEND_URL=http://backend:8090
  // Local dev (no Docker): defaults to localhost
  const backendTarget = env.VITE_BACKEND_URL ?? 'http://localhost:8090'
  const capTarget = env.VITE_CAP_SERVER_URL ?? 'http://localhost:3001'

  return {
    plugins: [react()],
    server: {
      host: '0.0.0.0',
      port: 5173,
      proxy: {
        '/api': {
          target: backendTarget,
          changeOrigin: true,
        },
        '/cap-api': {
          target: capTarget,
          changeOrigin: true,
          rewrite: (path) => path.replace(/^\/cap-api/, ''),
        },
      },
    },
  }
})
