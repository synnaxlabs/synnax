// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type RefCallback, useCallback, useRef } from "react";

const LINE_HEIGHT = 16;

const deltaPixels = (e: WheelEvent, pageSize: number): number => {
  switch (e.deltaMode) {
    case WheelEvent.DOM_DELTA_LINE:
      return e.deltaY * LINE_HEIGHT;
    case WheelEvent.DOM_DELTA_PAGE:
      return e.deltaY * pageSize;
    default:
      return e.deltaY;
  }
};

const scrollsY = (el: Element, delta: number): boolean => {
  const { overflowY } = getComputedStyle(el);
  if (overflowY !== "auto" && overflowY !== "scroll") return false;
  if (delta < 0) return el.scrollTop > 0;
  return Math.ceil(el.scrollTop + el.clientHeight) < el.scrollHeight;
};

/**
 * Makes a vertical mouse wheel scroll an element horizontally. The wheel passes
 * through when Ctrl or Cmd is held, when the gesture is already horizontal, when a
 * scroller inside the element can still move vertically, or when the element is at
 * its edge.
 *
 * @returns a ref callback to attach to the horizontally scrolling element.
 */
export const useWheelScrollX = <E extends HTMLElement>(): RefCallback<E> => {
  const detachRef = useRef<(() => void) | null>(null);
  return useCallback((el: E | null) => {
    detachRef.current?.();
    detachRef.current = null;
    if (el == null) return;
    const handleWheel = (e: WheelEvent): void => {
      if (e.ctrlKey || e.metaKey || Math.abs(e.deltaX) > Math.abs(e.deltaY)) return;
      const delta = deltaPixels(e, el.clientWidth);
      for (
        let node = e.target instanceof Element ? e.target : null;
        node != null && node !== el;
        node = node.parentElement
      )
        if (scrollsY(node, delta)) return;
      const max = Math.max(el.scrollWidth - el.clientWidth, 0);
      const next = Math.min(Math.max(el.scrollLeft + delta, 0), max);
      if (next === el.scrollLeft) return;
      e.preventDefault();
      el.scrollLeft = next;
    };
    // React registers wheel listeners passively, so onWheel can't preventDefault.
    el.addEventListener("wheel", handleWheel, { passive: false });
    detachRef.current = () => el.removeEventListener("wheel", handleWheel);
  }, []);
};
