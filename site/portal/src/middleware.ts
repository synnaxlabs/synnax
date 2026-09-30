// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { clerkMiddleware } from "@clerk/astro/server";
import { type MiddlewareHandler } from "astro";
import { sequence } from "astro:middleware";

import { open } from "@/portal";

const CLERK = "https://*.clerk.accounts.dev https://clerk.portal.synnaxlabs.com";

const CSP = [
  "default-src 'self'",
  `script-src 'self' 'unsafe-inline' https://vercel.live ${CLERK} https://challenges.cloudflare.com`,
  "style-src 'self' 'unsafe-inline'",
  "font-src 'self' data:",
  `connect-src 'self' https://vercel.live ${CLERK} https://clerk-telemetry.com`,
  "worker-src 'self' blob:",
  "img-src 'self' data: https://img.clerk.com https://vercel.com https://vercel.live",
  "frame-src https://vercel.live https://challenges.cloudflare.com",
  "object-src 'none'",
].join("; ");

const headers: MiddlewareHandler = async (_, next) => {
  const response = await next();
  if (!import.meta.env.DEV) response.headers.set("Content-Security-Policy", CSP);
  return response;
};

const portal: MiddlewareHandler = async (context, next) => {
  context.locals.portal = open(context);
  return await next();
};

export const onRequest = sequence(clerkMiddleware(), portal, headers);
