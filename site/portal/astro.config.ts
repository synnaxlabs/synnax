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
import clerk from "@clerk/astro";
import { layers, lyraLayers } from "@synnaxlabs/vite-plugin";
import { defineConfig, envField } from "astro/config";
import path from "path";

const secret = envField.string({ context: "server", access: "secret" });

export default defineConfig({
  integrations: [react(), clerk({ signInUrl: "/login", signUpUrl: "/sign-up" })],
  output: "server",
  adapter: vercel(),
  // Astro's check refuses the Desktop renewal. src/pipeline.ts checks origins instead.
  security: { checkOrigin: false },
  // These ship ESM with CSS imports, which Node cannot load; Vite bundles them for SSR.
  vite: {
    ssr: { noExternal: ["@synnaxlabs/lyra", "@synnaxlabs/site-common"] },
    css: {
      postcss: {
        plugins: [layers(lyraLayers(path.resolve(import.meta.dirname, "../..")))],
      },
    },
  },
  env: {
    schema: {
      DATABASE_URL: secret,
      AWS_ROLE_ARN: envField.string({
        context: "server",
        access: "secret",
        optional: true,
      }),
      LICENSE_KMS_KEY_ARN: secret,
      LICENSE_KID: envField.string({
        context: "server",
        access: "public",
        default: "2",
      }),
      STAFF_ORG_ID: secret,
      CLERK_WEBHOOK_SIGNING_SECRET: secret,
      RESEND_API_KEY: envField.string({
        context: "server",
        access: "secret",
        optional: true,
      }),
      MAIL_FROM: envField.string({
        context: "server",
        access: "public",
        default: "Synnax Labs <licenses@synnaxlabs.com>",
      }),
      CRON_SECRET: secret,
      POSTHOG_API_KEY: envField.string({
        context: "server",
        access: "secret",
        optional: true,
      }),
      SLACK_WEBHOOK_URL: envField.string({
        context: "server",
        access: "secret",
        optional: true,
      }),
      VERCEL_ENV: envField.enum({
        context: "server",
        access: "secret",
        values: ["production", "preview", "development"],
        optional: true,
      }),
    },
  },
  site: "https://portal.synnaxlabs.com",
});
