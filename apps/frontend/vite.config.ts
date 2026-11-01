import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import { fileURLToPath } from "node:url";
import { defineConfig, loadEnv } from "vite";

export default defineConfig(({ mode }) => ({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      "@assets": fileURLToPath(new URL("../../assets", import.meta.url)),
      "@": fileURLToPath(new URL("./src", import.meta.url)),
    },
  },
  server: {
    fs: { allow: ["../.."] },
    proxy: { "/api": `http://localhost:${loadEnv(mode, "../..", "SERVER_").SERVER_PORT || 8080}` },
  },
}));
