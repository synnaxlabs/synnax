// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// <reference types="vitest/config" />

import { layers, plutoLayers, reactCompiler } from "@synnaxlabs/vite-plugin";
import react from "@vitejs/plugin-react";
import * as fs from "node:fs/promises";
import * as path from "path";
import { defineConfig, normalizePath, type Plugin } from "vite";

const isDev = process.env.VITE_IS_DEV === "true";
// Builds compile workspace packages from source, so code splitting reaches into them.
// Specs run against the built packages.
const fromSource = process.env.VITEST == null;
// Builds Synnax Desktop, the app that bundles and manages its own Core.
const desktop = process.env.VITE_DESKTOP === "true";
// Builds a Desktop release in CI.
const release = process.env.VITE_RELEASE === "true";
const repoRoot = path.resolve(import.meta.dirname, "..");
const DESKTOP_SPECS = "src/**/*.desktop.spec.{ts,tsx}";

// Rollup ignores the sourceMappingURL comment inside prebuilt workspace bundles, so
// the Console's map would bottom out at pluto/dist/pluto.js and friends. Loading the
// adjacent .map here lets Rollup chain it, so crash-screen frames resolve all the way
// to library TS sources.
const workspaceSourcemaps = (): Plugin => {
  const distRoot = normalizePath(repoRoot);
  return {
    name: "workspace-sourcemaps",
    async load(id) {
      const cleanId = normalizePath(id.split("?")[0]);
      if (
        !cleanId.startsWith(distRoot) ||
        !cleanId.includes("/dist/") ||
        !cleanId.endsWith(".js")
      )
        return null;
      try {
        const [code, map] = await Promise.all([
          fs.readFile(cleanId, "utf-8"),
          fs.readFile(`${cleanId}.map`, "utf-8"),
        ]);
        return { code, map };
      } catch {
        // A dist file without a map falls through to the default loader.
        return null;
      }
    },
  };
};

// The dev server loads the standalone React DevTools (`pnpm devtools`) before React, so
// React finds the DevTools hook when it starts. Builds never load it.
const reactDevtools = (): Plugin => ({
  name: "react-devtools",
  apply: "serve",
  transformIndexHtml: () => [
    {
      tag: "script",
      attrs: { src: "http://localhost:8097" },
      injectTo: "head-prepend",
    },
  ],
});

// sourcesContent is 83% of the emitted map weight, and the Core embeds every byte of
// dist into its binary. The crash screen reads only positions, so the inlined copies of
// the original files are dropped.
// Rewriting in generateBundle does not hold: the entry chunk's map is re-serialized
// from chunk.map during the write, discarding the edit.
const stripSourcesContent = (): Plugin => ({
  name: "strip-sources-content",
  async writeBundle(options, bundle) {
    const dir = options.dir;
    if (dir == null) return;
    await Promise.all(
      Object.keys(bundle)
        .filter((name) => name.endsWith(".map"))
        .map(async (name) => {
          const file = path.join(dir, name);
          const map = JSON.parse(await fs.readFile(file, "utf-8")) as {
            sourcesContent?: unknown;
          };
          delete map.sourcesContent;
          await fs.writeFile(file, JSON.stringify(map));
        }),
    );
  },
});

export default defineConfig({
  clearScreen: false,
  server: { port: 5173, strictPort: true },
  resolve: {
    tsconfigPaths: true,
    alias: fromSource
      ? {
          "@synnaxlabs/pluto/dist": path.resolve(repoRoot, "pluto/dist"),
          "@synnaxlabs/pluto": path.resolve(repoRoot, "pluto/src"),
          "@synnaxlabs/lyra/dist": path.resolve(repoRoot, "lyra/dist"),
          "@synnaxlabs/lyra": path.resolve(repoRoot, "lyra/src"),
          "@synnaxlabs/x/dist": path.resolve(repoRoot, "x/ts/dist"),
          "@synnaxlabs/x": path.resolve(repoRoot, "x/ts/src"),
          "@synnaxlabs/drift/dist": path.resolve(repoRoot, "drift/dist"),
          "@synnaxlabs/drift": path.resolve(repoRoot, "drift/src"),
        }
      : {},
  },
  envPrefix: ["VITE_", "TAURI_"],
  // The Console's own CSS, left unlayered, overrides every layer.
  css: { postcss: { plugins: [layers(plutoLayers(repoRoot))] } },
  plugins: [
    react(),
    reactCompiler(),
    reactDevtools(),
    workspaceSourcemaps(),
    stripSourcesContent(),
  ],
  build: {
    target: process.env.TAURI_ENV_PLATFORM === "windows" ? "chrome111" : "safari16.4",
    minify: !isDev,
    // Always emit source maps. The Fallback error UI fetches them at runtime to resolve
    // minified stack traces (see pluto/src/errors/resolveStack.ts).
    sourcemap: true,
    // The Console ships two ways: bundled into the Tauri desktop app (loaded from disk)
    // and embedded into the Synnax Core binary (served over HTTP to browsers via
    // core/pkg/console). Bundle size matters in the browser-served path, but not enough
    // to enforce the default Vite warning threshold.
    chunkSizeWarningLimit: 10000 /* kbs */,
  },
  define: { IS_DEV: isDev, DESKTOP: desktop, RELEASE: release },
  worker: {
    format: "es",
    // The worker is a separate build and does not inherit `plugins`, so its frames
    // would otherwise stop at client/ts/dist.
    plugins: () => [workspaceSourcemaps()],
  },
  test: {
    globals: true,
    environment: "jsdom",
    setupFiles: ["src/testutil/setuptests.ts"],
    testTimeout: 15_000,
    // DESKTOP is a build constant, so the specs of each build run as their own project.
    projects: [
      {
        extends: true,
        test: {
          name: "console",
          exclude: ["**/node_modules/**", "**/dist/**", DESKTOP_SPECS],
        },
      },
      {
        extends: true,
        define: { DESKTOP: true },
        test: { name: "desktop", include: [DESKTOP_SPECS] },
      },
    ],
    coverage: {
      include: ["src/**/*.ts", "src/**/*.tsx"],
      exclude: ["src/**/*.spec.ts", "src/**/*.spec.tsx", "src/**/*.bench.ts"],
    },
  },
});
