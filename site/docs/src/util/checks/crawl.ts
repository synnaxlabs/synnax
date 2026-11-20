// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { sleep, TimeSpan, unique } from "@synnaxlabs/x";
import fs from "fs";
import path from "path";

import { type Page, type Source } from "@/util/checks/check";

const PAGES_DIR = "./src/pages";
const DOCS_DIR = "./src/docs";
const CONCURRENCY = 4;

const listFiles = (root: string): string[] => {
  const files: string[] = [];
  const walk = (dir: string): void => {
    for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
      const child = path.join(dir, entry.name);
      if (entry.isDirectory()) walk(child);
      else files.push(path.relative(root, child));
    }
  };
  walk(root);
  return files;
};

/** Returns true if Astro routes the page file: its name has no leading underscore. */
const routed = (f: string): boolean => !path.basename(f).startsWith("_");

/** Lists every file under src/pages, as paths relative to it. */
const pageFiles = (): string[] => listFiles(PAGES_DIR);

/** Lists every docs page file under src/docs, as paths relative to it. */
export const docFiles = (): string[] =>
  listFiles(DOCS_DIR).filter((f) => f.endsWith(".mdx") && routed(f));

// Maps a route back to the page source file that renders it.
const routeSource = (route: string): string | null => {
  const rel = route === "/" ? "index" : route.replace(/^\//, "");
  for (const suffix of [".mdx", "/index.mdx", ".astro", "/index.astro"])
    for (const dir of [DOCS_DIR, PAGES_DIR]) {
      const file = path.join(dir, `${rel}${suffix}`);
      if (fs.existsSync(file)) return file;
    }
  return null;
};

// Characters that can continue a URL; a match bounded by one is a substring of a
// longer URL, not the reference itself.
const URL_CHAR = /[\w\-.~/#?&=%]/;

/**
 * Finds the line of the page's source file that holds needle. Returns null when the
 * source does not hold it, such as a link emitted by a layout component.
 */
export const locateSource = (route: string, needle: string): Source | null => {
  const file = routeSource(route);
  if (file == null) return null;
  const text = fs.readFileSync(file, "utf8");
  for (let at = text.indexOf(needle); at !== -1; at = text.indexOf(needle, at + 1)) {
    if (URL_CHAR.test(text[at - 1] ?? "")) continue;
    if (URL_CHAR.test(text[at + needle.length] ?? "")) continue;
    return { file, line: text.slice(0, at).split("\n").length };
  }
  return null;
};

/** Points a failure at its source: "route:line" when known, else the bare route. */
export const locate = (route: string, needle: string): string => {
  const source = locateSource(route, needle);
  return source == null ? route : `${route}:${source.line}`;
};

/** Returns the route of a page file, given relative to src/pages or src/docs. */
export const toRoute = (f: string): string =>
  `/${f.replace(/\.(mdx|astro)$/, "").replace(/(^|\/)index$/, "$1")}`;

/**
 * Derives every route from the src/pages and src/docs file trees. Dynamic routes are
 * left out, since they serve pages that have files of their own.
 */
export const enumerateRoutes = (): string[] => {
  const routes = pageFiles()
    .filter((f) => f.endsWith(".astro") && routed(f) && !f.includes("["))
    .concat(docFiles())
    .map(toRoute);
  return unique.unique(routes);
};

/** A link and where it came from. */
export interface Ref {
  source: string;
  base: string;
  href: string;
  /** Where a nav definition holds the link. Null for a link of a rendered page. */
  at: Source | null;
}

/**
 * Lists every href in the nav definitions. Deep nav entries only render after
 * hydration, so the crawl cannot see them.
 */
export const navRefs = (): Ref[] =>
  pageFiles()
    .filter((f) => f.endsWith("_nav.ts"))
    .flatMap((f) => {
      const text = fs.readFileSync(path.join(PAGES_DIR, f), "utf8");
      return [...text.matchAll(/href:\s*"([^"]+)"/g)].map((m) => {
        const line = text.slice(0, m.index).split("\n").length;
        return {
          source: `src/pages/${f}:${line}`,
          base: "/",
          href: m[1],
          at: { file: path.join(PAGES_DIR, f), line },
        };
      });
    });

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
          await sleep.sleep(TimeSpan.milliseconds(2000));
        }

      if (failure !== "") failures.push(failure);
    }
  };
  await Promise.all(Array.from({ length: CONCURRENCY }, worker));
  return { pages, failures };
};
