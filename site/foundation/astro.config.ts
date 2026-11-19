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
import { integration } from "@synnaxlabs/site-common/integration";
import { defineConfig } from "astro/config";

/** VERCEL_LIVE is the Vercel toolbar, which Vercel adds to preview deployments. */
const VERCEL_LIVE = "https://vercel.live";

export default defineConfig({
  integrations: [integration(), react()],
  adapter: vercel(),
  security: {
    csp: {
      directives: [
        "default-src 'self'",
        `connect-src 'self' https://formspree.io ${VERCEL_LIVE}`,
        `img-src 'self' data: ${VERCEL_LIVE} https://vercel.com`,
        `frame-src ${VERCEL_LIVE}`,
        "object-src 'none'",
        "base-uri 'self'",
      ],
      scriptDirective: { resources: ["'self'", VERCEL_LIVE] },
      // Components set sizes and custom properties in style attributes.
      styleDirective: {
        resources: ["'self'", { resource: "'unsafe-inline'", kind: "attribute" }],
      },
    },
  },
  vite: {
    // Prebundle the island's linked workspace modules so their icon dependencies
    // are ready before the browser hydrates it.
    optimizeDeps: {
      include: [
        "@synnaxlabs/lyra/button",
        "@synnaxlabs/lyra/icon",
        "@synnaxlabs/lyra/logo",
      ],
    },
  },
  site: "https://foundation.synnaxlabs.com",
});
