// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// <reference types="vitest/config" />

import { lib } from "@synnaxlabs/vite-plugin";
import react from "@vitejs/plugin-react";
import path from "path";
import { defineConfig, esmExternalRequirePlugin } from "vite";

export default defineConfig({
  base: "/charon/",
  plugins: [
    esmExternalRequirePlugin({ external: [/^react(-dom)?(\/.*)?$/] }),
    react(),
    lib({ name: "charon" }),
  ],
  build: {
    lib: {
      entry: {
        index: path.resolve(".", "src/index.ts"),
      },
    },
    rolldownOptions: {
      external: ["@synnaxlabs/x"],
      output: {
        globals: {
          react: "React",
          "react-dom": "ReactDOM",
        },
        preserveModules: false,
      },
    },
  },
  test: {
    globals: true,
    environment: "jsdom",
    exclude: ["**/node_modules/**", "**/dist/**"],
    coverage: {
      include: ["src/**/*.ts", "src/**/*.tsx"],
      exclude: ["src/**/*.spec.ts", "src/**/*.spec.tsx", "src/**/*.bench.ts"],
    },
  },
});
