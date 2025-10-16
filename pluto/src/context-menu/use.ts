// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, location, unique, xy } from "@synnaxlabs/x";
import { type RefCallback, useCallback, useRef, useState } from "react";

import { SELECTED_CSS_CLASS, TARGET_CSS_CLASS } from "@/context-menu/types";
import { CSS } from "@/css";
import { Dialog } from "@/dialog";
import { useClickOutside } from "@/hooks";

export interface State {
  visible: boolean;
  keys: string[];
  position: xy.XY;
  cursor: xy.XY;
}

/** Supported event types for triggering a context menu. */
export interface Event extends xy.Client {
  preventDefault: () => void;
  stopPropagation: () => void;
  target: Element;
}

/** Opens the context menu. See {@link Menu.useContextMenu} for more details. */
export interface Open {
  (pos: xy.Crude | Event, keys?: string[]): void;
}

/** Return value for the {@Menu.useContextMenu} hook. */
export interface UseReturn extends State {
  visible: boolean;
  close: () => void;
  open: Open;
  ref: RefCallback<HTMLDivElement>;
  className: string;
}

const INITIAL_STATE: State = {
  visible: false,
  position: xy.ZERO,
  cursor: xy.ZERO,
  keys: [],
};

const CONTEXT_MENU_CONTAINER = CSS.BE("context-menu", "container");

const findTarget = (target: HTMLElement): HTMLElement | null => {
  let candidate = target;
  while (!candidate.classList.contains(TARGET_CSS_CLASS)) {
    if (candidate.classList.contains(CONTEXT_MENU_CONTAINER)) return target;
    if (candidate.parentElement == null) return target;
    candidate = candidate.parentElement;
  }
  if (!candidate.classList.contains(TARGET_CSS_CLASS)) return target;
  return candidate;
};

const findSelected = (target_: HTMLElement): HTMLElement[] => {
  const target = findTarget(target_);
  if (target == null) return [];
  const selected: HTMLElement[] = Array.from(
    target.parentElement?.querySelectorAll(`.${SELECTED_CSS_CLASS}`) ?? [],
  );
  if (selected.includes(target)) return selected;
  return [target];
};

const PREFERENCES: Dialog.LocationPreference[] = [
  { targetCorner: location.BOTTOM_RIGHT, dialogCorner: location.TOP_LEFT },
  { targetCorner: location.BOTTOM_LEFT, dialogCorner: location.TOP_RIGHT },
  { targetCorner: location.TOP_RIGHT, dialogCorner: location.BOTTOM_LEFT },
  { targetCorner: location.TOP_LEFT, dialogCorner: location.BOTTOM_RIGHT },
];

/**
 * ContextMenu.use extracts the logic for toggling a context menu, allowing the caller
 * to control the menu's visibility and position.
 *
 * @returns visible - Whether the menu is visible.
 * @returns close - A function to close the menu.
 * @returns open - A function to open the menu. The function accepts an XY coordinate
 * and an optional set of keys to set as the selected menu items. It's important to note
 * that these keys override the default behavior of the menu, which is explained in the
 * documentation for {@link ContextMenu.ContextMenu}.
 */
export const use = (): UseReturn => {
  const menuRef = useRef<HTMLDivElement | null>(null);
  const [state, setMenuState] = useState<State>(INITIAL_STATE);
  const handleOpen: Open = useCallback((e, keys) => {
    const p = xy.construct(e);
    if (typeof e === "object" && "preventDefault" in e) {
      e.preventDefault();
      // Prevent parent context menus from opening.
      e.stopPropagation();
      const selected = findSelected(e.target as HTMLElement);
      keys ??= unique.unique(selected.map((el) => el.id).filter((id) => id.length > 0));
    } else keys = [];
    setMenuState({ visible: true, keys, position: p, cursor: p });
  }, []);
  const refCallback = useCallback((el: HTMLDivElement): void => {
    menuRef.current = el;
    if (el == null) return;
    setMenuState((prev) => {
      if (!prev.visible) return prev;
      const { adjustedDialog } = Dialog.position({
        container: box.construct(0, 0, window.innerWidth, window.innerHeight),
        dialog: box.construct(el),
        target: box.construct(prev.cursor, 0, 0),
        prefer: PREFERENCES,
      });
      const nextPos = box.topLeft(adjustedDialog);
      if (xy.equals(prev.position, nextPos)) return prev;
      return { ...prev, position: nextPos };
    });
  }, []);

  const hideMenu = (): void => setMenuState(INITIAL_STATE);

  useClickOutside({ ref: menuRef, onClickOutside: hideMenu });

  return {
    ...state,
    close: hideMenu,
    open: handleOpen,
    ref: refCallback,
    className: CONTEXT_MENU_CONTAINER,
  };
};
