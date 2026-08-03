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
import path from "path";
import { defineConfig } from "vite";

import packageJSON from "./package.json" with { type: "json" };

export default defineConfig({
  define: { __VERSION__: JSON.stringify(packageJSON.version) },
  plugins: [lib({ name: "client" })],
  build: {
    lib: {
      entry: {
        index: path.resolve(".", "src/index.ts"),
        testutil: path.resolve(".", "src/testutil/index.ts"),
      },
    },
    rolldownOptions: { external: ["zod", "vitest", /^@vitest\//] },
  },
  test: {
    globals: true,
    exclude: ["**/node_modules/**", "**/dist/**"],
    coverage: {
      include: ["src/**/*.ts", "src/**/*.tsx"],
      exclude: ["src/**/*.spec.ts", "src/**/*.spec.tsx", "src/**/*.bench.ts"],
    },
  },
});
