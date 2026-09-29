// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { enabled } from "@/flags";
import { normalizeRoute } from "@/util/route";

interface Frontmatter {
  title: string;
  heading?: string;
  description?: string;
  /** An ISO 8601 date, set on blog posts. */
  published?: string;
  flag?: string;
}

export interface Page {
  route: string;
  title: string;
  description?: string;
  published?: string;
}

const FRONTMATTER = import.meta.glob<Frontmatter>("/src/pages/**/*.mdx", {
  import: "frontmatter",
  eager: true,
});

// Each version's notes render inside /releases, never on their own.
const RELEASE_NOTES = /^\/releases\/.+/;

const toRoute = (file: string): string =>
  normalizeRoute(
    file
      .replace(/^\/src\/pages/, "")
      .replace(/\.mdx$/, "")
      .replace(/(^|\/)index$/, "$1"),
  );

/** Every published MDX page. A page behind a flag is published while the flag is on. */
export const PAGES: Page[] = Object.entries(FRONTMATTER)
  .map(([file, frontmatter]) => ({ route: toRoute(file), ...frontmatter }))
  .filter(
    ({ route, flag }) => !RELEASE_NOTES.test(route) && (flag == null || enabled(flag)),
  )
  .map(({ route, title, heading, description, published }) => ({
    route,
    title: (heading ?? title).replace(/`/g, ""),
    description,
    published,
  }))
  .sort((a, b) => a.route.localeCompare(b.route));

/** The blog posts, newest first. */
export const POSTS: Page[] = PAGES.filter(({ route }) =>
  route.startsWith("/blog/"),
).sort((a, b) => (b.published ?? "").localeCompare(a.published ?? ""));

/**
 * Every route a search engine should index. The other Astro pages redirect or rewrite
 * to an MDX page.
 */
export const ROUTES: string[] = [
  ...PAGES.map(({ route }) => route),
  "/blog",
  "/privacy-policy",
].sort();
