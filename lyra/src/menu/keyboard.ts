// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type KeyboardEvent } from "react";

import { Triggers } from "@/triggers";

/**
 * @returns the key of the event if it opens a context menu, else null. Shift+F10 opens
 * the menu on its press, and the Menu key on its release, as on Windows. A caller
 * prevents the default of both, so the webview does not open its own menu.
 */
export const keyboardMenuKey = (
  e: KeyboardEvent<HTMLElement>,
): "F10" | "ContextMenu" | null => {
  if (e.altKey || e.ctrlKey || e.metaKey) return null;
  const key = Triggers.eventKey(e);
  if (key === "ContextMenu" || (key === "F10" && e.shiftKey)) return key;
  return null;
};

/** Opens the context menu of the element, as a right click on its bottom left would. */
export const openContextMenu = (element: Element): void => {
  const { left, bottom } = element.getBoundingClientRect();
  element.dispatchEvent(
    new MouseEvent("contextmenu", {
      bubbles: true,
      cancelable: true,
      clientX: left,
      clientY: bottom,
    }),
  );
};
