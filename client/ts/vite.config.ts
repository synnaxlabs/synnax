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
import { playwright } from "@vitest/browser-playwright";
import { globSync } from "node:fs";
import path from "path";
import { defineConfig, type Plugin } from "vite";
import type { TestProjectInlineConfiguration } from "vitest/config";

import packageJSON from "./package.json" with { type: "json" };

// Specs that read the test Core as a browser project provides it.
const PROVIDED_CORE_SPECS = "src/**/*.provided.spec.ts";

// Spec files that need a Node API, so only the Node project runs them.
const NODE_SPECS = [
  // Set the Cookie header, which a browser forbids.
  "src/auth/auth.cookie.spec.ts",
  // Start HTTP, HTTPS, or WebSocket servers, or use the file system.
  "src/client.spec.ts",
  "src/env/env.spec.ts",
  "src/framer/streamer.spec.ts",
  "src/transport/transport.spec.ts",
  // Reach the Core through a TCP proxy.
  "src/channel/channel.spec.ts",
  "src/connection/downtime.spec.ts",
  "src/task/task.spec.ts",
  // Listen for unhandled rejections on `process`.
  "src/framer/cache/streamer/streamer.spec.ts",
  // Read the filter vectors from the file system.
  "src/**/where.spec.ts",
  // Reads the orphan vectors from the file system.
  "src/schematic/orphans.spec.ts",
  // Checks the error for a Core that no project provides.
  "src/testutil/address.spec.ts",
];

// A stale entry would let a browser run a Node spec.
for (const spec of NODE_SPECS)
  if (globSync(spec, { cwd: import.meta.dirname }).length === 0)
    throw new Error(`${spec} in NODE_SPECS matches no file`);

/**
 * @returns a project that runs the specs in one browser. Browsers run one after
 * another, after the Node projects, so two runs of a spec never share the Core at once.
 */
const browserProject = (
  browser: "chromium" | "firefox" | "webkit",
  groupOrder: number,
): TestProjectInlineConfiguration => ({
  extends: true,
  test: {
    name: browser,
    setupFiles: ["src/testutil/browser.ts"],
    include: ["src/**/*.spec.ts"],
    exclude: [PROVIDED_CORE_SPECS, ...NODE_SPECS],
    benchmark: { include: [] },
    sequence: { groupOrder },
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
      instances: [{ browser }],
    },
  },
});

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
      ],
    },
  },
  test: {
    globals: true,
    testTimeout: 15_000,
    expect: { poll: { timeout: 5000 } },
    projects: [
      {
        extends: true,
        test: {
          name: "client",
          setupFiles: ["src/testutil/setup.ts"],
          exclude: ["**/node_modules/**", "**/dist/**", PROVIDED_CORE_SPECS],
        },
      },
      {
        extends: true,
        test: {
          name: "provided-core",
          setupFiles: ["src/testutil/setup.ts"],
          include: [PROVIDED_CORE_SPECS],
          benchmark: { include: [] },
          provide: { synnaxTestCore: { host: "provided", port: 1 } },
        },
      },
      // V8 coverage reads only Chromium, so `cov` leaves out WebKit and Firefox.
      browserProject("chromium", 1),
      browserProject("webkit", 2),
      browserProject("firefox", 3),
    ],
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
