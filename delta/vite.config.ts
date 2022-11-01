import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import svgr from "vite-plugin-svgr";
import tsconfigPaths from "vite-tsconfig-paths";

// https://vitejs.dev/config/
export default defineConfig({
  clearScreen: false,
  server: {
    port: 5174,
    strictPort: true,
  },
  envPrefix: ["VITE_", "TAURI_"],
  plugins: [tsconfigPaths(), react(), svgr()],
  build: {
    target: ["es2021", "chrome100", "safari13"],
    minify: !process.env.TAURI_DEBUG ? "esbuild" : false,
    sourcemap: !process.env.TAURI_DEBUG,
  },
});
