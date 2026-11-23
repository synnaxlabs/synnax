// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { KeyboardEvent } from "react";

import { FOCUSABLE } from "@/util";

/** Whether b comes after a in the document. */
export const follows = (a: Node, b: Node): boolean =>
  (a.compareDocumentPosition(b) & Node.DOCUMENT_POSITION_FOLLOWING) !== 0;

/** @returns the elements in root that Tab can stop on, in document order. */
export const tabbable = (root: HTMLElement): HTMLElement[] =>
  Array.from(root.querySelectorAll<HTMLElement>(FOCUSABLE)).filter(
    (el) =>
      el.tabIndex >= 0 && !el.matches(":disabled") && el.getClientRects().length > 0,
  );

/**
 * Moves Tab and Shift+Tab through the tab stops of a modal surface, wrapping at its
 * ends. WebKit on macOS skips buttons on Tab, so the modal moves the focus itself and
 * stops on the same elements in every engine. A Tab that a field already handled is
 * left alone.
 */
export const trapTab = (e: KeyboardEvent, surface: HTMLElement): void => {
  if (e.key !== "Tab" || e.defaultPrevented || e.ctrlKey || e.metaKey || e.altKey)
    return;
  e.preventDefault();
  const active = document.activeElement ?? document.body;
  const items = tabbable(surface);
  const next = e.shiftKey
    ? (items.findLast((el) => follows(el, active)) ?? items.at(-1))
    : (items.find((el) => follows(active, el)) ?? items[0]);
  next?.focus();
};
