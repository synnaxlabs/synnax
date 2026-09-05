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
import { styleText } from "util";

import { type Check, type Context } from "./check.ts";
import { locate, normalizeRoute, pageFiles, PAGES_DIR } from "./crawl.ts";
import { attrValues, idValues } from "./html.ts";

// Hosts that block automated requests or answer too slowly to probe; entries skip the
// external check. Omron redirects and takes several seconds from a datacenter IP, so it
// exhausts both attempts and fails the run.
const IGNORED_HOSTS: string[] = ["automation.omron.com"];

// Sentinel origin for resolving relative hrefs; any other host is external.
const INTERNAL = "http://internal.invalid";

// Release artifact links 404 until the release ships, so a 404 on a URL naming the
// repo's own version warns instead of failing.
const PENDING_VERSION = fs
  .readFileSync("../../core/pkg/version/VERSION", "utf8")
  .trim();

// npm's website blocks non-browser requests, so package links are validated against
// the registry API instead: 200 = the package/version exists, 404 = dead.
const NPM_HOSTS = ["npmjs.com", "www.npmjs.com"];
const NPM_PACKAGE = /^\/package\/(@[^/]+\/[^/]+|[^/]+)(?:\/v\/([^/]+))?\/?$/;

const externalURL = (url: URL): string => {
  if (NPM_HOSTS.includes(url.host)) {
    const match = NPM_PACKAGE.exec(url.pathname);
    if (match != null) {
      const version = match[2] != null ? `/${match[2]}` : "";
      return `https://registry.npmjs.org/${match[1]}${version}`;
    }
  }
  return url.toString();
};

interface Ref {
  source: string;
  base: string;
  href: string;
}

// Deep nav entries only render after hydration, so the crawl can't see them; pull
// every href straight from the nav definitions.
const navRefs = (): Ref[] =>
  pageFiles()
    .filter((f) => f.endsWith("_nav.ts"))
    .flatMap((f) => {
      const text = fs.readFileSync(path.join(PAGES_DIR, f), "utf8");
      return [...text.matchAll(/href:\s*"([^"]+)"/g)].map((m) => ({
        source: `src/pages/${f}:${text.slice(0, m.index).split("\n").length}`,
        base: "/",
        href: m[1],
      }));
    });

const checkRef = async (
  ctx: Context,
  ids: Map<string, Set<string>>,
  { base, href }: Ref,
): Promise<string | null> => {
  let url: URL;
  try {
    url = new URL(href, `${INTERNAL}${base}`);
  } catch {
    return `unparseable href "${href}"`;
  }
  if (url.protocol !== "http:" && url.protocol !== "https:") return null;
  if (url.host !== "internal.invalid") {
    if (IGNORED_HOSTS.includes(url.host)) return null;
    return await ctx.fetchOk(externalURL(url));
  }
  const route = normalizeRoute(url.pathname);
  const fragment = decodeURIComponent(url.hash).replace(/^#/, "");
  if (!ctx.routes.has(route)) {
    const reason = await ctx.fetchOk(`${ctx.baseURL}${url.pathname}${url.search}`);
    return reason == null ? null : `dead internal link ${href} (${reason})`;
  }
  if (fragment !== "" && ids.get(route)?.has(fragment) !== true)
    return `missing anchor ${href}`;
  return null;
};

export const links = (fullCrawl: boolean): Check => {
  const refs: Ref[] = [];
  let anchors = 0;
  const seen = new Set<string>();
  const ids = new Map<string, Set<string>>();
  const collect = (ref: Ref): void => {
    const key = `${ref.source}|${ref.href}`;
    if (seen.has(key)) return;
    seen.add(key);
    refs.push(ref);
  };
  return {
    name: "links",
    page: ({ route, html }) => {
      ids.set(normalizeRoute(route), new Set(idValues(html)));
      for (const href of attrValues(html, "a", "href")) {
        anchors += 1;
        collect({ source: route, base: route, href });
      }
      return [];
    },
    finish: async (ctx, report, progress) => {
      // Canary for the attrValues regex silently matching nothing (see tabs.ts).
      if (fullCrawl && anchors === 0) report("no <a href> found: markup has changed");
      navRefs().forEach(collect);
      let done = 0;
      await Promise.all(
        refs.map(async (ref) => {
          const reason = await checkRef(ctx, ids, ref);
          progress(++done, refs.length);
          if (reason == null) return;
          // Page refs carry a route; nav refs already resolved a file:line.
          const where = ref.source.startsWith("/")
            ? locate(ref.source, ref.href)
            : ref.source;
          if (reason.includes("HTTP 404") && ref.href.includes(PENDING_VERSION)) {
            console.warn(
              `${styleText(["yellow", "bold"], "links")} ${where} - ${reason} ` +
                `(pending the ${PENDING_VERSION} release)`,
            );
            return;
          }
          report(`${where} - ${reason}`);
        }),
      );
    },
  };
};
