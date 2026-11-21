// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// <reference types="vitest/config" />

import { layers, lib, plutoLayers } from "@synnaxlabs/vite-plugin";
import react from "@vitejs/plugin-react";
import { playwright } from "@vitest/browser-playwright";
import path from "path";
import { defineConfig, esmExternalRequirePlugin } from "vite";

export default defineConfig({
  base: "/pluto/",
  define: { CORE_PORT: Number(process.env.SYNNAX_TEST_PORT ?? 9090) },
  plugins: [
    esmExternalRequirePlugin({ external: [/^react(-dom)?(\/.*)?$/] }),
    react(),
    // Pluto's maps are 17MB gzipped, mostly sourcesContent from bundled dependencies,
    // so they stay out of the tarball. Every other package publishes its maps.
    lib({ name: "pluto", publishSourcemaps: false, reactCompiled: true }),
  ],
  build: {
    lib: {
      entry: {
        index: path.resolve(".", "src/index.ts"),
        ether: path.resolve(".", "src/ether.ts"),
        testutil: path.resolve(".", "src/testutil/index.ts"),
      },
    },
    rolldownOptions: {
      external: [
        "vitest",
        /^@vitest\//,
        "react-hook-form",
        /^zod(\/|$)/,
        "@synnaxlabs/x",
        /^@synnaxlabs\/lyra\//,
        "@synnaxlabs/client",
        "@synnaxlabs/alamos",
        "@synnaxlabs/freighter",
        /^@fontsource(-variable)?\//,
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
    // vscode-languageclient's "./browser" entry only resolves under the "browser"
    // exports condition, which Vitest's node-based resolver does not apply.
    alias: {
      "vscode-languageclient/browser": path.resolve(
        ".",
        "node_modules/vscode-languageclient/lib/browser/main.js",
      ),
    },
    testTimeout: 15_000,
    coverage: {
      include: ["src/**/*.ts", "src/**/*.tsx"],
      exclude: [
        "src/**/*.spec.ts",
        "src/**/*.spec.tsx",
        "src/**/*.bench.ts",
        "src/**/*.bench.tsx",
      ],
    },
    projects: [
      {
        extends: true,
        test: {
          name: "jsdom",
          environment: "jsdom",
          setupFiles: ["src/testutil/setup.ts"],
          include: ["src/**/*.spec.{ts,tsx}"],
          exclude: [
            "**/node_modules/**",
            "**/dist/**",
            "src/**/*.browser.spec.tsx",
            "src/**/*.builtins.spec.{ts,tsx}",
          ],
        },
      },
      {
        extends: true,
        test: {
          // Every spec file shares the built-in records of one Core, so the specs that
          // change them run one file at a time, after every other spec.
          name: "builtins",
          environment: "jsdom",
          setupFiles: ["src/testutil/setup.ts"],
          include: ["src/**/*.builtins.spec.{ts,tsx}"],
          fileParallelism: false,
          sequence: { groupOrder: 1 },
        },
      },
      {
        extends: true,
        // Specs see the cascade layers of production.
        css: {
          postcss: {
            plugins: [layers(plutoLayers(path.resolve(import.meta.dirname, "..")))],
          },
        },
        test: {
          name: "browser",
          setupFiles: ["src/testutil/browser.ts"],
          include: ["src/**/*.browser.spec.tsx"],
          // A browser has no `process` to read the test Core from.
          provide: {
            synnaxTestCore: {
              host: process.env.SYNNAX_TEST_HOST ?? "localhost",
              port: Number(process.env.SYNNAX_TEST_PORT ?? 9090),
            },
          },
          browser: {
            enabled: true,
            headless: true,
            provider: playwright(),
            // V8 coverage reads only Chromium, so `cov` leaves out WebKit.
            instances: [
              { browser: "chromium", name: "chromium" },
              { browser: "webkit", name: "webkit" },
            ],
          },
        },
      },
    ],
  },
});
