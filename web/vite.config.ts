import path from "path"
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src")
    }
  },
  server: {
      host: '0.0.0.0',
      port: 4000,
      allowedHosts: true,
      proxy: {
        '/api': {
	  target: 'http://app:8080',
	  changeOrigin: true,
	  secure: false,
	},
	'/ws': {
	    target: 'http://app:8080',
	    ws: true,
	    rewriteWsOrigin: false,
	    configure: (proxy) => {
	      proxy.on('proxyReqWs', (proxyReq, req) => {
	        const clientIp = req.socket.remoteAddress;
	        if (clientIp) {
	          proxyReq.setHeader('X-Forwarded-For', clientIp);
	        }
	      });
	    },
	},
      }
  }
})
