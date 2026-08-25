// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, xy } from "@synnaxlabs/x";
import { type RefObject, useCallback, useEffect } from "react";

import { useSyncedRef } from "@/hooks";

/**
 * Marks an element as the logical parent of a portaled subtree. Set it on the element
 * a portal renders from, and set {@link PORTAL_OWNER_ATTR} to the same value on the
 * portaled element.
 */
export const PORTAL_ID_ATTR = "data-portal-id";

/** Points a portaled element at the {@link PORTAL_ID_ATTR} of its logical parent. */
export const PORTAL_OWNER_ATTR = "data-portal-owner";

/**
 * Whether the target sits inside el, following portal ownership links so a portaled
 * subtree counts as inside the element it renders from.
 */
const contains = (el: HTMLElement, target: Node | null): boolean => {
  let node: Node | null = target;
  while (node != null) {
    if (el.contains(node)) return true;
    const element = node instanceof Element ? node : node.parentElement;
    const portal = element?.closest(`[${PORTAL_OWNER_ATTR}]`);
    if (portal == null) return false;
    const owner = portal.getAttribute(PORTAL_OWNER_ATTR);
    node = document.querySelector(`[${PORTAL_ID_ATTR}="${owner}"]`);
  }
  return false;
};

/** Props for {@link useClickOutside}. */
export interface UseClickOutsideProps {
  ref: RefObject<HTMLElement | null>;
  /** Elements, or a predicate over the event, whose clicks do not count as outside. */
  exclude?: Array<RefObject<HTMLElement>> | ((e: MouseEvent) => boolean);
  onClickOutside: () => void;
}

/**
 * Calls back on a click outside the element, treating a portaled subtree as inside the
 * element it renders from. Clicks past the viewport edge, such as on a scrollbar, do
 * not count.
 */
export const useClickOutside = ({
  ref,
  onClickOutside,
  exclude,
}: UseClickOutsideProps): void => {
  const excludeRef = useSyncedRef(exclude);
  const handleClickOutside = useCallback(
    (e: MouseEvent): void => {
      const el = ref.current;
      const windowBox = box.construct(window.document.documentElement);
      const pos = xy.construct(e);

      const exclude = excludeRef.current;
      if (exclude != null)
        if (typeof exclude === "function") {
          if (exclude(e)) return;
        } else if (exclude.some((r) => r.current?.contains(e.target as Node))) return;

      if (
        el == null ||
        contains(el, e.target as Node) ||
        box.contains(el, pos) ||
        !box.contains(windowBox, pos)
      )
        return;
      onClickOutside();
    },
    [onClickOutside],
  );
  // pointerdown instead of mousedown: Tauri's drag-region script calls
  // stopImmediatePropagation on mousedown, which would swallow dismissal
  // clicks landing on window drag regions.
  useEffect(() => {
    document.addEventListener("pointerdown", handleClickOutside);
    return () => document.removeEventListener("pointerdown", handleClickOutside);
  }, [handleClickOutside]);
};
