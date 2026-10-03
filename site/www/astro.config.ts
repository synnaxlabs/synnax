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
import { layers } from "@synnaxlabs/vite-plugin";
import { defineConfig } from "astro/config";

export default defineConfig({
  integrations: [react()],
  output: "server",
  adapter: vercel(),
  vite: {
    // These ship ESM with CSS imports, which Node cannot load; Vite bundles them for
    // SSR.
    ssr: { noExternal: ["@synnaxlabs/lyra", "@synnaxlabs/site-common"] },
    css: {
      postcss: { plugins: [layers([{ name: "pluto", files: /[\\/]lyra[\\/]/ }])] },
    },
  },
  site: "https://www.synnaxlabs.com",
});
