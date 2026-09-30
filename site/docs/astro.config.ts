// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { satteri } from "@astrojs/markdown-satteri";
import mdx from "@astrojs/mdx";
import react from "@astrojs/react";
import vercel from "@astrojs/vercel";
import { grammar as arcGrammar } from "@synnaxlabs/arc";
import { layers } from "@synnaxlabs/vite-plugin";
import { defineConfig, envField } from "astro/config";

import { outline } from "./src/util/outline";
import { symbols, theme } from "./src/util/shiki";

// https://astro.build/config
export default defineConfig({
  integrations: [react(), mdx()],
  output: "server",
  env: {
    schema: {
      // Vercel sets this at build time; flags read it to light up preview deploys.
      VERCEL_ENV: envField.string({
        context: "client",
        access: "public",
        optional: true,
      }),
      FLAG_DESKTOP: envField.boolean({
        context: "client",
        access: "public",
        default: false,
      }),
    },
  },
  adapter: vercel({
    // Misses read Vercel's shared page cache, not the function, and pages refresh
    // with the release listing. The updater feeds stay uncached so a failed lookup
    // never sticks.
    isr: { expiration: 300, exclude: [/^\/releases\/[^/]+\/(latest|next)\.json$/] },
  }),
  vite: {
    // Lyra is ESM with CSS imports, which Node cannot load; Vite bundles it for SSR.
    ssr: { noExternal: ["@synnaxlabs/lyra"] },
    css: {
      postcss: { plugins: [layers([{ name: "pluto", files: /[\\/]lyra[\\/]/ }])] },
    },
    // An inlined font ships in the render-blocking stylesheet whether or not a page
    // needs its characters.
    build: {
      assetsInlineLimit: (file) => (/\.woff2?$/.test(file) ? false : undefined),
    },
  },
  markdown: {
    processor: satteri({ hastPlugins: [outline] }),
    shikiConfig: {
      theme,
      langs: [arcGrammar],
      transformers: [symbols],
    },
  },
  redirects: {
    // Shipped clients print these paths in errors and docstrings.
    "/reference/client/resources/troubleshooting": "/reference/client/troubleshooting",
    "/reference/client/working-with-data/streaming-data":
      "/reference/client/stream-data",
    "/concepts/write": "/reference/client/write-data",
    "/guides/comparison/performance/one-billion-rows": "/blog/one-billion-rows",
    "/guides/get-started/installation": "/reference/enterprise",
    "/guides/": "/reference/",
    "/guides/get-started": "/reference/",
    "/guides/analyst": "/reference/",
    "/guides/sys-admin": "/reference/",
    "/guides/operations": "/reference/",
    "/guides/comparison": "/reference/",
    "/reference/device-drivers/standalone": "/reference/enterprise/driver/install",
    "/reference/client": "/reference/client/quick-start",
    "/reference/client/advanced": "/reference/client/advanced/auto-commit",
    "/reference/console/clusters": "/reference/enterprise/console",
    "/reference/console/cores": "/reference/enterprise/console",
    "/reference/console/requirements": "/reference/enterprise/console",
    "/reference/console/channels": "/reference/channels",
    "/reference/console/calculated-channels": "/reference/channels/calculated",
    "/reference/console/ranges": "/reference/ranges/find",
    "/reference/console/workspaces": "/reference/visualize/projects",
    "/reference/control/arc/concepts":
      "/reference/control/arc/concepts/control-authority",
    "/reference/control/arc/how-to": "/reference/control/arc/how-to/data-processing",
    "/reference/driver/timing": "/reference/hardware/tasks",
    "/reference/driver/http/get-started": "/reference/hardware/http/connect-server",
    "/reference/driver/labjack/get-started":
      "/reference/hardware/labjack/configure-device",
    "/reference/driver/modbus/get-started": "/reference/hardware/modbus/connect-server",
    "/reference/driver/ni/get-started": "/reference/hardware/ni/configure-device",
    "/reference/driver/opc-ua/get-started": "/reference/hardware/opc-ua/connect-server",
    "/reference/driver/pagerduty/get-started": "/reference/alarms/pagerduty",
    "/reference/pluto": "/reference/",
    "/reference/pluto/get-started": "/reference/",
    "/reference/pluto/provider-and-canvas": "/reference/",
    "/reference/pluto/theming": "/reference/",
    "/reference/pluto/line-plot": "/reference/",
    "/reference/pluto/example-app": "/reference/",
    "/reference/python-client": "/reference/client/quick-start",
    "/reference/python-client/get-started": "/reference/client/quick-start",
    "/reference/python-client/channels": "/reference/channels/create",
    "/reference/python-client/ranges": "/reference/ranges/create",
    "/reference/python-client/read-data": "/reference/client/read-data",
    "/reference/python-client/write-data": "/reference/client/write-data",
    "/reference/python-client/stream-data": "/reference/client/stream-data",
    "/reference/python-client/delete-data": "/reference/client/advanced/delete-data",
    "/reference/python-client/series-and-frames": "/reference/client/series-and-frames",
    "/reference/python-client/examples": "/reference/client/examples",
    "/reference/python-client/troubleshooting": "/reference/client/troubleshooting",
    "/reference/python-client/device-driver": "/reference/hardware/custom",
    "/reference/typescript-client": "/reference/client/quick-start",
    "/reference/typescript-client/get-started": "/reference/client/quick-start",
    "/reference/typescript-client/channels": "/reference/channels/create",
    "/reference/typescript-client/ranges": "/reference/ranges/create",
    "/reference/typescript-client/read-data": "/reference/client/read-data",
    "/reference/typescript-client/write-data": "/reference/client/write-data",
    "/reference/typescript-client/stream-data": "/reference/client/stream-data",
    "/reference/typescript-client/delete-data":
      "/reference/client/advanced/delete-data",
    "/reference/typescript-client/series-and-frames":
      "/reference/client/series-and-frames",
    "/reference/typescript-client/timestamps": "/reference/client/time-types",
    "/reference/typescript-client/examples": "/reference/client/examples",
    "/reference/typescript-client/troubleshooting": "/reference/client/troubleshooting",
    "/reference/client/advanced/build-device-driver": "/reference/hardware/custom",
    "/reference/client/advanced/write-authorities": "/reference/control/authority",
    "/reference/client/calculated-channels": "/reference/channels/calculated",
    "/reference/client/channels": "/reference/channels/create",
    "/reference/client/ranges": "/reference/ranges/create",
    "/reference/concepts": "/reference/channels",
    "/reference/concepts/channels": "/reference/channels",
    "/reference/concepts/clusters-and-nodes": "/reference/enterprise/clusters",
    "/reference/concepts/overview": "/reference/channels",
    "/reference/concepts/ranges": "/reference/ranges",
    "/reference/concepts/reads": "/reference/client/read-data",
    "/reference/concepts/streams": "/reference/client/stream-data",
    "/reference/concepts/writes": "/reference/client/write-data",
    "/reference/console": "/reference/visualize",
    "/reference/console/get-started": "/reference/enterprise/console",
    "/reference/console/line-plots": "/reference/visualize/line-plots",
    "/reference/console/logs": "/reference/visualize/logs",
    "/reference/console/projects": "/reference/visualize/projects",
    "/reference/console/schematics": "/reference/visualize/schematics",
    "/reference/console/tables": "/reference/visualize/tables",
    "/reference/console/ui-overview": "/reference/get-started/ui-overview",
    "/reference/console/users": "/reference/enterprise/users",
    "/reference/control/arc/how-to/alarms": "/reference/alarms/arc",
    "/reference/core": "/reference/enterprise",
    "/reference/core/cli-reference": "/reference/enterprise/cli-reference",
    "/reference/core/installation": "/reference/enterprise",
    "/reference/core/installing-certificates": "/reference/enterprise/certificates",
    "/reference/core/production": "/reference/enterprise/production",
    "/reference/core/quick-start": "/reference/enterprise",
    "/reference/core/requirements": "/reference/enterprise/requirements",
    "/reference/core/systemd-service": "/reference/enterprise/systemd-service",
    "/reference/driver": "/reference/hardware",
    "/reference/driver/ethercat": "/reference/hardware/ethercat",
    "/reference/driver/ethercat/configure-device":
      "/reference/hardware/ethercat/configure-device",
    "/reference/driver/ethercat/get-started":
      "/reference/hardware/ethercat/get-started",
    "/reference/driver/ethercat/read-task": "/reference/hardware/ethercat/read-task",
    "/reference/driver/ethercat/write-task": "/reference/hardware/ethercat/write-task",
    "/reference/driver/get-started": "/reference/hardware",
    "/reference/driver/http": "/reference/hardware/http",
    "/reference/driver/http/connect-server": "/reference/hardware/http/connect-server",
    "/reference/driver/http/read-task": "/reference/hardware/http/read-task",
    "/reference/driver/http/write-task": "/reference/hardware/http/write-task",
    "/reference/driver/installation": "/reference/enterprise/driver/install",
    "/reference/driver/labjack": "/reference/hardware/labjack",
    "/reference/driver/labjack/configure-device":
      "/reference/hardware/labjack/configure-device",
    "/reference/driver/labjack/read-task": "/reference/hardware/labjack/read-task",
    "/reference/driver/labjack/write-task": "/reference/hardware/labjack/write-task",
    "/reference/driver/modbus": "/reference/hardware/modbus",
    "/reference/driver/modbus/connect-server":
      "/reference/hardware/modbus/connect-server",
    "/reference/driver/modbus/read-task": "/reference/hardware/modbus/read-task",
    "/reference/driver/modbus/write-task": "/reference/hardware/modbus/write-task",
    "/reference/driver/ni": "/reference/hardware/ni",
    "/reference/driver/ni/analog-read-task": "/reference/hardware/ni/analog-read-task",
    "/reference/driver/ni/analog-write-task":
      "/reference/hardware/ni/analog-write-task",
    "/reference/driver/ni/configure-device": "/reference/hardware/ni/configure-device",
    "/reference/driver/ni/counter-read-task":
      "/reference/hardware/ni/counter-read-task",
    "/reference/driver/ni/digital-read-task":
      "/reference/hardware/ni/digital-read-task",
    "/reference/driver/ni/digital-write-task":
      "/reference/hardware/ni/digital-write-task",
    "/reference/driver/opc-ua": "/reference/hardware/opc-ua",
    "/reference/driver/opc-ua/connect-server":
      "/reference/hardware/opc-ua/connect-server",
    "/reference/driver/opc-ua/read-task": "/reference/hardware/opc-ua/read-task",
    "/reference/driver/opc-ua/write-task": "/reference/hardware/opc-ua/write-task",
    "/reference/driver/pagerduty": "/reference/alarms/pagerduty",
    "/reference/driver/pagerduty/alert-task": "/reference/alarms/pagerduty",
    "/reference/driver/task-basics": "/reference/hardware/tasks",
    "/reference/installation": "/reference/enterprise",
  },
  site: "https://docs.synnaxlabs.com",
});
