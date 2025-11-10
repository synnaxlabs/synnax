// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// <reference types="vitest/config" />

import { lib } from "@synnaxlabs/vite-plugin";
import path from "path";
import { defineConfig } from "vite";

export default defineConfig({
  base: "/drift/",
  plugins: [lib({ name: "drift" })],
  build: {
    lib: {
      entry: {
        index: path.resolve(".", "src/index.ts"),
        react: path.resolve(".", "src/react/index.ts"),
        tauri: path.resolve(".", "src/tauri/index.ts"),
      },
    },
    rollupOptions: {
      external: [
        "react",
        "react-dom",
        "react-redux",
        "@reduxjs/toolkit",
        "proxy-memoize",
        "@tauri-apps/api",
      ],
      output: {
        globals: { react: "react", "react-dom": "ReactDOM" },
      },
    },
  },
  test: { globals: true, environment: "jsdom" },
});
