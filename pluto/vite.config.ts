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
import { lib } from "@synnaxlabs/vite-plugin";
import path from "path";
import { defineConfig, esmExternalRequirePlugin } from "vite";

export default defineConfig({
  base: "/pluto/",
  plugins: [
    esmExternalRequirePlugin({ external: [/^react(-dom)?(\/.*)?$/] }),
    react(),
    // Pluto's maps are 17MB gzipped, mostly sourcesContent from bundled dependencies,
    // so they stay out of the tarball. Every other package publishes its maps.
    lib({ name: "pluto", publishSourcemaps: false }),
  ],
  build: {
    lib: {
      entry: {
        index: path.resolve(".", "src/index.ts"),
        ether: path.resolve(".", "src/ether.ts"),
        testutil: path.resolve(".", "src/testutil/index.ts"),
        color: path.resolve(".", "src/color/index.ts"),
      },
    },
    rolldownOptions: {
      external: [
        "vitest",
        /^@vitest\//,
        "react-hook-form",
        "zod",
        "@synnaxlabs/x",
        /^@synnaxlabs\/lyra\//,
        "@synnaxlabs/client",
        "@synnaxlabs/alamos",
        "@synnaxlabs/freighter",
        "@synnaxlabs/media",
      ],
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
    // vscode-languageclient's "./browser" entry only resolves under the "browser"
    // exports condition, which Vitest's node-based resolver does not apply.
    alias: {
      "vscode-languageclient/browser": path.resolve(
        ".",
        "node_modules/vscode-languageclient/lib/browser/main.js",
      ),
    },
    setupFiles: ["src/mock/setuptests.ts"],
    testTimeout: 15_000,
    exclude: ["**/node_modules/**", "**/dist/**"],
    coverage: {
      include: ["src/**/*.ts", "src/**/*.tsx"],
      exclude: ["src/**/*.spec.ts", "src/**/*.spec.tsx", "src/**/*.bench.ts"],
    },
  },
});
