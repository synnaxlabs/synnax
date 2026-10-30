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
import { playwright } from "@vitest/browser-playwright";
import { layers, lib, lyraLayers } from "@synnaxlabs/vite-plugin";
import path from "path";
import { defineConfig } from "vite";
import { copyFileSync, mkdirSync } from "fs";

export default defineConfig({
  base: "/lyra/",
  plugins: [
    react(),
    lib({ name: "lyra", modules: true, reactCompiled: true }),
    {
      name: "copy-app-css",
      closeBundle() {
        mkdirSync("dist", { recursive: true });
        for (const file of [
          "base.css",
          "static/theme.css",
          "static/theme-dark.css",
          "static/theme-light.css",
        ])
          copyFileSync(
            path.resolve(`src/theming/${file}`),
            path.resolve(`dist/${path.basename(file)}`),
          );
      },
    },
  ],
  test: {
    globals: true,
    testTimeout: 15_000,
    coverage: {
      include: ["src/**/*.ts", "src/**/*.tsx"],
      exclude: ["src/**/*.spec.ts", "src/**/*.spec.tsx"],
    },
    projects: [
      {
        extends: true,
        test: {
          name: "jsdom",
          environment: "jsdom",
          setupFiles: ["src/testutil/setup.ts"],
          include: ["src/**/*.spec.{ts,tsx}"],
          exclude: ["**/node_modules/**", "**/dist/**", "src/**/*.browser.spec.tsx"],
        },
      },
      {
        extends: true,
        // Specs see the cascade layers of production.
        css: {
          postcss: {
            plugins: [layers(lyraLayers(path.resolve(import.meta.dirname, "..")))],
          },
        },
        test: {
          name: "browser",
          setupFiles: ["src/testutil/browser.ts"],
          include: ["src/**/*.browser.spec.tsx"],
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
