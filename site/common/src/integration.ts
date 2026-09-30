// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type AstroIntegration } from "astro";

const ROUTES = ["favicon.ico", "favicon.svg", "robots.txt"];

/** Serves the favicons and the robots.txt that every Synnax site shares. */
export const integration = (): AstroIntegration => ({
  name: "@synnaxlabs/site-common",
  hooks: {
    "astro:config:setup": ({ injectRoute }) => {
      for (const route of ROUTES)
        injectRoute({
          pattern: `/${route}`,
          entrypoint: new URL(`./routes/${route}.ts`, import.meta.url),
          prerender: true,
        });
    },
  },
});
