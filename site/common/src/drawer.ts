// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

const setOpen = (open: boolean): void => {
  document.body.classList.toggle("mobile-menu-open", open);
  document
    .querySelector(".mobile-drawer")
    ?.classList.toggle("mobile-drawer--open", open);
  document
    .querySelector(".mobile-overlay")
    ?.classList.toggle("mobile-overlay--open", open);
};

/**
 * Starts the mobile menu: its buttons open and close the drawer, and following a link
 * closes it. Call it once per document; it keeps working across page swaps.
 */
export const start = (): void => {
  document.addEventListener("click", (e) => {
    if (!(e.target instanceof Element)) return;
    if (e.target.closest("[data-mobile-open]") != null) setOpen(true);
    else if (e.target.closest("[data-mobile-close], .mobile-drawer a[href]") != null)
      setOpen(false);
  });
  document.addEventListener("astro:after-swap", () => setOpen(false));
};
