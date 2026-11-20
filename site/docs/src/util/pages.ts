// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { compare } from "@synnaxlabs/x";

import { NOTES_PAGE } from "@/util/releaseNotes";
import { normalizeRoute } from "@/util/route";

interface Frontmatter {
  title: string;
  heading?: string;
  description?: string;
  /** An ISO 8601 date, set on blog posts. */
  published?: string;
}

export interface Page {
  route: string;
  title: string;
  description?: string;
  published?: string;
}

// Astro does not route files that start with an underscore.
const FRONTMATTER = import.meta.glob<Frontmatter>(
  ["/src/pages/**/*.mdx", "!/src/pages/**/_*.mdx"],
  { import: "frontmatter", eager: true },
);

const toRoute = (file: string): string =>
  normalizeRoute(
    file
      .replace(/^\/src\/pages/, "")
      .replace(/\.mdx$/, "")
      .replace(/(^|\/)index$/, "$1"),
  );

/** Every published docs page. */
export const PAGES: Page[] = [
  ...Object.entries(FRONTMATTER)
    .map(([file, frontmatter]) => ({ route: toRoute(file), ...frontmatter }))
    .map(({ route, title, heading, description, published }) => ({
      route,
      title: (heading ?? title).replace(/`/g, ""),
      description,
      published,
    })),
  NOTES_PAGE,
].sort(compare.by((p) => p.route));

/** The blog posts, newest first. */
export const POSTS: Page[] = PAGES.filter(({ route }) =>
  route.startsWith("/blog/"),
).sort(compare.reverse(compare.by((p) => p.published ?? "")));

/**
 * Every route a search engine should index. The other Astro pages redirect or rewrite
 * to an MDX page.
 */
export const ROUTES: string[] = [
  ...PAGES.map(({ route }) => route),
  "/blog",
  "/privacy-policy",
  "/reliability",
  "/terms-of-service",
].sort();
