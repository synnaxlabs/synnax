import path from "node:path";

import { defineConfig } from "vite";
import dts from "vite-plugin-dts";
import tsconfigPaths from "vite-tsconfig-paths";

export default defineConfig({
  plugins: [
    tsconfigPaths(),
    dts({
      insertTypesEntry: true,
    }),
  ],
  build: {
    sourcemap: true,
    lib: {
      entry: path.resolve(__dirname, "src/index.ts"),
      name: "client",
      formats: ["cjs", "es"],
      fileName: (format) => `client.${format}.js`,
    },
  },
  test: {
    globals: true,
    environment: "jsdom",
  },
});
