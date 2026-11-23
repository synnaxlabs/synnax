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
import { Portal } from "@/portal";
import { formatTip } from "@/tooltip/Dialog";
import { Frame } from "@/tooltip/Frame";
import { useVisibility } from "@/tooltip/useVisibility";

const FADE_CLASS = CSS.BM("text", "overflow", "fade");
const CLAMP_CLASS = CSS.BM("text", "line-clamp");
const CUT_SELECTOR = `.${FADE_CLASS}, .${CLAMP_CLASS}`;

// A clamped label wraps, so only its height shows a cut.
const isCut = (el: HTMLElement): boolean =>
  !el.isContentEditable &&
  (el.classList.contains(CLAMP_CLASS)
    ? el.scrollHeight > el.clientHeight
    : el.scrollWidth > el.clientWidth);

/**
 * Shows the full text of a cut label in a tooltip while the pointer rests on it. A
 * label is cut when it outruns its box or its line clamp. Mount it once inside {@link
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
          ? e.target.closest<HTMLElement>(CUT_SELECTOR)
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
  // A modal makes the page outside it inert and covers it, so a label in a modal gets
  // its tooltip in the modal.
  return (
    <Portal.Target element={anchor.closest("dialog")}>
      <Frame
        anchor={anchor}
        className={closing ? CSS.M("closing") : undefined}
        onAnchorLost={() => close(true)}
      >
        {formatTip(anchor.textContent)}
      </Frame>
    </Portal.Target>
  );
};
