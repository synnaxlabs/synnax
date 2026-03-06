// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { clerkMiddleware } from "@clerk/astro/server";
import { sequence } from "astro:middleware";
import type { MiddlewareHandler } from "astro";

const cspMiddleware: MiddlewareHandler = async (_, next) => {
  const response = await next();

  response.headers.set(
    "Content-Security-Policy",
    "default-src 'self'; script-src 'self' 'unsafe-inline' https://vercel.live https://us-assets.i.posthog.com https://*.clerk.accounts.dev https://challenges.cloudflare.com; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; font-src 'self' https://fonts.gstatic.com data:; connect-src 'self' https://ywd9t0jxcs-dsn.algolia.net https://raw.githubusercontent.com https://us.i.posthog.com https://formspree.io https://vercel.live https://*.clerk.accounts.dev; img-src 'self' data: https://us-assets.i.posthog.com https://synnax.nyc3.cdn.digitaloceanspaces.com https://vercel.com https://*.vercel.app https://vercel.live https://img.clerk.com; media-src 'self' https://synnax.nyc3.cdn.digitaloceanspaces.com; frame-src https://vercel.live https://www.youtube.com http://localhost:4321 https://challenges.cloudflare.com; worker-src 'self' blob:; object-src 'none';",
  );

  return response;
};

export const onRequest = sequence(clerkMiddleware(), cspMiddleware);
