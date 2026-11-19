// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Serves the consumer project at argv[2] and loads two pages in Chromium, Firefox, and
// WebKit: a Vite bundle, and importmap.html, which loads the packed files with no
// bundler.

import { readFile } from "node:fs/promises";
import { createServer } from "node:http";
import { extname, join } from "node:path";

import { chromium, firefox, webkit } from "playwright";
import { build } from "vite";

const root = process.argv[2];
await build({ root, base: "./", logLevel: "error" });

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
      for (const page of ["dist/index.html", "importmap.html"]) {
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
