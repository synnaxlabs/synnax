// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/** Adds modal keyboard behavior to the shared site's drawer. */
export const enhanceMenu = (): void => {
  const drawer = document.querySelector<HTMLElement>(".mobile-drawer");
  const trigger = document.querySelector<HTMLButtonElement>("[data-mobile-open]");
  const close = drawer?.querySelector<HTMLButtonElement>("[data-mobile-close]");
  if (drawer == null || trigger == null || close == null) return;
  const background = document.querySelectorAll<HTMLElement>(
    ".site-header, main, footer",
  );
  drawer.id = "foundation-menu";
  drawer.setAttribute("role", "dialog");
  drawer.setAttribute("aria-label", "Foundation navigation");
  drawer.setAttribute("aria-modal", "true");
  drawer.querySelector(".logo-link")?.setAttribute("aria-label", "Foundation home");
  trigger.setAttribute("aria-controls", drawer.id);

  const sync = (): void => {
    const open = drawer.classList.contains("mobile-drawer--open");
    drawer.inert = !open;
    drawer.setAttribute("aria-hidden", String(!open));
    trigger.setAttribute("aria-expanded", String(open));
    background.forEach((element) => {
      element.inert = open;
    });
    if (open) close.focus();
    else if (drawer.contains(document.activeElement)) trigger.focus();
  };
  sync();
  new MutationObserver(sync).observe(drawer, {
    attributes: true,
    attributeFilter: ["class"],
  });
  drawer.addEventListener("keydown", (event) => {
    if (event.key === "Escape") {
      event.preventDefault();
      close.click();
    }
    if (event.key !== "Tab") return;
    const focusable = Array.from(
      drawer.querySelectorAll<HTMLElement>("a[href], button"),
    );
    const first = focusable[0];
    const last = focusable.at(-1);
    if (event.shiftKey && document.activeElement === first) {
      event.preventDefault();
      last?.focus();
    } else if (!event.shiftKey && document.activeElement === last) {
      event.preventDefault();
      first?.focus();
    }
  });
  window.matchMedia("(min-width: 801px)").addEventListener("change", (event) => {
    if (event.matches) close.click();
  });
};
