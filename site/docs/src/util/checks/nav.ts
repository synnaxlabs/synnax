// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Check } from "@/util/checks/check";
import {
  enumerateRoutes,
  navRefs,
  pageFiles,
  routed,
  toRoute,
} from "@/util/checks/crawl";
import { normalizeRoute } from "@/util/route";

const referencePages = (): string[] =>
  pageFiles()
    .filter((f) => f.startsWith("reference/") && f.endsWith(".mdx") && routed(f))
    .map((f) => normalizeRoute(toRoute(f)));

/**
 * Fails every reference page missing from the nav, since the nav sets reading order and
 * edition, and every nav entry that names no page, since a redirect is not a page.
 */
export const nav = (fullCrawl: boolean): Check => ({
  name: "nav",
  finish: async (_, report) => {
    if (!fullCrawl) return;
    const refs = navRefs().map((ref) => ({
      ...ref,
      route: normalizeRoute(ref.href.split("#")[0]),
    }));
    const linked = new Set(refs.map(({ route }) => route));
    for (const route of referencePages())
      if (!linked.has(route)) report(`${route}: page is missing from the nav`);
    const routes = new Set(enumerateRoutes().map(normalizeRoute));
    for (const { source, href, route } of refs)
      if (!routes.has(route)) report(`${source}: nav entry ${href} names no page`);
  },
});
