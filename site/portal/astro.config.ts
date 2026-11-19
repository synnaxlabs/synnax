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
import { HASHES, POSTHOG, VERCEL_LIVE } from "@synnaxlabs/site-common/csp";
import { layers, lyraLayers } from "@synnaxlabs/vite-plugin";
import { type AstroIntegration } from "astro";
import { defineConfig, envField } from "astro/config";
import path from "path";

const secret = envField.string({ context: "server", access: "secret" });

const CLERK = "https://*.clerk.accounts.dev https://clerk.synnaxlabs.com";
/** TURNSTILE is Cloudflare's bot check, which Clerk shows at sign-up. */
const TURNSTILE = "https://challenges.cloudflare.com";

/** jitless stops Zod's probe for eval, which the CSP refuses, before islands load. */
const jitless: AstroIntegration = {
  name: "jitless",
  hooks: {
    "astro:config:setup": ({ injectScript }) =>
      injectScript(
        "before-hydration",
        'import { z } from "zod"; z.config({ jitless: true });',
      ),
  },
};

export default defineConfig({
  integrations: [
    react(),
    clerk({ signInUrl: "/login", signUpUrl: "/sign-up" }),
    jitless,
  ],
  output: "server",
  adapter: vercel(),
  security: {
    // Astro's check refuses the Desktop renewal. src/pipeline.ts checks origins
    // instead.
    checkOrigin: false,
    csp: {
      directives: [
        "default-src 'self'",
        `connect-src 'self' ${CLERK} https://clerk-telemetry.com ${POSTHOG} ${VERCEL_LIVE}`,
        "font-src 'self' data:",
        "worker-src 'self' blob:",
        `img-src 'self' data: https://img.clerk.com ${VERCEL_LIVE} https://vercel.com`,
        `frame-src ${TURNSTILE} ${VERCEL_LIVE}`,
        "frame-ancestors 'none'",
        "object-src 'none'",
        "base-uri 'self'",
      ],
      scriptDirective: {
        resources: ["'self'", CLERK, TURNSTILE, POSTHOG, VERCEL_LIVE],
        hashes: HASHES,
      },
      // Components set sizes and custom properties in style attributes.
      styleDirective: {
        resources: ["'self'", { resource: "'unsafe-inline'", kind: "attribute" }],
      },
    },
  },
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
