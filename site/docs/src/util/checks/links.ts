// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Check, type Context } from "@/util/checks/check";
import { locate, locateSource, navRefs, type Ref } from "@/util/checks/crawl";
import { attrValues, idValues } from "@/util/checks/html";
import { normalizeRoute } from "@/util/route";

// Hosts that block automated requests or answer too slowly to probe; entries skip the
// external check. Omron redirects and takes several seconds from a datacenter IP, so it
// exhausts every attempt and fails the run. The website and portal deploy apart from
// this build, and a Vercel bot challenge blocks CI runners from them.
const IGNORED_HOSTS: string[] = [
  "automation.omron.com",
  "portal.synnaxlabs.com",
  "synnaxlabs.com",
  "www.synnaxlabs.com",
];

// Sentinel origin for resolving relative hrefs; any other host is external.
const INTERNAL = "http://internal.invalid";

// The site's own absolute links resolve against this build. The live site serves the
// last deploy, so a page this build adds would 404 there.
const SELF_HOST = "docs.synnaxlabs.com";

// npm's website blocks non-browser requests, so package links are validated against the
// registry API instead: 200 = the package/version exists, 404 = dead.
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

interface Failure {
  reason: string;
  // The link leaves the site, so the failure can come from its host, not the page.
  external: boolean;
}

const checkRef = async (
  ctx: Context,
  ids: Map<string, Set<string>>,
  { base, href }: Ref,
): Promise<Failure | null> => {
  let url: URL;
  try {
    url = new URL(href, `${INTERNAL}${base}`);
  } catch {
    return { reason: `unparseable href "${href}"`, external: false };
  }
  if (url.protocol !== "http:" && url.protocol !== "https:") return null;
  if (url.host === SELF_HOST)
    url = new URL(`${url.pathname}${url.search}${url.hash}`, INTERNAL);
  if (url.host !== "internal.invalid") {
    if (IGNORED_HOSTS.includes(url.host)) return null;
    const reason = await ctx.fetchOk(externalURL(url));
    return reason == null ? null : { reason, external: true };
  }
  const route = normalizeRoute(url.pathname);
  const fragment = decodeURIComponent(url.hash).replace(/^#/, "");
  if (!ctx.routes.has(route)) {
    const reason = await ctx.fetchOk(`${ctx.baseURL}${url.pathname}${url.search}`);
    if (reason == null) return null;
    return { reason: `dead internal link ${href} (${reason})`, external: false };
  }
  if (fragment !== "" && ids.get(route)?.has(fragment) !== true)
    return { reason: `missing anchor ${href}`, external: false };
  return null;
};

/**
 * Verifies every link of a page or a nav definition. A broken internal link always
 * fails. A broken external link fails only when strict is true, and warns otherwise.
 */
export const links = (fullCrawl: boolean, strict: boolean): Check => {
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
        collect({ source: route, base: route, href, at: null });
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
          const failure = await checkRef(ctx, ids, ref);
          progress(++done, refs.length);
          if (failure == null) return;
          // Page refs carry a route; nav refs already resolved a file:line.
          const fromPage = ref.source.startsWith("/");
          const where = fromPage ? locate(ref.source, ref.href) : ref.source;
          const message = `${where} - ${failure.reason}`;
          if (strict || !failure.external) {
            report(message);
            return;
          }
          ctx.warn(message, fromPage ? locateSource(ref.source, ref.href) : ref.at);
        }),
      );
    },
  };
};
