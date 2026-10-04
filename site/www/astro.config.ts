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
import { layers } from "@synnaxlabs/vite-plugin";
import { defineConfig } from "astro/config";

export default defineConfig({
  integrations: [integration(), react()],
  adapter: vercel(),
  // Pictures from the CDN are fetched and re-encoded at build time.
  image: { domains: ["synnax.nyc3.cdn.digitaloceanspaces.com"] },
  vite: {
    // These ship ESM with CSS imports, which Node cannot load; Vite bundles them for
    // SSR.
    ssr: { noExternal: ["@synnaxlabs/lyra", "@synnaxlabs/site-common"] },
    css: {
      postcss: { plugins: [layers([{ name: "pluto", files: /[\\/]lyra[\\/]/ }])] },
    },
    // An inlined font ships in the render-blocking stylesheet whether or not a page
    // needs its characters.
    build: {
      assetsInlineLimit: (file) => (/\.woff2?$/.test(file) ? false : undefined),
    },
  },
  site: "https://www.synnaxlabs.com",
});
