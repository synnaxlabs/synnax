// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";
import svgr from "vite-plugin-svgr";
import tsconfigPaths from "vite-tsconfig-paths";

export default defineConfig({
  clearScreen: false,
  server: {
    port: 5173,
    strictPort: true,
  },
  envPrefix: ["VITE_", "TAURI_"],
  plugins: [react(), tsconfigPaths(), svgr()],
  build: {
    target: ["es2021", "chrome100", "safari13"],
    // minify: !process.env.TAURI_DEBUG ? "esbuild" : false,
    minify: false,
    sourcemap: false,
    // We don't really care about maintaining a small bundle size right now, as this file
    // is loaded directly from disc instead of OTN
    chunkSizeWarningLimit: 10000 /* kbs */,
  },
});
