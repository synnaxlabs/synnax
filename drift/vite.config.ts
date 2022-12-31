// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

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
      name: "drift",
      formats: ["cjs", "es"],
      fileName: (format) => `drift.${format}.js`,
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
  },
});
