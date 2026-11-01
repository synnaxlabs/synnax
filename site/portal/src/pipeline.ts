// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type APIContext, type MiddlewareHandler } from "astro";
import { sequence } from "astro/middleware";

import { type Portal } from "@/portal";
import { forbidden, toResponse } from "@/server/errors";

const CLERK = "https://*.clerk.accounts.dev https://clerk.synnaxlabs.com";

/** POSTHOG is our own address for PostHog, which the analytics banner loads from. */
const POSTHOG = "https://t.synnaxlabs.com";

const CSP = [
  "default-src 'self'",
  `script-src 'self' 'unsafe-inline' https://vercel.live ${CLERK} https://challenges.cloudflare.com ${POSTHOG}`,
  "style-src 'self' 'unsafe-inline'",
  "font-src 'self' data:",
  `connect-src 'self' https://vercel.live ${CLERK} https://clerk-telemetry.com ${POSTHOG}`,
  "worker-src 'self' blob:",
  "img-src 'self' data: https://img.clerk.com https://vercel.com https://vercel.live",
  "frame-src https://vercel.live https://challenges.cloudflare.com",
  "frame-ancestors 'none'",
  "object-src 'none'",
].join("; ");

const SAFE_METHODS = new Set(["GET", "HEAD", "OPTIONS"]);

/**
 * BEARER_ROUTES authenticate with a bearer secret and no cookie, so a request from any
 * origin is safe there.
 */
const BEARER_ROUTES = new Set([
  "/api/desktop/renew",
  "/api/desktop/unlink",
  "/api/handoff",
]);

/**
 * crossOrigin reports whether a request that changes state came from another origin. A
 * browser without `Sec-Fetch-Site` is judged by its `Origin`.
 */
const crossOrigin = ({ method, headers }: Request, url: URL): boolean => {
  if (SAFE_METHODS.has(method)) return false;
  const site = headers.get("sec-fetch-site");
  if (site != null && site !== "") return site !== "same-origin" && site !== "none";
  const origin = headers.get("origin");
  return origin != null && origin !== url.origin;
};

/** origin refuses a cross-origin request that changes state. */
const origin: MiddlewareHandler = async ({ request, url }, next) => {
  if (!BEARER_ROUTES.has(url.pathname) && crossOrigin(request, url))
    return toResponse(forbidden("Cross-site requests are not allowed"));
  return await next();
};

const headers: MiddlewareHandler = async (_, next) => {
  const response = await next();
  if (!import.meta.env.DEV) response.headers.set("Content-Security-Policy", CSP);
  return response;
};

export interface Config {
  /** auth reads the Clerk session of the request. */
  auth: MiddlewareHandler;
  /** open wires the portal services for one request. */
  open: (context: APIContext) => Portal;
}

/**
 * pipeline builds the middleware every request passes through: the origin check, the
 * session, the portal, and the security headers.
 */
export const pipeline = ({ auth, open }: Config): MiddlewareHandler => {
  const portal: MiddlewareHandler = async (context, next) => {
    context.locals.portal = open(context);
    return await next();
  };
  return sequence(origin, auth, portal, headers);
};
