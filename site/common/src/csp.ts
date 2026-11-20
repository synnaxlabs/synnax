// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createHash } from "node:crypto";

import {
  type AstroIntegration,
  type AstroUserConfig,
  type HookParameters,
} from "astro";

import { API_HOST as POSTHOG } from "./analytics";

type Csp = Exclude<
  NonNullable<AstroUserConfig["security"]>["csp"],
  boolean | undefined
>;

/** Directive names a policy lists sources for. */
type Directive =
  | "connect-src"
  | "font-src"
  | "frame-ancestors"
  | "frame-src"
  | "img-src"
  | "media-src"
  | "worker-src";

const DIRECTIVES: Directive[] = [
  "connect-src",
  "font-src",
  "frame-ancestors",
  "frame-src",
  "img-src",
  "media-src",
  "worker-src",
];

const VERCEL_LIVE = "https://vercel.live";

/** TOOLBAR lists what the Vercel toolbar on a preview deployment loads. */
const TOOLBAR: Record<Directive | "script-src" | "style-src", string[]> = {
  "connect-src": [VERCEL_LIVE, "wss://ws-us3.pusher.com"],
  "font-src": [VERCEL_LIVE, "https://assets.vercel.com"],
  "frame-ancestors": [],
  "frame-src": [VERCEL_LIVE],
  "img-src": [VERCEL_LIVE, "https://vercel.com", "data:", "blob:"],
  "media-src": [],
  "worker-src": [],
  "script-src": [VERCEL_LIVE],
  "style-src": [VERCEL_LIVE],
};

/**
 * POSTHOG_SOURCES lists what PostHog loads. Its session recorder reads `blob:` URLs,
 * which WebKit checks against `connect-src`.
 */
const POSTHOG_SOURCES: Partial<Record<keyof typeof TOOLBAR, string[]>> = {
  "connect-src": [POSTHOG, "blob:"],
  "script-src": [POSTHOG],
};

export interface PolicyConfig {
  /** Sources for each directive. A fetch directive needs `'self'` listed. */
  sources?: Partial<Record<Directive, string[]>>;
  /** Origins that scripts load from, besides `'self'`. */
  scripts?: string[];
  /** Hashes of the `is:inline` scripts, which Astro does not hash. */
  hashes?: `sha256-${string}`[];
  /** Allows the analytics that `Analytics.astro` loads. */
  posthog?: boolean;
  /** Allows every inline style element, not only the hashed ones. */
  inlineStyles?: boolean;
  /** The `VERCEL_ENV` of the build. A preview also allows the Vercel toolbar. */
  vercelEnv?: string;
}

/**
 * policy returns the `security.csp` config of a site. It refuses plugins and other
 * base URLs, and allows inline `style` attributes, which every site sets.
 */
export const policy = ({
  sources = {},
  scripts = [],
  hashes = [],
  posthog = false,
  inlineStyles = false,
  vercelEnv,
}: PolicyConfig): Csp => {
  const toolbar = vercelEnv === "preview";
  const extra = (d: keyof typeof TOOLBAR): string[] => [
    ...(posthog ? (POSTHOG_SOURCES[d] ?? []) : []),
    ...(toolbar ? TOOLBAR[d] : []),
  ];
  const directives: `${Directive} ${string}`[] = [];
  for (const d of DIRECTIVES) {
    const own = sources[d];
    if (own == null && extra(d).length === 0) continue;
    const list = new Set([...(own ?? ["'self'"]), ...extra(d)]);
    directives.push(`${d} ${[...list].join(" ")}`);
  }
  // The toolbar adds style elements that no hash can name.
  const elements = inlineStyles || toolbar;
  return {
    directives: [
      "default-src 'self'",
      ...directives,
      "object-src 'none'",
      "base-uri 'self'",
    ],
    scriptDirective: {
      resources: ["'self'", ...scripts, ...extra("script-src")],
      hashes: posthog ? [...hashes, hash(LOADER)] : hashes,
    },
    styleDirective: {
      resources: elements
        ? ["'self'", "'unsafe-inline'", ...extra("style-src")]
        : ["'self'", { resource: "'unsafe-inline'", kind: "attribute" }],
    },
  };
};

/** jitless stops Zod's probe for eval, which the policy refuses, before islands run. */
export const jitless = () =>
  ({
    name: "@synnaxlabs/site-common/jitless",
    hooks: {
      "astro:config:setup": ({
        injectScript,
      }: Pick<HookParameters<"astro:config:setup">, "injectScript">) =>
        injectScript(
          "before-hydration",
          'import { z } from "zod"; z.config({ jitless: true });',
        ),
    },
  }) satisfies AstroIntegration;

/** hash returns the CSP source that lets an inline script or style with `text` run. */
export const hash = (text: string): `sha256-${string}` =>
  `sha256-${createHash("sha256").update(text).digest("base64")}`;

/**
 * LOADER defines `window.posthog` as a queue before any page script runs. It is
 * PostHog's loader, rewritten to add its queueing methods at once instead of in init.
 * The page waits to init, and a call made meanwhile is queued, not thrown.
 */
export const LOADER = `(function (posthog) {
  if (posthog.__SV) return;
  window.posthog = posthog;
  posthog._i = [];
  posthog.people = [];
  const queue = (name) => {
    posthog[name] = (...args) => posthog.push([name, ...args]);
  };
  ["capture", "identify", "opt_in_capturing", "opt_out_capturing", "reset"].forEach(
    queue,
  );
  posthog.init = (token, config) => {
    const script = document.createElement("script");
    script.async = true;
    script.src = \`\${config.api_host}/static/array.js\`;
    document.head.append(script);
    queue("init");
    posthog._i.push([token, config]);
  };
  posthog.__SV = 1;
})(window.posthog || []);`;
