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
import { HASHES, POSTHOG, VERCEL_LIVE } from "@synnaxlabs/site-common/csp";
import { integration } from "@synnaxlabs/site-common/integration";
import { defineConfig } from "astro/config";

export default defineConfig({
  integrations: [integration(), react()],
  adapter: vercel(),
  security: {
    csp: {
      directives: [
        "default-src 'self'",
        `connect-src 'self' ${POSTHOG} https://formspree.io ${VERCEL_LIVE}`,
        `img-src 'self' data: ${VERCEL_LIVE} https://vercel.com`,
        `media-src 'self' ${CDN}`,
        `frame-src ${VERCEL_LIVE}`,
        "object-src 'none'",
        "base-uri 'self'",
      ],
      scriptDirective: { resources: ["'self'", POSTHOG, VERCEL_LIVE], hashes: HASHES },
      // Shiki colors code with style attributes.
      styleDirective: {
        resources: ["'self'", { resource: "'unsafe-inline'", kind: "attribute" }],
      },
    },
  },
  // Pictures from the CDN are fetched and re-encoded at build time.
  image: { domains: [new URL(CDN).hostname] },
  site: "https://www.synnaxlabs.com",
});
