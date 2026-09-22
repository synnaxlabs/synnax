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
 * ROUTES maps every portal URL to its source file. The files live outside
 * `src/pages` so the static site check never builds or crawls a session-bound page.
 */
const ROUTES: Record<string, string> = {
  "/sign-in/reset": "pages/reset.astro",
  "/sign-in/[...rest]": "pages/sign-in.astro",
  "/sign-up/[...rest]": "pages/sign-up.astro",
  "/sso-callback": "pages/sso-callback.astro",
  "/portal": "pages/licenses/index.astro",
  "/portal/licenses/activate": "pages/licenses/activate.astro",
  "/portal/licenses/[key]": "pages/licenses/[key].astro",
  "/portal/account": "pages/account.astro",
  "/portal/staff/licenses": "pages/staff/licenses.astro",
  "/desktop/sign-in": "pages/desktop/sign-in.astro",
  "/api/webhooks/clerk": "routes/webhooks/clerk.ts",
  "/api/cron/expiry": "routes/cron/expiry.ts",
  "/api/portal/licenses": "routes/licenses/issue.ts",
  "/api/portal/licenses/[key]": "routes/licenses/amend.ts",
  "/api/portal/licenses/[key]/activate": "routes/licenses/activate.ts",
  "/api/portal/licenses/[key]/revoke": "routes/licenses/revoke.ts",
  "/api/portal/licenses/[key]/floating": "routes/licenses/floating.ts",
  "/api/portal/activations/[key]/token": "routes/activations/token.ts",
  "/api/portal/activations/[key]/release": "routes/activations/release.ts",
  "/api/portal/activations/[key]/unlink": "routes/activations/unlink.ts",
  "/api/portal/desktop/link": "routes/desktop/link.ts",
  "/api/portal/desktop/renew": "routes/desktop/renew.ts",
};

/** portal adds the signed-in portal pages, API routes, and Clerk middleware. */
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
