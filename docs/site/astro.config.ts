// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import mdx from "@astrojs/mdx";
import react from "@astrojs/react";
import vercel from "@astrojs/vercel";
import { grammar as arcGrammar } from "@synnaxlabs/arc";
import { defineConfig } from "astro/config";

const shikiResourcePaths = Object.keys(
  import.meta.glob([
    "../../node_modules/.pnpm/shiki@*/node_modules/shiki/languages/*.tmLanguage.json",
    "../../node_modules/.pnpm/shiki@*/node_modules/shiki/themes/*.json",
  ]),
);

// https://astro.build/config
export default defineConfig({
  integrations: [react(), mdx()],
  output: "server",
  adapter: vercel({ includeFiles: shikiResourcePaths }),
  markdown: {
    shikiConfig: {
      theme: "css-variables",
      langs: [arcGrammar],
    },
  },
  redirects: {
    "/reference/device-drivers/standalone": "/reference/driver/installation",
    "/reference/cluster/[...slug]": "/reference/core/[...slug]",
    "/reference/device-drivers/[...slug]": "/reference/driver/[...slug]",
    "/reference/console/clusters": "/reference/console/cores",
  },
  site: "https://docs.synnaxlabs.com",
});
