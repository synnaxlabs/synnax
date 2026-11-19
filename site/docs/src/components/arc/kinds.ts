// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { hashTarget } from "@/util/hash";

const KIND = "details.arc-kind";

const openHashTarget = (): void => {
  const target = hashTarget();
  if (target instanceof HTMLDetailsElement && target.matches(KIND)) target.open = true;
};

/**
 * Starts opening the Arc diagnostic kind that the URL hash names, and writing the kind
 * that the reader opens to the URL. It follows page swaps by itself.
 *
 * @returns a function that stops listening.
 */
export const start = (): (() => void) => {
  const controller = new AbortController();
  const { signal } = controller;
  openHashTarget();
  document.addEventListener("astro:page-load", openHashTarget, { signal });
  window.addEventListener("hashchange", openHashTarget, { signal });
  // The capture phase is before the router, which then leaves the click alone.
  document.addEventListener(
    "click",
    (e) => {
      if (!(e.target instanceof Element)) return;
      if (e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) return;
      const kind = e.target.closest(`${KIND} > summary a`)?.closest("details");
      if (kind == null) return;
      e.preventDefault();
      kind.open = !kind.open;
    },
    { capture: true, signal },
  );
  // An open kind is not a navigation, so it stays out of the history.
  document.addEventListener(
    "toggle",
    ({ target }) => {
      if (!(target instanceof HTMLDetailsElement) || !target.matches(KIND)) return;
      if (target.open)
        window.history.replaceState(window.history.state, "", `#${target.id}`);
    },
    { capture: true, signal },
  );
  return () => controller.abort();
};
