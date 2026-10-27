// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type AstroIntegration, type HookParameters } from "astro";

const FAVICONS = ["favicon.ico", "favicon.svg"];

export interface Config {
  /** Leaves out the robots.txt, for a site that search engines do not list. */
  unlisted?: boolean;
}

/** Serves the favicons and the robots.txt that every Synnax site shares. */
export const integration = ({ unlisted = false }: Config = {}) =>
  ({
    name: "@synnaxlabs/site-common",
    hooks: {
      "astro:config:setup": ({
        injectRoute,
      }: Pick<HookParameters<"astro:config:setup">, "injectRoute">) => {
        for (const route of unlisted ? FAVICONS : [...FAVICONS, "robots.txt"])
          injectRoute({
            pattern: `/${route}`,
            entrypoint: new URL(`./routes/${route}.ts`, import.meta.url),
            prerender: true,
          });
      },
    },
  }) satisfies AstroIntegration;
