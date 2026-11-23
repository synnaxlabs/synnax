// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { hash } from "@synnaxlabs/site-common/csp";

import { STORAGE_KEY } from "@/components/nav/collapse";

/**
 * MARK marks the reader's platform on the root before the body paints, so keys print in
 * its own symbols. A platform the URL names wins over the browser's own. The client
 * router swaps in the next page's root attributes before the tabs read the mark, so the
 * mark goes on the next page before the swap.
 */
export const MARK = `(() => {
  const agent = navigator.userAgent.toLowerCase();
  const own = agent.includes("mac")
    ? "macOS"
    : agent.includes("win")
      ? "Windows"
      : agent.includes("linux")
        ? "Linux"
        : null;
  const mark = (root, url) => {
    const named = url.searchParams.get("platform");
    const os = ["macOS", "Windows", "Linux"].includes(named) ? named : own;
    if (os != null) root.dataset.os = os;
  };
  mark(document.documentElement, new URL(location.href));
  document.addEventListener("astro:before-swap", (e) =>
    mark(e.newDocument.documentElement, e.to),
  );
})();`;

/**
 * COLLAPSE applies a remembered sidebar collapse before the body parses. A script in
 * the body would let the browser paint a half-loaded page that then shifts.
 */
export const COLLAPSE = `try {
  if (localStorage.getItem(${JSON.stringify(STORAGE_KEY)}) === "true")
    document.documentElement.dataset.pageNavCollapsed = "";
} catch {}`;

/**
 * DOWNLOADS keeps only the download buttons of the reader's platform, before paint,
 * since the CDN serves every browser the same page. It must follow the buttons. An
 * unknown agent, or a phone or tablet that can install neither, keeps every platform:
 * iPhones say "like Mac OS X" and iPads may pose as a Macintosh, so mobile is checked
 * first. The client router skips an inline script whose text already ran, so the
 * script needs `data-astro-rerun`.
 */
export const DOWNLOADS = `(() => {
  const agent = navigator.userAgent;
  if (/iPhone|iPad|Android|Mobile/.test(agent)) return;
  const platforms = [
    ...document.currentScript.previousElementSibling.querySelectorAll("[data-agent]"),
  ];
  const matches = (platform) => new RegExp(platform.dataset.agent).test(agent);
  if (!platforms.some(matches)) return;
  platforms.forEach((platform) => {
    if (!matches(platform)) platform.remove();
  });
})();`;

/**
 * HASHES let the inline scripts run. The client router keeps the first page's policy,
 * so every page allows them all.
 */
export const HASHES = [MARK, COLLAPSE, DOWNLOADS].map(hash);
