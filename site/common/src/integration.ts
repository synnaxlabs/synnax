// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import path from "node:path";

import { layers, lyraLayers } from "@synnaxlabs/vite-plugin";
import type { AstroIntegration, AstroUserConfig, HookParameters } from "astro";

const ROUTES = ["favicon.ico", "favicon.svg", "robots.txt"];

const TYPES = `declare namespace Astro {
  interface ClientDirectives {
    "client:interaction"?: boolean | string;
  }
}
`;

const REPO = path.resolve(import.meta.dirname, "../../..");

/**
 * Serves the favicons and the robots.txt that every Synnax site shares, and the
 * `/consent` route that the analytics banner asks. Adds the `client:interaction`
 * directive, which hydrates an island on its first click. Sets the Vite options every
 * site needs to build Lyra.
 */
export const integration = () =>
  ({
    name: "@synnaxlabs/site-common",
    hooks: {
      "astro:config:setup": ({
        injectRoute,
        addClientDirective,
        updateConfig,
      }: Pick<
        HookParameters<"astro:config:setup">,
        "injectRoute" | "addClientDirective"
      > & {
        updateConfig: (config: Pick<AstroUserConfig, "vite">) => unknown;
      }) => {
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
        updateConfig({
          vite: {
            // These ship ESM with CSS imports, which Node cannot load; Vite bundles
            // them for SSR.
            ssr: { noExternal: ["@synnaxlabs/lyra", "@synnaxlabs/site-common"] },
            css: { postcss: { plugins: [layers(lyraLayers(REPO))] } },
            // An inlined font ships in the render-blocking stylesheet whether or not a
            // page needs its characters.
            build: {
              assetsInlineLimit: (file) => (/\.woff2?$/.test(file) ? false : undefined),
            },
          },
        });
      },
      "astro:config:done": ({
        injectTypes,
      }: Pick<HookParameters<"astro:config:done">, "injectTypes">) => {
        injectTypes({ filename: "types.d.ts", content: TYPES });
      },
    },
  }) satisfies AstroIntegration;
