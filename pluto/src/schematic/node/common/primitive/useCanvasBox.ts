// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, xy } from "@synnaxlabs/x";
import { type RefCallback, useRef, useState } from "react";

import { CSS } from "@/css";

const ROOT_SELECTOR = `.${CSS.B("symbol-primitive")}`;

const rootOf = (el: HTMLElement): HTMLElement => {
  const root = el.closest(ROOT_SELECTOR);
  if (!(root instanceof HTMLElement))
    throw new Error("useCanvasBox needs an element inside a symbol primitive");
  return root;
};

// The diagram zooms with a transform. Layout sizes such as offsetWidth ignore it and
// client rects include it, so their ratio is the zoom.
const measure = (el: HTMLElement, root: HTMLElement): box.Box => {
  // A hidden symbol has no layout to draw into.
  if (root.offsetWidth === 0) return box.ZERO;
  const rootRect = root.getBoundingClientRect();
  const rect = el.getBoundingClientRect();
  const zoom = rootRect.width / root.offsetWidth;
  const style = getComputedStyle(el);
  const padLeft = parseFloat(style.paddingLeft);
  const padTop = parseFloat(style.paddingTop);
  return box.construct(
    {
      x: (rect.left - rootRect.left) / zoom + el.clientLeft + padLeft,
      y: (rect.top - rootRect.top) / zoom + el.clientTop + padTop,
    },
    {
      width: el.clientWidth - padLeft - parseFloat(style.paddingRight),
      height: el.clientHeight - padTop - parseFloat(style.paddingBottom),
    },
  );
};

export interface UseCanvasBoxReturn<E extends HTMLElement> {
  /** ref takes the element to measure. */
  ref: RefCallback<E>;
  /** box is the element's content box in diagram coordinates. Empty before mount. */
  box: box.Box;
}

/**
 * useCanvasBox measures where an element sits in a symbol, so the worker can draw into
 * it. It measures again when the element or the symbol resizes.
 * @param position - The node's position. The symbol's root must sit at it.
 * @throws {Error} if the element is not inside a symbol primitive.
 */
export const useCanvasBox = <E extends HTMLElement>(
  position: xy.XY = xy.ZERO,
): UseCanvasBoxReturn<E> => {
  const [offset, setOffset] = useState<box.Box>(box.ZERO);
  const disconnectRef = useRef<(() => void) | null>(null);
  const ref = (el: E | null): void => {
    disconnectRef.current?.();
    disconnectRef.current = null;
    if (el == null) return;
    const root = rootOf(el);
    const observer = new ResizeObserver(() => {
      const next = measure(el, root);
      setOffset((prev) => (box.equals(prev, next) ? prev : next));
    });
    // A sibling that grows moves the element without resizing it, but resizes the root.
    observer.observe(el);
    observer.observe(root);
    disconnectRef.current = () => observer.disconnect();
  };
  return { ref, box: box.translate(offset, position) };
};
