import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

const devHost = process.env.HOST || "localhost";

export default defineConfig({
  plugins: [react()],
  server: {
    host: devHost,
    port: 5174,
    proxy: {
      "/api": {
        target: "http://localhost:8090",
        changeOrigin: true,
        ws: true,
      },
      "/healthz": {
        target: "http://localhost:8090",
        changeOrigin: true,
      },
    },
  },
  build: {
    outDir: "dist",
    sourcemap: true,
  },
});
