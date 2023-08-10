// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import path from "path";
import { lib } from "@synnaxlabs/vite-plugin";
import { defineConfig } from "vite";

export default defineConfig({
  plugins: [lib({ name: "pluto" })],
  build: {
    sourcemap: true,
    minify: true,
    lib: {
      entry: {
        index: path.resolve(".", "src/index.ts"),
        worker: path.resolve(".", "src/worker.ts"),
        std: path.resolve(".", "src/core/std/index.ts"),
        theming: path.resolve(".", "src/core/theming/index.ts"),
      },
    },
    rollupOptions: {
      external: ["react", "react-dom", "react-hook-form"],
      output: {
        globals: {
          react: "React",
          "react-dom": "ReactDOM",
        },
      },
    },
  },
  test: {
    globals: true,
    environment: "jsdom",
    setupFiles: ["src/mock/setuptests.ts"],
  },
});
