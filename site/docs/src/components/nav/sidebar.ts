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
const SELECTED_CLASS = "pluto--selected";
const CARET = ".pluto-tree__expansion-indicator";

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
  caret?.classList.toggle("pluto--location-bottom", open);
  caret?.classList.toggle("pluto--location-right", !open);
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

/**
 * Starts the reference tree: carets expand and collapse nodes, and the current page
 * stays selected across page swaps. Call it once; the tree persists across swaps.
 */
export const start = (): void => {
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
    for (const tree of document.querySelectorAll(TREE))
      select(tree, window.location.pathname);
  });
};
