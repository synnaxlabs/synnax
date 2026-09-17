// Copyright 2026 Synnax Labs, Inc.
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
import clerk from "@clerk/astro";
import { grammar as arcGrammar } from "@synnaxlabs/arc";
import { type AstroUserConfig } from "astro";
import { envField } from "astro/config";

import { portal } from "./src/portal/integration";
import { symbols, theme } from "./src/util/shiki";

const secret = envField.string({ context: "server", access: "secret" });

/**
 * docs is the site without the portal: what the static site check builds. The portal
 * integrations need Clerk keys and a database, which the check has no use for.
 */
export const docs = {
  integrations: [react(), mdx()],
  output: "server",
  adapter: vercel(),
  markdown: {
    shikiConfig: {
      theme,
      langs: [arcGrammar],
      transformers: [symbols],
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
    "/reference/client": "/reference/client/quick-start",
    "/reference/client/advanced": "/reference/client/advanced/auto-commit",
    "/reference/console/clusters": "/reference/console/get-started",
    "/reference/console/cores": "/reference/console/get-started",
    "/reference/console/requirements": "/reference/console/get-started",
    "/reference/console/workspaces": "/reference/console/projects",
    "/reference/control/arc/concepts":
      "/reference/control/arc/concepts/control-authority",
    "/reference/control/arc/how-to": "/reference/control/arc/how-to/data-processing",
    "/reference/driver/timing": "/reference/driver/task-basics",
    "/reference/driver/http/get-started": "/reference/driver/http/connect-server",
    "/reference/driver/labjack/get-started":
      "/reference/driver/labjack/configure-device",
    "/reference/driver/modbus/get-started": "/reference/driver/modbus/connect-server",
    "/reference/driver/ni/get-started": "/reference/driver/ni/configure-device",
    "/reference/driver/opc-ua/get-started": "/reference/driver/opc-ua/connect-server",
    "/reference/driver/pagerduty/get-started": "/reference/driver/pagerduty/alert-task",
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
} satisfies AstroUserConfig;

const config: AstroUserConfig = {
  ...docs,
  integrations: [
    ...docs.integrations,
    clerk({ signInUrl: "/sign-in", signUpUrl: "/sign-up" }),
    portal(),
  ],
  env: {
    schema: {
      DATABASE_URL: secret,
      LICENSE_KMS_KEY_ARN: secret,
      LICENSE_KID: envField.string({
        context: "server",
        access: "public",
        default: "1",
      }),
      STAFF_ORG_ID: secret,
      CLERK_WEBHOOK_SIGNING_SECRET: secret,
      RESEND_API_KEY: secret,
      MAIL_FROM: envField.string({
        context: "server",
        access: "public",
        default: "Synnax Labs <licenses@synnaxlabs.com>",
      }),
      CRON_SECRET: secret,
      LINEAR_API_KEY: secret,
      LINEAR_SUPPORT_TEAM: envField.string({
        context: "server",
        access: "public",
        default: "SUP",
      }),
    },
  },
};

export default config;
