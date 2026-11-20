// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { policy } from "@synnaxlabs/site-common/csp";

const CLERK = ["https://*.clerk.accounts.dev", "https://clerk.synnaxlabs.com"];

/** TURNSTILE is Cloudflare's bot check, which Clerk shows at sign-up. */
const TURNSTILE = "https://challenges.cloudflare.com";

/**
 * csp returns the portal's Content Security Policy for a build with the given
 * `VERCEL_ENV`. Astro sends it as a header, so other sites cannot frame the portal.
 */
export const csp = (vercelEnv?: string): ReturnType<typeof policy> =>
  policy({
    sources: {
      "connect-src": ["'self'", ...CLERK, "https://clerk-telemetry.com"],
      "font-src": ["'self'", "data:"],
      "frame-ancestors": ["'none'"],
      "frame-src": [TURNSTILE],
      "img-src": ["'self'", "data:", "https://img.clerk.com"],
      "worker-src": ["'self'", "blob:"],
    },
    scripts: [...CLERK, TURNSTILE],
    posthog: true,
    vercelEnv,
  });
