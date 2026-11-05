// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type FocusEvent, type RefObject, useLayoutEffect, useRef } from "react";

import { inTopModal } from "@/dialog/Background";
import { FOCUSABLE } from "@/util";

/** Whether the node is in the page behind the surface: before it or around it. */
const behind = (surface: Node, node: Node): boolean =>
  (surface.compareDocumentPosition(node) & Node.DOCUMENT_POSITION_PRECEDING) !== 0;

const follows = (a: Node, b: Node): boolean =>
  (a.compareDocumentPosition(b) & Node.DOCUMENT_POSITION_FOLLOWING) !== 0;

const tabbable = (root: HTMLElement): HTMLElement[] =>
  Array.from(root.querySelectorAll<HTMLElement>(FOCUSABLE)).filter(
    (el) =>
      el.tabIndex >= 0 && !el.matches(":disabled") && el.getClientRects().length > 0,
  );

/** Props for the backdrop of a modal that {@link useFocusTrap} holds the focus in. */
export interface FocusTrapProps {
  onFocusCapture: (e: FocusEvent<HTMLElement>) => void;
}

/**
 * Holds the focus in a modal surface while enabled. The surface takes the focus unless
 * a field in it already has it, Tab and Shift+Tab step through it and wrap at its ends,
 * and focus that lands on the page behind comes back. Once disabled, the focus goes
 * back to the opener. Only the top modal acts, and an overlay drawn over it, such as a
 * dropdown, keeps its own focus.
 *
 * @param surfaceRef - The modal surface. It must take focus, so give it a tabIndex.
 * @returns props for the backdrop of the modal.
 */
export const useFocusTrap = (
  surfaceRef: RefObject<HTMLElement | null>,
  enabled: boolean,
): FocusTrapProps => {
  // A field with autoFocus takes the focus before any effect runs, so the opener comes
  // from the first focus event that reaches the modal.
  const openerRef = useRef<Element | null>(null);
  const enteredRef = useRef(false);
  const handleFocusCapture = (e: FocusEvent<HTMLElement>): void => {
    if (!enabled || enteredRef.current) return;
    enteredRef.current = true;
    openerRef.current = e.relatedTarget;
  };

  // A layout effect gives the focus back before a modal opened in the same commit
  // takes it.
  useLayoutEffect(() => {
    const surface = surfaceRef.current;
    if (!enabled || surface == null) return;
    const active = document.activeElement;
    let last =
      active instanceof HTMLElement && surface.contains(active) ? active : surface;
    if (last === surface) surface.focus({ preventScroll: true });

    const handleFocusIn = (e: globalThis.FocusEvent): void => {
      const { target } = e;
      if (!(target instanceof HTMLElement) || !inTopModal(surface)) return;
      if (surface.contains(target)) last = target;
      else if (behind(surface, target))
        (last.isConnected ? last : surface).focus({ preventScroll: true });
    };

    const handleKeyDown = (e: KeyboardEvent): void => {
      if (e.key !== "Tab" || e.defaultPrevented || e.ctrlKey || e.metaKey || e.altKey)
        return;
      const active = document.activeElement ?? document.body;
      // An overlay drawn over the modal, such as a dropdown, moves its own focus.
      if (
        !inTopModal(surface) ||
        (!surface.contains(active) && !behind(surface, active))
      )
        return;
      // WebKit on macOS skips buttons on Tab, so the modal moves the focus itself and
      // stops on the same elements in every engine.
      e.preventDefault();
      const items = tabbable(surface);
      const next = e.shiftKey
        ? (items.findLast((el) => follows(el, active)) ?? items.at(-1))
        : (items.find((el) => follows(active, el)) ?? items[0]);
      (next ?? surface).focus();
    };

    document.addEventListener("focusin", handleFocusIn);
    document.addEventListener("keydown", handleKeyDown);
    return () => {
      document.removeEventListener("focusin", handleFocusIn);
      document.removeEventListener("keydown", handleKeyDown);
      const opener = openerRef.current;
      openerRef.current = null;
      enteredRef.current = false;
      if (opener instanceof HTMLElement && opener.isConnected)
        opener.focus({ preventScroll: true });
    };
  }, [enabled]);

  return { onFocusCapture: handleFocusCapture };
};
