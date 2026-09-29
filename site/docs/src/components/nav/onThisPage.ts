// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { CHANGE_EVENT } from "@/components/tabs/sync";

const ACTIVE_CLASS = "active";
// Marks an outline menu; the value selects the headings its entries link to.
const MENU = "[data-outline]";
// A heading this close to the top of the viewport counts as the one being read.
const READING_LINE = 120;
const READING_LIMIT = 150;

const findReading = (selector: string): string | null => {
  const headings = document.querySelectorAll(selector);
  let reading: Element | null = null;
  let closest = Infinity;
  for (const heading of headings) {
    const { top, height } = heading.getBoundingClientRect();
    const distance = Math.abs(top - READING_LINE);
    if (top <= READING_LIMIT && top > -height && distance < closest) {
      closest = distance;
      reading = heading;
    }
  }
  if (reading == null)
    for (const heading of headings)
      if (heading.getBoundingClientRect().top < READING_LIMIT) reading = heading;
  return reading?.id ?? null;
};

const follow = (menu: HTMLElement, signal: AbortSignal): void => {
  const headings = menu.dataset.outline ?? "";
  const items = [...menu.querySelectorAll<HTMLAnchorElement>("[data-item-key]")];
  const indicator = menu.querySelector<HTMLElement>("[data-outline-indicator]");

  const activate = (id: string | null): void => {
    const active = items.find((item) => item.dataset.itemKey === id);
    if (active == null) return;
    for (const item of items) item.classList.toggle(ACTIVE_CLASS, item === active);
    if (indicator == null) return;
    indicator.style.transform = `translateY(${active.offsetTop}px)`;
    indicator.style.height = `${active.offsetHeight}px`;
    indicator.style.opacity = "1";
  };

  const hideTabbedAway = (): void => {
    for (const item of items) {
      const heading = document.getElementById(item.dataset.itemKey ?? "");
      item.hidden = heading?.closest("[hidden]") != null;
    }
    activate(findReading(headings));
  };

  hideTabbedAway();
  document.addEventListener(CHANGE_EVENT, hideTabbedAway, { signal });
  window.addEventListener("scroll", () => activate(findReading(headings)), {
    passive: true,
    signal,
  });
  menu.addEventListener(
    "click",
    (e) => {
      const item = (e.target as Element).closest<HTMLElement>("[data-item-key]");
      if (item != null) activate(item.dataset.itemKey ?? null);
    },
    { signal },
  );
};

/**
 * Starts marking the heading being read in each outline menu of the page and hiding
 * entries for headings in hidden tab panels. Call it once, before the tabs script
 * starts; it follows page swaps by itself.
 */
export const start = (): void => {
  let bound: AbortController | null = null;

  const bind = (): void => {
    bound?.abort();
    bound = new AbortController();
    const { signal } = bound;
    for (const menu of document.querySelectorAll<HTMLElement>(MENU))
      follow(menu, signal);
  };

  document.addEventListener("astro:after-swap", bind);
  bind();
};
