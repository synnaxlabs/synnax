// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createHash } from "node:crypto";

export { API_HOST as POSTHOG } from "./analytics";

/**
 * VERCEL_LIVE is the origin of the Vercel toolbar, which Vercel adds to preview
 * deployments.
 */
export const VERCEL_LIVE = "https://vercel.live";

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

/** HASHES let the inline scripts of the shared components run. */
export const HASHES = [hash(LOADER)];
