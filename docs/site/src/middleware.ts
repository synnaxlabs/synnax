// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { MiddlewareHandler } from "astro";

export const onRequest: MiddlewareHandler = async (context, next) => {
  const { pathname } = context.url;

  // Handle dynamic redirects (catch-all patterns don't work in Astro config on Vercel)
  if (pathname.startsWith("/reference/cluster/")) {
    const slug = pathname.replace("/reference/cluster/", "");
    return context.redirect(`/reference/core/${slug}`, 301);
  }

  if (pathname.startsWith("/reference/device-drivers/")) {
    const slug = pathname.replace("/reference/device-drivers/", "");
    return context.redirect(`/reference/driver/${slug}`, 301);
  }

  const response = await next();

  response.headers.set(
    "Content-Security-Policy",
    "default-src 'self'; script-src 'self' 'unsafe-inline' https://vercel.live https://us-assets.i.posthog.com; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; font-src 'self' https://fonts.gstatic.com data:; connect-src 'self' https://ywd9t0jxcs-dsn.algolia.net https://raw.githubusercontent.com https://us.i.posthog.com https://formspree.io; img-src 'self' data: https://us-assets.i.posthog.com https://synnax.nyc3.cdn.digitaloceanspaces.com; media-src 'self' https://synnax.nyc3.cdn.digitaloceanspaces.com; frame-src https://vercel.live https://www.youtube.com http://localhost:4321; object-src 'none';",
  );

  return response;
};
