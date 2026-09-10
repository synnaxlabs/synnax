// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import fs from "fs";
import path from "path";

import { type Page } from "./check.ts";

export const PAGES_DIR = "./src/pages";
const CONCURRENCY = 4;

export const normalizeRoute = (route: string): string =>
  route !== "/" && route.endsWith("/") ? route.slice(0, -1) : route;

/** Lists every file under src/pages, as paths relative to it. */
export const pageFiles = (): string[] => {
  const files: string[] = [];
  const walk = (dir: string): void => {
    for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
      const child = path.join(dir, entry.name);
      if (entry.isDirectory()) walk(child);
      else files.push(path.relative(PAGES_DIR, child));
    }
  };
  walk(PAGES_DIR);
  return files;
};

// Maps a route back to the page source file that renders it.
const routeSource = (route: string): string | null => {
  const rel = route === "/" ? "index" : route.replace(/^\//, "");
  for (const suffix of [".mdx", "/index.mdx", ".astro", "/index.astro"]) {
    const file = path.join(PAGES_DIR, `${rel}${suffix}`);
    if (fs.existsSync(file)) return file;
  }
  return null;
};

// Characters that can continue a URL; a match bounded by one is a substring of a
// longer URL, not the reference itself.
const URL_CHAR = /[\w\-.~/#?&=%]/;

/**
 * Points a failure at its source: "route:line" when needle appears in the page's
 * source file, the bare route otherwise (e.g. links emitted by a layout component).
 */
export const locate = (route: string, needle: string): string => {
  const file = routeSource(route);
  if (file == null) return route;
  const text = fs.readFileSync(file, "utf8");
  for (let at = text.indexOf(needle); at !== -1; at = text.indexOf(needle, at + 1)) {
    if (URL_CHAR.test(text[at - 1] ?? "")) continue;
    if (URL_CHAR.test(text[at + needle.length] ?? "")) continue;
    return `${route}:${text.slice(0, at).split("\n").length}`;
  }
  return route;
};

/** Derives every route from the src/pages file tree. */
export const enumerateRoutes = (): string[] => {
  const routes = pageFiles()
    .filter((f) => /\.(mdx|astro)$/.test(f))
    .map((f) => `/${f.replace(/\.(mdx|astro)$/, "").replace(/(^|\/)index$/, "$1")}`);
  return [...new Set(routes)];
};

export interface CrawlResult {
  pages: Page[];
  failures: string[];
}

// Node's fetch wraps the useful error (ECONNREFUSED and friends) in e.cause.
const describe = (e: unknown): string => {
  if (!(e instanceof Error)) return String(e);
  return e.cause instanceof Error ? `${e.message}: ${e.cause.message}` : e.message;
};

export const crawlPages = async (
  baseURL: string,
  routes: string[],
): Promise<CrawlResult> => {
  const queue = [...routes];
  const pages: Page[] = [];
  const failures: string[] = [];
  const worker = async (): Promise<void> => {
    for (;;) {
      const route = queue.shift();
      if (route == null) return;
      // One retry, for a DOCS_URL dev server dropping connections mid-compile; the
      // static server answers instantly.
      let failure = "";
      for (let attempt = 1; attempt <= 2; attempt++)
        try {
          const res = await fetch(`${baseURL}${route}`, {
            signal: AbortSignal.timeout(15000),
          });
          if (!res.ok) {
            failure = `${route}: HTTP ${res.status}`;
            break;
          }
          pages.push({ route, html: await res.text() });
          failure = "";
          break;
        } catch (e) {
          failure = `${route}: ${describe(e)}`;
          await new Promise((resolve) => setTimeout(resolve, 2000));
        }

      if (failure !== "") failures.push(failure);
    }
  };
  await Promise.all(Array.from({ length: CONCURRENCY }, worker));
  return { pages, failures };
};
