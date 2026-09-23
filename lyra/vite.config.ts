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
import { copyFileSync, mkdirSync } from "fs";

export default defineConfig({
  base: "/lyra/",
  plugins: [
    esmExternalRequirePlugin({ external: [/^react(-dom)?(\/.*)?$/] }),
    react(),
    lib({ name: "lyra", modules: true }),
    {
      name: "copy-theme-css",
      closeBundle() {
        mkdirSync("dist", { recursive: true });
        for (const file of ["theme.css", "theme-dark.css", "theme-light.css"])
          copyFileSync(
            path.resolve(`src/theming/static/${file}`),
            path.resolve(`dist/${file}`),
          );
      },
    },
  ],
  build: {
    rolldownOptions: {
      external: [
        "vitest",
        /^@vitest\//,
        /^@testing-library\//,
        /^react(-dom)?(\/.*)?$/,
        /^react-icons(\/.*)?$/,
        /^@fontsource(-variable)?\//,
        "clsx",
        "zod",
        "@synnaxlabs/x",
      ],
    },
  },
  test: {
    globals: true,
    environment: "jsdom",
    setupFiles: ["src/testutil/setup.ts"],
    testTimeout: 15_000,
    exclude: ["**/node_modules/**", "**/dist/**"],
    coverage: {
      include: ["src/**/*.ts", "src/**/*.tsx"],
      exclude: ["src/**/*.spec.ts", "src/**/*.spec.tsx"],
    },
  },
});
