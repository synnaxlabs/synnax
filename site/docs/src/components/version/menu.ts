// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { track } from "@/components/version/links";

/**
 * Starts the version menus that `selector` matches. A menu closes on a page load, on a
 * click outside it, and on Escape, and its links follow the reader's place in the page.
 */
export const start = (selector: string): void => {
  const close = (except?: EventTarget | null): void => {
    for (const menu of document.querySelectorAll<HTMLDetailsElement>(selector))
      if (!(except instanceof Node && menu.contains(except))) menu.open = false;
  };
  track(`${selector} a[data-origin]`);
  document.addEventListener("astro:page-load", () => close());
  document.addEventListener("click", ({ target }) => close(target));
  document.addEventListener("keydown", ({ key }) => {
    if (key === "Escape") close();
  });
};
