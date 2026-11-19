// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type AstroIntegration, type HookParameters } from "astro";

const ROUTES = ["favicon.ico", "favicon.svg", "robots.txt"];

const TYPES = `declare namespace Astro {
  interface ClientDirectives {
    "client:interaction"?: boolean | string;
  }
}
`;

/**
 * Serves the favicons and the robots.txt that every Synnax site shares, and the
 * `/consent` route that the analytics banner asks. Adds the `client:interaction`
 * directive, which hydrates an island on its first click.
 */
export const integration = () =>
  ({
    name: "@synnaxlabs/site-common",
    hooks: {
      "astro:config:setup": ({
        injectRoute,
        addClientDirective,
      }: Pick<
        HookParameters<"astro:config:setup">,
        "injectRoute" | "addClientDirective"
      >) => {
        for (const route of ROUTES)
          injectRoute({
            pattern: `/${route}`,
            entrypoint: new URL(`./routes/${route}.ts`, import.meta.url),
            prerender: true,
          });
        injectRoute({
          pattern: "/consent",
          entrypoint: new URL("./routes/consent.ts", import.meta.url),
          prerender: false,
        });
        addClientDirective({
          name: "interaction",
          entrypoint: new URL("./interaction.ts", import.meta.url),
        });
      },
      "astro:config:done": ({
        injectTypes,
      }: Pick<HookParameters<"astro:config:done">, "injectTypes">) => {
        injectTypes({ filename: "types.d.ts", content: TYPES });
      },
    },
  }) satisfies AstroIntegration;
