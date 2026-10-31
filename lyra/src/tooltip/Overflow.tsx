// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, useEffect, useRef } from "react";

import { CSS } from "@/css";
import { formatTip } from "@/tooltip/Dialog";
import { Frame } from "@/tooltip/Frame";
import { useVisibility } from "@/tooltip/useVisibility";

const FADE_SELECTOR = `.${CSS.BM("text", "overflow", "fade")}`;

const isCut = (el: HTMLElement): boolean =>
  !el.isContentEditable && el.scrollWidth > el.clientWidth;

/**
 * Shows the full text of a cut label in a tooltip while the pointer rests on it. A
 * label is cut when it fades because it outruns its box. Mount it once inside {@link
 * Config}.
 */
export const Overflow = (): ReactElement | null => {
  // React handles pointer events at its root, before they reach the document, so an
  // element's own tooltip always starts its delay first and opens first. Yielding then
  // keeps that tooltip in place.
  const { anchor, closing, hover, close } = useVisibility({ yielding: true });
  const hoveredRef = useRef<HTMLElement | null>(null);
  useEffect(() => {
    const handleOver = (e: PointerEvent): void => {
      const label =
        e.target instanceof Element
          ? e.target.closest<HTMLElement>(FADE_SELECTOR)
          : null;
      if (label === hoveredRef.current) return;
      hoveredRef.current = label;
      close();
      if (label != null && e.pointerType !== "touch" && isCut(label)) hover(label);
    };
    const handleOut = (e: PointerEvent): void => {
      if (e.relatedTarget != null) return;
      hoveredRef.current = null;
      close();
    };
    const handleDown = (): void => close(true);
    document.addEventListener("pointerover", handleOver);
    document.addEventListener("pointerout", handleOut);
    document.addEventListener("pointerdown", handleDown, { capture: true });
    return () => {
      document.removeEventListener("pointerover", handleOver);
      document.removeEventListener("pointerout", handleOut);
      document.removeEventListener("pointerdown", handleDown, { capture: true });
    };
  }, [hover, close]);
  if (anchor == null) return null;
  return (
    <Frame
      anchor={anchor}
      className={closing ? CSS.M("closing") : undefined}
      onAnchorLost={() => close(true)}
    >
      {formatTip(anchor.textContent)}
    </Frame>
  );
};
