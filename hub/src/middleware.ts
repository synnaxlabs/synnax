// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { MiddlewareHandler } from "astro";

export const onRequest: MiddlewareHandler = async (_, next) => {
  const response = await next();
  if (import.meta.env.DEV) return response;

  response.headers.set(
    "Content-Security-Policy",
    "default-src 'self'; script-src 'self' 'unsafe-inline' https://vercel.live https://us-assets.i.posthog.com https://*.clerk.accounts.dev https://clerk.docs.synnaxlabs.com https://challenges.cloudflare.com; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; font-src 'self' https://fonts.gstatic.com data:; connect-src 'self' https://ywd9t0jxcs-dsn.algolia.net https://raw.githubusercontent.com https://us.i.posthog.com https://us-assets.i.posthog.com https://formspree.io https://vercel.live https://*.clerk.accounts.dev https://clerk.docs.synnaxlabs.com https://clerk-telemetry.com; worker-src 'self' blob:; img-src 'self' data: https://img.clerk.com https://us-assets.i.posthog.com https://synnax.nyc3.cdn.digitaloceanspaces.com https://vercel.com https://*.vercel.app https://vercel.live; media-src 'self' https://synnax.nyc3.cdn.digitaloceanspaces.com; frame-src https://vercel.live https://www.youtube.com https://challenges.cloudflare.com http://localhost:4321; object-src 'none';",
  );

  return response;
};
