// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { compare } from "@synnaxlabs/x";
import { type CollectionEntry, getCollection } from "astro:content";

import { NOTES_PAGE } from "@/util/releaseNotes";

export interface Page {
  route: string;
  title: string;
  description: string;
  published?: Date;
}

const toPage = ({
  id,
  data: { title, heading, description },
}: CollectionEntry<"reference" | "blog">): Page => ({
  route: `/${id}`,
  title: (heading ?? title).replace(/`/g, ""),
  description,
});

/** Every published docs page. */
export const PAGES: Page[] = [
  ...(await getCollection("reference")).map(toPage),
  ...(await getCollection("blog")).map((post) => ({
    ...toPage(post),
    published: post.data.published,
  })),
  NOTES_PAGE,
].sort(compare.by((p) => p.route));

/** The blog posts, newest first. */
export const POSTS: Page[] = PAGES.filter(({ route }) =>
  route.startsWith("/blog/"),
).sort(compare.reverse(compare.by((p) => p.published?.toISOString() ?? "")));

/**
 * Every route a search engine should index. The other Astro pages redirect or rewrite
 * to a docs page.
 */
export const ROUTES: string[] = [
  ...PAGES.map(({ route }) => route),
  "/blog",
  "/privacy-policy",
  "/reliability",
  "/terms-of-service",
].sort();
