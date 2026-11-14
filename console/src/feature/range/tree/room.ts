// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type RefObject, useLayoutEffect } from "react";

import { CSS } from "@/platform/css";

// The most of a long name, in rem, that keeps room from the labels.
const NAME_CAP = 38;

const width = (el: Element | null): number =>
  el == null ? 0 : parseFloat(getComputedStyle(el).width) || 0;

const part = (item: Element, name: string): Element | null =>
  item.querySelector(`:scope > .${CSS.BE("range-item", name)}`);

/**
 * Sets `--console-range-labels-room` on the list to the room its labels column can
 * take: what the items leave once every name has its full width, up to a cap. Refits
 * when the list resizes or its items change.
 */
export const useLabelsRoom = (ref: RefObject<HTMLElement | null>): void =>
  useLayoutEffect(() => {
    const list = ref.current;
    if (list == null) return;
    let frame = 0;
    const fit = (): void => {
      cancelAnimationFrame(frame);
      frame = 0;
      const items = Array.from(list.querySelectorAll(`.${CSS.B("range-item")}`));
      if (items.length === 0) return;
      const [first] = items;
      const cs = getComputedStyle(first);
      const children = Array.from(first.children);
      const flexible = ["indent", "name", "labels"].map((n) => part(first, n));
      const fixed = children
        .filter((c) => !flexible.includes(c) && getComputedStyle(c).display !== "none")
        .reduce((total, c) => total + width(c), 0);
      const shown = children.filter((c) => getComputedStyle(c).display !== "none");
      const rest =
        fixed +
        parseFloat(cs.columnGap || "0") * (shown.length - 1) +
        parseFloat(cs.paddingLeft) +
        parseFloat(cs.paddingRight);
      const names = Math.max(
        ...items.map(
          (item) =>
            width(part(item, "indent")) +
            (part(item, "name")?.firstElementChild?.scrollWidth ?? 0),
        ),
      );
      const rem = parseFloat(getComputedStyle(document.documentElement).fontSize);
      const room = width(first) - rest - Math.min(names, NAME_CAP * rem);
      list.style.setProperty("--console-range-labels-room", `${room}px`);
    };
    const request = (): void => {
      if (frame === 0) frame = requestAnimationFrame(fit);
    };
    // A resize refits before the frame paints.
    const resize = new ResizeObserver(fit);
    resize.observe(list);
    const mutations = new MutationObserver(request);
    mutations.observe(list, { childList: true, subtree: true, characterData: true });
    request();
    return () => {
      cancelAnimationFrame(frame);
      resize.disconnect();
      mutations.disconnect();
    };
  }, [ref]);
