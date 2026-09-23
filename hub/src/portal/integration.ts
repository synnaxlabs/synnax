// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type AstroIntegration } from "astro";

/**
 * ROUTES maps every signed-in URL to its source file. The files live outside
 * `src/pages` so the static site check never builds or crawls a session-bound page.
 */
const ROUTES: Record<string, string> = {
  "/sign-in/reset": "pages/reset.astro",
  "/sign-in/[...rest]": "pages/sign-in.astro",
  "/sign-up/[...rest]": "pages/sign-up.astro",
  "/sso-callback": "pages/sso-callback.astro",
  "/account": "pages/licenses/index.astro",
  "/account/licenses/activate": "pages/licenses/activate.astro",
  "/account/licenses/[key]": "pages/licenses/[key].astro",
  "/account/settings": "pages/settings.astro",
  "/account/staff/licenses": "pages/staff/licenses.astro",
  "/desktop/sign-in": "pages/desktop/sign-in.astro",
  "/api/webhooks/clerk": "routes/webhooks/clerk.ts",
  "/api/cron/expiry": "routes/cron/expiry.ts",
  "/api/licenses": "routes/licenses/issue.ts",
  "/api/licenses/[key]": "routes/licenses/amend.ts",
  "/api/licenses/[key]/activate": "routes/licenses/activate.ts",
  "/api/licenses/[key]/revoke": "routes/licenses/revoke.ts",
  "/api/licenses/[key]/floating": "routes/licenses/floating.ts",
  "/api/activations/[key]/token": "routes/activations/token.ts",
  "/api/activations/[key]/release": "routes/activations/release.ts",
  "/api/activations/[key]/unlink": "routes/activations/unlink.ts",
  "/api/activations/[key]/name": "routes/activations/name.ts",
  "/api/desktop/link": "routes/desktop/link.ts",
  "/api/desktop/renew": "routes/desktop/renew.ts",
};

const MATCHERS = Object.keys(ROUTES).map(
  (pattern) =>
    new RegExp(
      `^${pattern.replace(/\/\[\.\.\.[^\]]+\]/g, "(?:/.*)?").replace(/\[[^\]]+\]/g, "[^/]+")}$`,
    ),
);

/** owns reports whether route belongs to the portal. */
export const owns = (route: string): boolean =>
  MATCHERS.some((matcher) => matcher.test(route));

/** portal adds the signed-in pages, API routes, and Clerk middleware. */
export const portal = (): AstroIntegration => ({
  name: "portal",
  hooks: {
    "astro:config:setup": ({ injectRoute, addMiddleware }) => {
      for (const [pattern, file] of Object.entries(ROUTES))
        injectRoute({ pattern, entrypoint: new URL(file, import.meta.url) });
      addMiddleware({
        entrypoint: new URL("middleware.ts", import.meta.url),
        order: "pre",
      });
    },
  },
});
