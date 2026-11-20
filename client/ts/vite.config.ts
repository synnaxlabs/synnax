// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// <reference types="vitest/config" />

import { isProd, lib } from "@synnaxlabs/vite-plugin";
import path from "path";
import { defineConfig, type Plugin } from "vite";

import packageJSON from "./package.json" with { type: "json" };

// Browsers with no bundler have no `process`, so the browser entry must not read it.
const noProcess = (): Plugin => ({
  name: "no-process",
  generateBundle(_, bundle) {
    for (const chunk of Object.values(bundle))
      if (
        chunk.type === "chunk" &&
        chunk.name !== "testutil" &&
        /(?<![.\w$])process(\??\.[\w$]|\[)/.test(chunk.code)
      )
        this.error(`${chunk.fileName} reads process`);
  },
});

export default defineConfig({
  define: {
    __VERSION__: JSON.stringify(packageJSON.version),
    // The bundled Immer reads this at load, and browsers with no bundler have no
    // `process`.
    "process.env.NODE_ENV": JSON.stringify(isProd() ? "production" : "development"),
  },
  plugins: [lib({ name: "client" }), noProcess()],
  build: {
    lib: {
      entry: {
        index: path.resolve(".", "src/index.ts"),
        testutil: path.resolve(".", "src/testutil/index.ts"),
      },
    },
    rolldownOptions: {
      external: [
        /^zod(\/|$)/,
        "vitest",
        /^@vitest\//,
        /^node:/,
        "@synnaxlabs/alamos",
        "@synnaxlabs/freighter",
        "@synnaxlabs/x",
        "undici",
      ],
    },
  },
  test: {
    globals: true,
    testTimeout: 15_000,
    expect: { poll: { timeout: 5000 } },
    setupFiles: ["src/testutil/setup.ts"],
    exclude: ["**/node_modules/**", "**/dist/**"],
    coverage: {
      include: ["src/**/*.ts", "src/**/*.tsx"],
      exclude: [
        "src/**/*.spec.ts",
        "src/**/*.spec.tsx",
        "src/**/*.bench.ts",
        "src/**/*.gen.ts",
      ],
    },
  },
});
