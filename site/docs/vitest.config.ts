// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import react from "@astrojs/react";
import { layers, lyraLayers } from "@synnaxlabs/vite-plugin";
import { playwright } from "@vitest/browser-playwright";
import { getViteConfig } from "astro/config";
import path from "path";
import { configDefaults, type ViteUserConfigFn } from "vitest/config";

const BROWSER_SPECS = "src/**/*.browser.spec.tsx";

const config: ViteUserConfigFn = getViteConfig(
  {
    define: { __DOCS_VERSION__: "null" },
    test: {
      projects: [
        {
          extends: true,
          test: {
            name: "jsdom",
            environment: "jsdom",
            exclude: [...configDefaults.exclude, BROWSER_SPECS],
          },
        },
        {
          extends: true,
          // Specs see the cascade layers of production.
          css: {
            postcss: {
              plugins: [layers(lyraLayers(path.resolve(import.meta.dirname, "../..")))],
            },
          },
          test: {
            name: "browser",
            setupFiles: ["src/testutil/browser.ts"],
            include: [BROWSER_SPECS],
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
  },
  {
    root: import.meta.dirname,
    configFile: false,
    integrations: [react()],
  },
);

export default config;
