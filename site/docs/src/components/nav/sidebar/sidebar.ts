// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { normalizeRoute } from "@/util/route";

const TREE = ".reference-tree";
const ITEM = '[role="treeitem"]';
const SELECTED_CLASS = "lyra--selected";
const CARET = ".lyra-tree__expansion-indicator";
const SCROLL_KEY = "page-nav-scroll";

const level = (item: Element): number => Number(item.getAttribute("aria-level"));

const expanded = (item: Element): boolean =>
  item.getAttribute("aria-expanded") === "true";

// Items list in reading order, so a node's descendants are the deeper items after it.
const layout = (items: HTMLElement[]): void => {
  let collapsedAt = Infinity;
  for (const item of items) {
    const depth = level(item);
    if (depth <= collapsedAt) collapsedAt = Infinity;
    item.hidden = depth > collapsedAt;
    if (!item.hidden && item.getAttribute("aria-expanded") === "false")
      collapsedAt = depth;
  }
};

const setExpanded = (item: Element, open: boolean): void => {
  item.setAttribute("aria-expanded", String(open));
  const caret = item.querySelector(CARET);
  caret?.classList.toggle("lyra--location-bottom", open);
  caret?.classList.toggle("lyra--location-right", !open);
};

const select = (tree: Element, path: string): void => {
  const items = [...tree.querySelectorAll<HTMLElement>(ITEM)];
  const target = normalizeRoute(path);
  let current = items.findIndex(
    (item) => normalizeRoute(item.getAttribute("href") ?? "") === target,
  );
  items.forEach((item, i) => {
    item.setAttribute("aria-selected", String(i === current));
    item.classList.toggle(SELECTED_CLASS, i === current);
  });
  if (current === -1) return layout(items);
  if (items[current].hasAttribute("aria-expanded")) setExpanded(items[current], true);
  // Walks back to open each ancestor of the current page.
  for (let i = current - 1; i >= 0; i--)
    if (level(items[i]) < level(items[current])) {
      setExpanded(items[i], true);
      current = i;
    }
  layout(items);
};

// Centers the selected item when it lies outside the part of the tree in view.
const reveal = (tree: Element): void => {
  const item = tree.querySelector(`.${SELECTED_CLASS}`);
  if (item == null) return;
  const view = tree.getBoundingClientRect();
  const box = item.getBoundingClientRect();
  if (box.top >= view.top && box.bottom <= view.bottom) return;
  tree.scrollTop += box.top - view.top - (view.height - box.height) / 2;
};

// Keeps the scroll of the page nav across full page loads in one tab.
const persistScroll = (tree: Element): void => {
  let saved: string | null;
  try {
    saved = sessionStorage.getItem(SCROLL_KEY);
  } catch {
    // Blocked storage only costs the page nav its scroll on the next page.
    return;
  }
  if (saved != null) tree.scrollTop = parseFloat(saved);
  let timer: ReturnType<typeof setTimeout> | null = null;
  tree.addEventListener("scroll", () => {
    if (timer != null) clearTimeout(timer);
    timer = setTimeout(
      () => sessionStorage.setItem(SCROLL_KEY, String(tree.scrollTop)),
      100,
    );
  });
};

/**
 * Starts the reference trees: carets expand and collapse nodes, the current page stays
 * selected across page swaps, and a current page out of view scrolls to the center of
 * its tree. Call it once; the trees persist across swaps.
 */
export const start = (): void => {
  const pageNav = document.querySelector(`.page-nav ${TREE}`);
  if (pageNav != null) persistScroll(pageNav);
  for (const tree of document.querySelectorAll(TREE)) reveal(tree);
  document.addEventListener("click", (e) => {
    if (!(e.target instanceof Element)) return;
    const item = e.target.closest<HTMLElement>(`${TREE} ${ITEM}[aria-expanded]`);
    const tree = item?.closest(TREE);
    if (item == null || tree == null || level(item) === 1) return;
    // A caret or a node without a page toggles; a node with a page navigates.
    if (e.target.closest(CARET) == null && item.hasAttribute("href")) return;
    e.preventDefault();
    setExpanded(item, !expanded(item));
    layout([...tree.querySelectorAll<HTMLElement>(ITEM)]);
  });
  document.addEventListener("astro:after-swap", () => {
    for (const tree of document.querySelectorAll(TREE)) {
      select(tree, window.location.pathname);
      reveal(tree);
    }
  });
};
