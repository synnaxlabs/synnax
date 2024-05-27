import * as path from "path";
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tsconfigPaths from "vite-tsconfig-paths";

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react(), tsconfigPaths()],
  resolve: {
    alias: {
      "@synnaxlabs/pluto/dist": path.resolve(__dirname, "../dist"),
      "@synnaxlabs/pluto": path.resolve(__dirname, "../src"),
    },
  },
});
