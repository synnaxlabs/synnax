// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import react from "@astrojs/react";
import vercel from "@astrojs/vercel";
import { CDN } from "@synnaxlabs/site-common/cdn";
import { jitless, policy } from "@synnaxlabs/site-common/csp";
import { integration } from "@synnaxlabs/site-common/integration";
import { defineConfig } from "astro/config";

export default defineConfig({
  integrations: [integration(), react(), jitless()],
  adapter: vercel(),
  security: {
    csp: policy({
      sources: {
        "connect-src": ["'self'", "https://formspree.io"],
        "img-src": ["'self'", "data:"],
        "media-src": ["'self'", CDN],
      },
      posthog: true,
      vercelEnv: process.env.VERCEL_ENV,
    }),
  },
  // Pictures from the CDN are fetched and re-encoded at build time.
  image: { domains: [new URL(CDN).hostname] },
  site: "https://www.synnaxlabs.com",
});
