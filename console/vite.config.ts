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
import { playwright } from "@vitest/browser-playwright";
import { appendFileSync, createWriteStream, mkdirSync } from "node:fs";
import * as fs from "node:fs/promises";
import * as path from "path";
import { defineConfig, normalizePath, type Plugin } from "vite";
import type { TestProjectInlineConfiguration } from "vitest/config";

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
const BROWSER_SPECS = "src/**/*.browser.spec.{ts,tsx}";
const DESKTOP_BROWSER_SPECS = "src/**/*.desktop.browser.spec.{ts,tsx}";
const SAMPLE_SPEC = "src/feature/project/sample.desktop.browser.spec.ts";
const JSDOM = { environment: "jsdom", setupFiles: ["src/testutil/setuptests.ts"] };

interface BrowserProjectProps {
  name: string;
  include?: string[];
  exclude?: string[];
  desktop?: boolean;
  webkit?: boolean;
}

/**
 * @returns a project that runs its specs in Chromium and WebKit. A desktop project
 * builds the specs as Synnax Desktop, and its browsers carry a `desktop-` prefix. With
 * `webkit` false, the project runs Chromium alone, as `<name>-chromium`.
 */
const browserProject = ({
  name,
  include = [BROWSER_SPECS],
  exclude = [],
  desktop = false,
  webkit = true,
}: BrowserProjectProps): TestProjectInlineConfiguration => {
  const prefix = desktop ? "desktop-" : "";
  // A CI runner has 4 CPUs. Parallel Chromium pages starve the jsdom workers that share
  // them, so their timers fire seconds late and their specs time out. WebKit runs in a
  // step of its own, with no jsdom workers beside it.
  const pages = process.env.CI === "true" ? { maxWorkers: 1 } : {};
  return {
    extends: true,
    define: { DESKTOP: desktop },
    // The optimizer misses these until a spec loads them, then reloads the page and
    // fails the specs that are running.
    optimizeDeps: {
      include: [
        "@tauri-apps/plugin-deep-link",
        "@tauri-apps/plugin-dialog",
        "@tauri-apps/plugin-fs",
        "@tauri-apps/plugin-shell",
      ],
    },
    test: {
      name,
      setupFiles: ["src/testutil/browser.ts"],
      include,
      exclude,
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
        instances: webkit
          ? [
              { browser: "chromium", name: `${prefix}chromium`, ...pages },
              { browser: "webkit", name: `${prefix}webkit` },
            ]
          : [{ browser: "chromium", name: `${name}-chromium`, ...pages }],
      },
    },
  };
};

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

// Appends each response that is not 2xx or 304, and each request closed before its
// response, to the file in SYNNAX_TEST_REQUEST_LOG. WebKit names no URL when a module
// import fails.
const requestLog = (): Plugin => ({
  name: "request-log",
  apply: "serve",
  configureServer: (server) => {
    const file = process.env.SYNNAX_TEST_REQUEST_LOG;
    if (file == null) return;
    if (file.trim() === "") throw new Error("SYNNAX_TEST_REQUEST_LOG is blank");
    // Throws at start when the file cannot be written.
    mkdirSync(path.dirname(file), { recursive: true });
    appendFileSync(
      file,
      `${new Date().toISOString()} server started (pid ${process.pid})\n`,
    );
    // One stream keeps the lines in order with one open file.
    const stream = createWriteStream(file, { flags: "a" });
    let open = true;
    stream.on("error", (error) => {
      open = false;
      console.error(`Stopped writing ${file}: ${error.message}`);
    });
    server.httpServer?.once("close", () => {
      open = false;
      stream.end();
    });
    const start = Date.now();
    const log = (line: string): void => {
      if (open) stream.write(`${new Date().toISOString()} ${line}\n`);
    };
    server.middlewares.use((req, res, next) => {
      const begin = Date.now();
      res.on("close", () => {
        const at = `+${begin - start}ms (${Date.now() - begin}ms)`;
        const { statusCode: status } = res;
        if (!res.writableFinished) log(`${at} aborted ${req.method} ${req.url}`);
        else if ((status < 200 || status >= 300) && status !== 304)
          log(`${at} ${status} ${res.statusMessage} ${req.method} ${req.url}`);
      });
      next();
    });
  },
});

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
  server: { port: Number(process.env.SYNNAX_CONSOLE_PORT ?? 5173), strictPort: true },
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
  envPrefix: ["VITE_", "TAURI_ENV_"],
  // The Console's own CSS, left unlayered, overrides every layer.
  css: { postcss: { plugins: [layers(plutoLayers(repoRoot))] } },
  plugins: [
    react(),
    reactCompiler(),
    reactDevtools(),
    workspaceSourcemaps(),
    stripSourcesContent(),
    requestLog(),
  ],
  build: {
    target: process.env.TAURI_ENV_PLATFORM === "windows" ? "chrome125" : "safari17.4",
    minify: !isDev,
    // Always emit source maps. The Fallback error UI fetches them at runtime to resolve
    // minified stack traces (see pluto/src/errors/stack/stack.ts).
    sourcemap: true,
    // The Console ships two ways: bundled into the Tauri desktop app (loaded from disk)
    // and embedded into the Synnax Core binary (served over HTTP to browsers via
    // core/pkg/console). Bundle size matters in the browser-served path, but not enough
    // to enforce the default Vite warning threshold.
    chunkSizeWarningLimit: 10000 /* kbs */,
  },
  define: {
    IS_DEV: isDev,
    DESKTOP: desktop,
    RELEASE: release,
    CORE_PORT: Number(process.env.SYNNAX_TEST_PORT ?? 9090),
  },
  worker: {
    format: "es",
    // The worker is a separate build and does not inherit `plugins`, so its frames
    // would otherwise stop at client/ts/dist.
    plugins: () => [workspaceSourcemaps()],
  },
  test: {
    globals: true,
    testTimeout: 15_000,
    hookTimeout: 15_000,
    // A Core write can take more than the 1 s default, so polls wait as long as Testing
    // Library does.
    expect: { poll: { timeout: 5000 } },
    // Specs cover the work that flags hide, as the integration build does.
    env: { VITE_FLAGS_ENABLED: "true" },
    // DESKTOP is a build constant, so the specs of each build run as their own project.
    projects: [
      {
        extends: true,
        test: {
          ...JSDOM,
          name: "console",
          exclude: ["**/node_modules/**", "**/dist/**", DESKTOP_SPECS, BROWSER_SPECS],
        },
      },
      {
        extends: true,
        define: { DESKTOP: true },
        test: { ...JSDOM, name: "desktop", include: [DESKTOP_SPECS] },
      },
      browserProject({ name: "browser", exclude: [DESKTOP_BROWSER_SPECS] }),
      browserProject({
        name: "desktop-browser",
        include: [DESKTOP_BROWSER_SPECS],
        exclude: [SAMPLE_SPEC],
        desktop: true,
      }),
      // The product creates the sample project with fixed keys, so two browsers on one
      // Core collide. Its spec runs in Chromium alone.
      browserProject({
        name: "desktop-sample",
        include: [SAMPLE_SPEC],
        desktop: true,
        webkit: false,
      }),
    ],
    coverage: {
      include: ["src/**/*.ts", "src/**/*.tsx"],
      exclude: ["src/**/*.spec.ts", "src/**/*.spec.tsx", "src/**/*.bench.ts"],
    },
  },
});
