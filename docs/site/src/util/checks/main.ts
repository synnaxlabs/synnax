// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Crawls every page of the static check build in dist/ and runs each registered
// check over the results. Set DOCS_URL to check an already-running server instead.
//
// usage: check-site [check ...] [--route prefix ...]
//   pnpm check-site:build                        build dist/ for checking
//   pnpm check-site                              all checks, all pages
//   pnpm check-site links media                  only the named checks
//   pnpm check-site --route /reference/driver    only routes under the prefix

import { preview } from "astro";
import fs from "fs";
import { styleText } from "util";

import { type Check, type Context } from "./check.ts";
import { crawlPages, enumerateRoutes, normalizeRoute } from "./crawl.ts";
import { createFetcher } from "./fetch.ts";
import { links } from "./links.ts";
import { media } from "./media.ts";
import { tabs } from "./tabs.ts";

const PORT = 4399;
// A trailing slash would defeat the fetcher's prefix-matched localhost bypass.
const EXTERNAL = process.env.DOCS_URL?.replace(/\/+$/, "");

const names: string[] = [];
const prefixes: string[] = [];
const argv = process.argv.slice(2);
for (let i = 0; i < argv.length; i++)
  if (argv[i] === "--route") {
    const prefix = argv[i + 1];
    if (prefix == null) {
      console.error("--route requires a prefix, e.g. --route /reference/driver");
      process.exit(1);
    }
    prefixes.push(prefix);
    i++;
  } else if (argv[i].startsWith("--")) {
    console.error(`unknown flag ${argv[i]}`);
    console.error("usage: check-site [check ...] [--route prefix ...]");
    process.exit(1);
  } else names.push(argv[i]);

const fullCrawl = prefixes.length === 0;
const ALL: Check[] = [tabs(fullCrawl), links(fullCrawl), media(fullCrawl)];
const unknown = names.filter((name) => ALL.every((check) => check.name !== name));
if (unknown.length > 0) {
  const valid = ALL.map((check) => check.name).join(", ");
  console.error(`unknown checks: ${unknown.join(", ")} (valid: ${valid})`);
  process.exit(1);
}
const CHECKS =
  names.length === 0 ? ALL : ALL.filter((check) => names.includes(check.name));

let routes = enumerateRoutes();
if (prefixes.length > 0) {
  routes = routes.filter((route) => prefixes.some((p) => route.startsWith(p)));
  if (routes.length === 0) {
    console.error(`no routes match ${prefixes.join(", ")}`);
    process.exit(1);
  }
}

// Failures stream as they are found; the end of the run prints per-check totals. In
// a terminal each failure first clears any in-place progress line.
const counts = new Map<string, number>();
const record = (check: string, messages: string[]): void => {
  if (messages.length === 0) return;
  counts.set(check, (counts.get(check) ?? 0) + messages.length);
  for (const message of messages) {
    if (process.stdout.isTTY) process.stdout.write("\r\x1b[2K");
    console.error(`${styleText(["red", "bold"], check)} ${message}`);
  }
};

// In-place counter for a finish hook; a newline finishes the line so the next
// output starts fresh. CI logs skip it.
const progressFor =
  (check: string) =>
  (done: number, total: number): void => {
    if (!process.stdout.isTTY) return;
    process.stdout.write(styleText("dim", `\r${check}: checking ${done}/${total}...`));
    if (done === total) process.stdout.write("\n");
  };

let server: Awaited<ReturnType<typeof preview>> | undefined;
try {
  let baseURL = EXTERNAL;
  if (baseURL == null) {
    if (!fs.existsSync("dist")) {
      console.error("no dist/ build found: run pnpm check-site:build first");
      process.exit(1);
    }
    server = await preview({
      root: ".",
      configFile: "astro.check.config.ts",
      server: { port: PORT },
      logLevel: "error",
    });
    baseURL = `http://localhost:${server.port}`;
    console.log(styleText("dim", `serving dist/ at ${baseURL}`));
  }
  const ctx: Context = {
    routes: new Set(routes.map(normalizeRoute)),
    baseURL,
    fetchOk: createFetcher(baseURL),
  };
  console.log(styleText("dim", `crawling ${routes.length} routes...`));
  const { pages, failures } = await crawlPages(baseURL, routes);
  record("crawl", failures);
  for (const page of pages)
    for (const check of CHECKS) record(check.name, check.page?.(page) ?? []);
  for (const check of CHECKS)
    await check.finish?.(
      ctx,
      (message) => record(check.name, [message]),
      progressFor(check.name),
    );
  if (counts.size > 0) {
    const totals = [...counts].map(([check, n]) => `${check}: ${n}`).join(", ");
    console.error(styleText(["red", "bold"], `failures — ${totals}`));
    process.exitCode = 1;
  } else {
    const checked = CHECKS.map((check) => check.name).join(", ");
    console.log(styleText("green", `checked ${pages.length} pages: ${checked} passed`));
  }
} catch (e) {
  console.error(e instanceof Error ? e.message : String(e));
  process.exitCode = 1;
} finally {
  await server?.stop();
}
