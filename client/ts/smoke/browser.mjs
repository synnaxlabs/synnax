// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Serves the consumer project at argv[2] and loads four pages in Chromium, Firefox, and
// WebKit: a Vite, an esbuild, and a webpack bundle, and importmap.html, which loads the
// packed files with no bundler.

import { copyFile, readFile } from "node:fs/promises";
import { createServer } from "node:http";
import { extname, join } from "node:path";

import { build as esbuild } from "esbuild";
import { chromium, firefox, webkit } from "playwright";
import { build } from "vite";
import webpack from "webpack";

const root = process.argv[2];
await build({ root, base: "./", logLevel: "error" });
await esbuild({
  entryPoints: [join(root, "smoke.mjs")],
  bundle: true,
  format: "esm",
  outfile: join(root, "esbuild", "smoke.mjs"),
  logLevel: "error",
});
const stats = await new Promise((resolve, reject) =>
  webpack(
    {
      mode: "production",
      target: "web",
      context: root,
      entry: "./smoke.mjs",
      output: { path: join(root, "webpack"), filename: "smoke.mjs", module: true },
      experiments: { outputModule: true },
      performance: { hints: false },
    },
    (err, stats) => (err == null ? resolve(stats) : reject(err)),
  ),
);
if (stats.hasErrors()) throw new Error(stats.toString("errors-only"));
for (const bundler of ["esbuild", "webpack"])
  await copyFile(join(root, "index.html"), join(root, bundler, "index.html"));

const TYPES = {
  ".html": "text/html",
  ".js": "text/javascript",
  ".mjs": "text/javascript",
};
const server = createServer((req, res) => {
  const path = join(root, new URL(req.url, "http://localhost").pathname);
  readFile(path).then(
    (body) => {
      res.writeHead(200, { "Content-Type": TYPES[extname(path)] ?? "text/plain" });
      res.end(body);
    },
    () => res.writeHead(404).end(),
  );
});
await new Promise((resolve) => server.listen(0, "localhost", resolve));
const origin = `http://localhost:${server.address().port}`;
const env = {
  SYNNAX_TEST_HOST: process.env.SYNNAX_TEST_HOST,
  SYNNAX_TEST_PORT: process.env.SYNNAX_TEST_PORT,
};

try {
  for (const type of [chromium, firefox, webkit]) {
    const browser = await type.launch();
    try {
      const pages = [
        "dist/index.html",
        "esbuild/index.html",
        "webpack/index.html",
        "importmap.html",
      ];
      for (const page of pages) {
        const tab = await browser.newPage();
        await tab.addInitScript((env) => (globalThis.smokeEnv = env), env);
        const failed = new Promise((_, reject) => {
          tab.on("pageerror", reject);
          tab.on("response", (res) => {
            if (res.request().resourceType() === "script" && !res.ok())
              reject(new Error(`${res.status()} ${res.url()}`));
          });
        });
        await tab.goto(`${origin}/${page}`);
        await Promise.race([tab.waitForFunction(() => globalThis.smoked), failed]);
        console.log(`${type.name()} ${page}: ok`);
      }
    } finally {
      await browser.close();
    }
  }
} finally {
  server.close();
}
