// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type KeyboardEvent as ReactKeyboardEvent } from "react";

import { Triggers } from "@/triggers";

type MenuKeyEvent = KeyboardEvent | ReactKeyboardEvent;

const keyboardMenuKey = (e: MenuKeyEvent): "F10" | "ContextMenu" | null => {
  if (e.altKey || e.ctrlKey || e.metaKey) return null;
  const key = Triggers.eventKey(e);
  if (key === "ContextMenu" || (key === "F10" && e.shiftKey)) return key;
  return null;
};

/**
 * Calls open on the first press of Shift+F10. Shift+F10 and the Menu key lose their
 * default, so the webview does not open its own menu.
 * @returns true if the key opens a context menu.
 */
export const menuKeyDown = (e: MenuKeyEvent, open: () => void): boolean => {
  const key = keyboardMenuKey(e);
  if (key == null) return false;
  e.preventDefault();
  if (key === "F10" && !e.repeat) open();
  return true;
};

/**
 * Calls open on the release of the Menu key, as on Windows.
 * @returns true if the key opens a context menu.
 */
export const menuKeyUp = (e: MenuKeyEvent, open: () => void): boolean => {
  if (keyboardMenuKey(e) !== "ContextMenu") return false;
  e.preventDefault();
  open();
  return true;
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
