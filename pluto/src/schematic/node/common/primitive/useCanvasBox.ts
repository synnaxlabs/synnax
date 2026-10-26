// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { CSS } from "@synnaxlabs/lyra/css";
import { useResize } from "@synnaxlabs/lyra/hooks";
import { box, xy } from "@synnaxlabs/x";
import { type RefCallback, useState } from "react";

const ROOT_SELECTOR = `.${CSS.B("symbol-primitive")}`;

const px = (value: string): number => parseFloat(value) || 0;

// The diagram zooms with a transform. Layout sizes such as offsetWidth ignore it and
// client rects include it, so their ratio is the zoom.
const measure = (el: HTMLElement): box.Box => {
  const root = el.closest(ROOT_SELECTOR);
  if (!(root instanceof HTMLElement))
    throw new Error("useCanvasBox needs an element inside a symbol primitive");
  const rootRect = root.getBoundingClientRect();
  const rect = el.getBoundingClientRect();
  const zoom = root.offsetWidth === 0 ? 1 : rootRect.width / root.offsetWidth;
  const style = getComputedStyle(el);
  const padLeft = px(style.paddingLeft);
  const padTop = px(style.paddingTop);
  return box.construct(
    {
      x: (rect.left - rootRect.left) / zoom + el.clientLeft + padLeft,
      y: (rect.top - rootRect.top) / zoom + el.clientTop + padTop,
    },
    {
      width: el.clientWidth - padLeft - px(style.paddingRight),
      height: el.clientHeight - padTop - px(style.paddingBottom),
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
 * it. It measures again when the element resizes.
 * @param position - The node's position. The symbol's root must sit at it.
 * @throws {Error} if the element is not inside a symbol primitive.
 */
export const useCanvasBox = <E extends HTMLElement>(
  position: xy.XY = xy.ZERO,
): UseCanvasBoxReturn<E> => {
  const [offset, setOffset] = useState<box.Box>(box.ZERO);
  const ref = useResize<E>((_, el) => setOffset(measure(el)));
  return { ref, box: box.translate(offset, position) };
};
