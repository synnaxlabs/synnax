// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// <reference types="vitest/config" />

import react from "@vitejs/plugin-react";
import * as path from "path";
import { defineConfig } from "vite";
import tsconfigPaths from "vite-tsconfig-paths";
import importMetaUrlPlugin from "@codingame/esbuild-import-meta-url-plugin";

const isDev = process.env.VITE_IS_DEV === "true";

export default defineConfig({
  clearScreen: false,
  server: { port: 5173, strictPort: true },
  resolve: {
    alias: isDev
      ? {
          "@synnaxlabs/pluto/dist": path.resolve(__dirname, "../pluto/dist"),
          "@synnaxlabs/pluto": path.resolve(__dirname, "../pluto/src"),
          "@synnaxlabs/x/dist": path.resolve(__dirname, "../x/ts/dist"),
          "@synnaxlabs/x": path.resolve(__dirname, "../x/ts/src"),
          "@synnaxlabs/drift/dist": path.resolve(__dirname, "../drift/dist"),
          "@synnaxlabs/drift": path.resolve(__dirname, "../drift/src"),
          "@synnaxlabs/media/dist": path.resolve(__dirname, "../x/media/dist"),
          "@synnaxlabs/media": path.resolve(__dirname, "../x/media/src"),
        }
      : {},
  },
  envPrefix: ["VITE_", "TAURI_"],
  plugins: [react(), tsconfigPaths({ root: __dirname })],
  optimizeDeps: {
    esbuildOptions: {
      plugins: [importMetaUrlPlugin],
    },
  },
  build: {
    target: process.env.TAURI_PLATFORM === "windows" ? "chrome105" : "safari16",
    minify: !isDev,
    sourcemap: isDev,
    // We don't really care about maintaining a small bundle size right now, as this file
    // is loaded directly from disc instead of OTN
    chunkSizeWarningLimit: 10000 /* kbs */,
  },
  define: { IS_DEV: isDev },
  worker: {
    format: "es",
  },
  test: {
    globals: true,
    environment: "jsdom",
  },
});
