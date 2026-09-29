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
import { defineConfig, envField } from "astro/config";

import { layer } from "./src/util/layer";

const secret = envField.string({ context: "server", access: "secret" });

export default defineConfig({
  integrations: [react(), clerk({ signInUrl: "/sign-in", signUpUrl: "/sign-up" })],
  output: "server",
  adapter: vercel(),
  // Lyra is ESM with CSS imports, which Node cannot load; Vite bundles it for SSR.
  vite: {
    ssr: { noExternal: ["@synnaxlabs/lyra"] },
    // Lyra's CSS sits in a lower cascade layer, so portal styles override it without
    // out-specifying its selectors.
    css: { postcss: { plugins: [layer(/[\\/]lyra[\\/]/, "pluto")] } },
  },
  env: {
    schema: {
      DATABASE_URL: secret,
      LICENSE_KMS_KEY_ARN: secret,
      LICENSE_KID: envField.string({
        context: "server",
        access: "public",
        default: "2",
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
    },
  },
  site: "https://portal.synnaxlabs.com",
});
