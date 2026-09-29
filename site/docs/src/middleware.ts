// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { MiddlewareHandler } from "astro";

import { Releases } from "@/util/releases";

// The one lookup instance, so its listing cache serves every request of a warm
// function. `DOCS_GITHUB_TOKEN` raises the GitHub API rate limit when set.
const releases = new Releases({ token: process.env.DOCS_GITHUB_TOKEN });

// Every visitor gets the same page, so Vercel's CDN serves it and refreshes it in the
// background after the release listing TTL. The browser minute lets a click reuse the
// page its hover prefetched.
const PAGE_CACHE_CONTROL =
  "public, max-age=60, s-maxage=300, stale-while-revalidate=86400";

export const onRequest: MiddlewareHandler = async (context, next) => {
  context.locals.releases = releases;
  const response = await next();
  if (import.meta.env.DEV) return response;

  if (
    response.status === 200 &&
    response.headers.get("Content-Type")?.startsWith("text/html") === true
  )
    response.headers.set("Cache-Control", PAGE_CACHE_CONTROL);

  response.headers.set(
    "Content-Security-Policy",
    "default-src 'self'; script-src 'self' 'unsafe-inline' https://vercel.live https://us-assets.i.posthog.com; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; font-src 'self' https://fonts.gstatic.com data:; connect-src 'self' https://ywd9t0jxcs-dsn.algolia.net https://demo.synnaxlabs.com:9090 wss://demo.synnaxlabs.com:9090 https://us.i.posthog.com https://us-assets.i.posthog.com https://formspree.io https://vercel.live; img-src 'self' data: https://us-assets.i.posthog.com https://synnax.nyc3.cdn.digitaloceanspaces.com https://vercel.com https://*.vercel.app https://vercel.live; media-src 'self' https://synnax.nyc3.cdn.digitaloceanspaces.com; frame-src https://vercel.live https://www.youtube.com http://localhost:4321; object-src 'none';",
  );

  return response;
};
