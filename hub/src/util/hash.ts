// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

const SETTLE_MS = 5000;
const READER_SCROLL_EVENTS = ["wheel", "touchstart", "keydown"];

const hashTarget = (): HTMLElement | null => {
  try {
    return document.getElementById(decodeURIComponent(window.location.hash.slice(1)));
  } catch {
    return null;
  }
};

const pin = (): void => {
  const el = hashTarget();
  if (el == null) return;
  const controller = new AbortController();
  const { signal } = controller;
  const observer = new ResizeObserver(() => el.scrollIntoView());
  observer.observe(document.body);
  const stop = () => controller.abort();
  for (const event of READER_SCROLL_EVENTS)
    window.addEventListener(event, stop, { signal });
  document.addEventListener("astro:before-swap", stop, { signal });
  const timer = setTimeout(stop, SETTLE_MS);
  signal.addEventListener("abort", () => {
    observer.disconnect();
    clearTimeout(timer);
  });
};

/**
 * Keeps the URL hash target in view while media above it loads and resizes the page.
 * History traversals are left alone so the browser can restore the reader's position.
 *
 * @returns a function that stops listening for navigations.
 */
export const startPinningHashTarget = (): (() => void) => {
  const controller = new AbortController();
  const { signal } = controller;
  const [entry] = performance.getEntriesByType("navigation");
  if ((entry as PerformanceNavigationTiming | undefined)?.type === "navigate") pin();
  document.addEventListener(
    "astro:before-swap",
    (e) => {
      if ((e as Event & { navigationType?: string }).navigationType === "traverse")
        return;
      document.addEventListener("astro:page-load", pin, { once: true, signal });
    },
    { signal },
  );
  return () => controller.abort();
};
