// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type KeyboardEvent, type RefCallback, type RefObject } from "react";

import { follows, tabbable } from "@/dialog/tabOrder";
import { contains } from "@/dialog/useClickOutside";
import { useSyncedRef } from "@/hooks";

/** Props for the surface of a dropdown that {@link useDropdownFocus} manages. */
export interface DropdownFocusProps {
  ref: RefCallback<HTMLElement>;
  onKeyDown: (e: KeyboardEvent<HTMLElement>) => void;
}

/**
 * Gives a dropdown, which portals to the end of the page, the place right after its
 * trigger in the tab order. Tab past its last stop closes it and goes on from the
 * trigger. Shift+Tab past its first stop closes it and focuses the trigger. A close
 * while it, or a dropdown opened from it, holds the focus gives the focus to the
 * trigger. A dropdown with no trigger leaves the focus to the browser.
 *
 * @param enabled - Whether the hook acts.
 * @param trigger - The element that opens the dropdown.
 * @param close - Closes the dropdown.
 * @returns props for the dropdown surface.
 */
export const useDropdownFocus = (
  enabled: boolean,
  trigger: RefObject<HTMLElement | null>,
  close: () => void,
): DropdownFocusProps => {
  const enabledRef = useSyncedRef(enabled);

  // React detaches the ref before it removes the surface, so the focus is still there.
  const ref = (surface: HTMLElement | null) => {
    if (surface == null) return;
    return () => {
      if (enabledRef.current && contains(surface, document.activeElement))
        trigger.current?.focus({ preventScroll: true });
    };
  };

  const handleKeyDown = (e: KeyboardEvent<HTMLElement>): void => {
    if (
      !enabled ||
      e.key !== "Tab" ||
      e.defaultPrevented ||
      e.ctrlKey ||
      e.metaKey ||
      e.altKey
    )
      return;
    const surface = e.currentTarget;
    const active = document.activeElement;
    // Keys from a portaled child, such as a modal, bubble here through React.
    if (active == null || !surface.contains(active)) return;
    const items = tabbable(surface);
    const next = e.shiftKey
      ? items.findLast((el) => follows(el, active))
      : items.find((el) => follows(active, el));
    if (next != null) {
      // WebKit on macOS skips buttons on Tab, so the dropdown moves the focus itself.
      e.preventDefault();
      next.focus();
      return;
    }
    const el = trigger.current;
    if (el == null) return;
    // Tab goes on from the trigger, by default or by the focus trap of a modal.
    if (e.shiftKey) e.preventDefault();
    el.focus();
    close();
  };

  return { ref, onKeyDown: handleKeyDown };
};
