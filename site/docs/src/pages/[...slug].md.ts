// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type APIRoute } from "astro";

import { fromPage } from "@/util/markdown";
import { PAGES } from "@/util/pages";

const ROUTES = new Set(PAGES.map(({ route }) => route));

// The static check build prerenders every page; the deployed site renders on request.
export const getStaticPaths = () =>
  PAGES.map(({ route }) => ({ params: { slug: route.slice(1) } }));

/** Serves a docs page as Markdown at its URL plus ".md", for LLMs and agents. */
export const GET: APIRoute = async ({ params, rewrite, site }) => {
  const route = `/${params.slug}`;
  if (!ROUTES.has(route)) return new Response("Not found", { status: 404 });
  const page = await rewrite(route);
  return new Response(fromPage(await page.text(), new URL(route, site)), {
    headers: {
      "Content-Type": "text/markdown; charset=utf-8",
      "Cache-Control": "public, s-maxage=3600, stale-while-revalidate=86400",
    },
  });
};
