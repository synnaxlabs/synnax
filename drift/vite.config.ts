import { defineConfig } from "vite";
import path from "node:path";
import react from "@vitejs/plugin-react";
import dts from "vite-plugin-dts";
import tsConfigPaths from "vite-tsconfig-paths";

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [
    tsConfigPaths(),
    react(),
    dts({
      insertTypesEntry: true,
    }),
  ],
  build: {
    sourcemap: true,
    lib: {
      entry: path.resolve(__dirname, "src/index.ts"),
      name: "pluto",
      formats: ["cjs", "es"],
      fileName: (format) => `pluto.${format}.js`,
    },
    rollupOptions: {
      external: [
        "react",
        "react-dom",
        "react-redux",
        "@reduxjs/toolkit",
        "proxy-memoize",
      ],
      output: {
        globals: {
          react: "React",
          "react-dom": "ReactDOM",
          "react-redux": "ReactRedux",
          "@reduxjs/toolkit": "ReduxToolkit",
          "proxy-memoize": "ProxyMemoize",
        },
      },
    },
  },
  test: {
    globals: true,
    environment: "jsdom",
    setupFiles: ["setuptests.ts"],
  },
});
