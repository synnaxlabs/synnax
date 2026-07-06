// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { unified } from "@astrojs/markdown-remark";
import mdx from "@astrojs/mdx";
import react from "@astrojs/react";
import vercel from "@astrojs/vercel";
import { grammar as arcGrammar } from "@synnaxlabs/arc";
import { defineConfig } from "astro/config";

// https://astro.build/config
export default defineConfig({
  integrations: [react(), mdx()],
  output: "server",
  adapter: vercel(),
  markdown: {
    // Astro 7's native Markdown pipeline (Sätteri) emits highlighted code fences as
    // raw HTML that bypasses the MDX `pre` component override (Block.astro). Opt back
    // into the legacy remark/rehype pipeline so the override keeps applying.
    processor: unified(),
    shikiConfig: {
      theme: "css-variables",
      langs: [arcGrammar],
    },
  },
  redirects: {
    "/guides/comparison/performance/one-billion-rows": "/blog/one-billion-rows",
    "/guides/get-started/installation": "/reference/installation",
    "/guides/": "/reference/",
    "/guides/get-started": "/reference/",
    "/guides/analyst": "/reference/",
    "/guides/sys-admin": "/reference/",
    "/guides/operations": "/reference/",
    "/guides/comparison": "/reference/",
    "/reference/device-drivers/standalone": "/reference/driver/installation",
    "/reference/console/clusters": "/reference/console/cores",
    // Python client redirects
    "/reference/python-client": "/reference/client/quick-start",
    "/reference/python-client/get-started": "/reference/client/quick-start",
    "/reference/python-client/channels": "/reference/client/channels",
    "/reference/python-client/ranges": "/reference/client/ranges",
    "/reference/python-client/read-data": "/reference/client/read-data",
    "/reference/python-client/write-data": "/reference/client/write-data",
    "/reference/python-client/stream-data": "/reference/client/read-data",
    "/reference/python-client/delete-data": "/reference/client/advanced/delete-data",
    "/reference/python-client/series-and-frames": "/reference/client/series-and-frames",
    "/reference/python-client/examples": "/reference/client/examples",
    "/reference/python-client/troubleshooting": "/reference/client/troubleshooting",
    "/reference/python-client/device-driver":
      "/reference/client/advanced/build-device-driver",
    // TypeScript client redirects
    "/reference/typescript-client": "/reference/client/quick-start",
    "/reference/typescript-client/get-started": "/reference/client/quick-start",
    "/reference/typescript-client/channels": "/reference/client/channels",
    "/reference/typescript-client/ranges": "/reference/client/ranges",
    "/reference/typescript-client/read-data": "/reference/client/read-data",
    "/reference/typescript-client/write-data": "/reference/client/write-data",
    "/reference/typescript-client/stream-data": "/reference/client/read-data",
    "/reference/typescript-client/delete-data":
      "/reference/client/advanced/delete-data",
    "/reference/typescript-client/series-and-frames":
      "/reference/client/series-and-frames",
    "/reference/typescript-client/timestamps": "/reference/client/time-types",
    "/reference/typescript-client/examples": "/reference/client/examples",
    "/reference/typescript-client/troubleshooting": "/reference/client/troubleshooting",
  },
  site: "https://docs.synnaxlabs.com",
});
