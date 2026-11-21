// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { math } from "@synnaxlabs/x";
import { type RefCallback, useState } from "react";

import { CSS } from "@/platform/css";

// The most of a long name, in rem, that keeps room from the labels.
const NAME_CAP = 38;

const width = (el: Element | null): number =>
  el == null ? 0 : parseFloat(getComputedStyle(el).width) || 0;

const part = (item: Element, name: string): Element | null =>
  item.querySelector(`:scope > .${CSS.BE("range-item", name)}`);

const ITEM = `.${CSS.B("range-item")}`;
const NAME = `.${CSS.BE("range-item", "name")}`;

const holdsItem = (node: Node): boolean =>
  node instanceof Element && (node.matches(ITEM) || node.querySelector(ITEM) != null);

// Whether a change adds or removes items or an item's cells, or changes a name or a
// parent's path. Other text, such as the duration of a running range, leaves the room
// as it was.
const changesRoom = (r: MutationRecord): boolean => {
  const el = r.target instanceof Element ? r.target : r.target.parentElement;
  if (el?.closest(NAME) != null) return true;
  if (r.type === "childList" && el?.matches(ITEM) === true) return true;
  return [...r.addedNodes, ...r.removedNodes].some(holdsItem);
};

// The width a name cell's content takes in full: the name, and its parent's path where
// the item shows one.
const content = (cell: Element | null): number => {
  if (cell == null) return 0;
  const parts = Array.from(cell.children);
  const gap = parseFloat(getComputedStyle(cell).columnGap) || 0;
  return math.sum(parts.map((p) => p.scrollWidth)) + gap * (parts.length - 1);
};

/**
 * Sets `--console-range-labels-room` on the list to the room its labels column can
 * take: what the items leave once every name has its full width, up to a cap. Refits
 * when the list resizes or its items or their names change.
 */
const fit = (list: HTMLElement | null): void => {
  if (list == null) return;
  const items = Array.from(list.querySelectorAll(ITEM));
  if (items.length === 0) return;
  const [first] = items;
  const cs = getComputedStyle(first);
  const children = Array.from(first.children);
  const flexible = ["indent", "name", "labels"].map((n) => part(first, n));
  const fixed = math.sum(
    children
      .filter((c) => !flexible.includes(c) && getComputedStyle(c).display !== "none")
      .map((c) => width(c)),
  );
  const shown = children.filter((c) => getComputedStyle(c).display !== "none");
  const rest =
    fixed +
    parseFloat(cs.columnGap || "0") * (shown.length - 1) +
    parseFloat(cs.paddingLeft) +
    parseFloat(cs.paddingRight);
  const names = Math.max(
    ...items.map((item) => width(part(item, "indent")) + content(part(item, "name"))),
  );
  const rem = parseFloat(getComputedStyle(document.documentElement).fontSize);
  const room = Math.max(0, width(first) - rest - Math.min(names, NAME_CAP * rem));
  // A change styles every item again, so set it only when it changes.
  const value = `${room}px`;
  if (list.style.getPropertyValue("--console-range-labels-room") !== value)
    list.style.setProperty("--console-range-labels-room", value);
};

/**
 * @returns a ref for a range list. Each list it attaches to keeps its labels room
 * fitted until it unmounts, so a list that mounts again fits again.
 */
export const useLabelsRoom = (): RefCallback<HTMLElement> => {
  // Made before the tags' observer, so it reads widths before the tags write theirs.
  const [resize] = useState(
    () =>
      new ResizeObserver(([entry]) =>
        fit(entry.target.closest<HTMLElement>(`.${CSS.BE("range-tree", "list")}`)),
      ),
  );
  return (list) => {
    if (list == null) return;
    // Fitting as each item first lays out reads widths with no extra layout.
    resize.observe(list);
    const observed = new WeakSet<Element>();
    const observe = (): void =>
      list.querySelectorAll(ITEM).forEach((item) => {
        if (observed.has(item)) return;
        observed.add(item);
        resize.observe(item);
      });
    // A name or a parent's path that loads or changes leaves the items' widths as they
    // were, so only the content tells.
    const mutations = new MutationObserver((records) => {
      if (!records.some(changesRoom)) return;
      observe();
      fit(list);
    });
    mutations.observe(list, { childList: true, subtree: true, characterData: true });
    observe();
    return () => {
      resize.disconnect();
      mutations.disconnect();
    };
  };
};
