// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Bundles the page in the consumer project at argv[2] with Vite, then loads it in
// Chromium and WebKit.

import { chromium, webkit } from "playwright";
import { build, preview } from "vite";

const root = process.argv[2];
const env = (name) => JSON.stringify(process.env[name]) ?? "undefined";
const define = {
  "process.env.SYNNAX_TEST_HOST": env("SYNNAX_TEST_HOST"),
  "process.env.SYNNAX_TEST_PORT": env("SYNNAX_TEST_PORT"),
};
await build({ root, define, logLevel: "error" });
const server = await preview({ root, preview: { port: 4173, strictPort: true } });
try {
  for (const type of [chromium, webkit]) {
    const browser = await type.launch();
    try {
      const page = await browser.newPage();
      const failed = new Promise((_, reject) => page.on("pageerror", reject));
      await page.goto(server.resolvedUrls.local[0]);
      await Promise.race([page.waitForFunction(() => globalThis.smoked), failed]);
      console.log(`${type.name()}: ok`);
    } finally {
      await browser.close();
    }
  }
} finally {
  await server.close();
}
