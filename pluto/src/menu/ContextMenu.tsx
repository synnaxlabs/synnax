// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type ComponentPropsWithoutRef,
  type ForwardedRef,
  forwardRef,
  type ReactElement,
  type RefCallback,
  useRef,
  useState,
} from "react";

import { box, unique, xy } from "@synnaxlabs/x";

import { CSS } from "@/css";
import { useClickOutside } from "@/hooks";
import { type RenderProp } from "@/util/renderProp";

import "@/menu/ContextMenu.css";

interface ContextMenuState {
  visible: boolean;
  keys: string[];
  xy: xy.XY;
}

/** Supported event types for triggering a context menu. */
export type ContextMenuEvent = xy.Client & {
  preventDefault: () => void;
  stopPropagation: () => void;
  target: Element;
};

/** Opens the context menu. See {@link Menu.useContextMenu} for more details. */
export type ContextMenuOpen = (
  pos: xy.Crude | ContextMenuEvent,
  keys?: string[],
) => void;

/** Return value for the {@Menu.useContextMenu} hook. */
export interface UseContextMenuReturn extends ContextMenuState {
  visible: boolean;
  close: () => void;
  open: ContextMenuOpen;
  ref: RefCallback<HTMLDivElement>;
}

const INITIAL_STATE: ContextMenuState = {
  visible: false,
  keys: [],
  xy: xy.ZERO,
};

export const CONTEXT_SELECTED = CSS.BM("context", "selected");
export const CONTEXT_TARGET = CSS.BE("context", "target");
const CONTEXT_MENU_CONTAINER = CSS.BE("menu-context", "container");

const findTarget = (target: HTMLElement): HTMLElement => {
  let candidate = target;
  while (candidate != null && !candidate.classList.contains(CONTEXT_TARGET)) {
    if (candidate.classList.contains(CONTEXT_MENU_CONTAINER)) return target;
    candidate = candidate.parentElement as HTMLElement;
  }
  return candidate;
};

const findSelected = (target_: HTMLElement): HTMLElement[] => {
  const target = findTarget(target_);
  const selected = (target.parentElement?.querySelectorAll(`.${CONTEXT_SELECTED}`) ??
    []) as HTMLElement[];
  return [target, ...Array.from(selected)];
};

/**
 * Menu.useContextMenu extracts the logic for toggling a context menu, allowing
 * the caller to control the menu's visibility and position.
 *
 * @returns visible - Whether the menu is visible.
 * @returns close - A function to close the menu.
 * @returns open - A function to open the menu. The function accepts an XY coordinate and
 * an optional set of keys to set as the selected menu items. It's important to note
 * that these keys override the default behavior of the menu, which is explained in
 * the documentation for {@link Menu.ContextMenu}.
 */
export const useContextMenu = (): UseContextMenuReturn => {
  const menuRef = useRef<HTMLDivElement | null>(null);
  const [state, setMenuState] = useState<ContextMenuState>(INITIAL_STATE);

  const handleOpen: ContextMenuOpen = (e, keys) => {
    const p = xy.construct(e);
    if ("preventDefault" in e) {
      e.preventDefault();
      // Prevent parent context menus from opening.
      e.stopPropagation();
      keys = keys ?? unique(findSelected(e.target as HTMLElement).map((el) => el.id));
    } else keys = [];
    setMenuState({ visible: true, keys, xy: p });
  };

  const refCallback = (el: HTMLDivElement): void => {
    menuRef.current = el;
    if (el == null) return;
    setMenuState((prev) => {
      if (prev.visible) {
        const [repositioned, changed] = box.positionSoVisible(
          el,
          window.document.documentElement,
        );
        if (changed) return { ...prev, xy: box.topLeft(repositioned) };
      }
      return prev;
    });
  };

  const hideMenu = (): void => setMenuState(INITIAL_STATE);

  useClickOutside(menuRef, hideMenu);

  return {
    ...state,
    close: hideMenu,
    open: handleOpen,
    ref: refCallback,
  };
};

export interface ContextMenuMenuProps {
  keys: string[];
}

export interface ContextMenuProps
  extends UseContextMenuReturn,
    ComponentPropsWithoutRef<"div"> {
  menu?: RenderProp<ContextMenuMenuProps>;
}

const ContextMenuCore = (
  {
    children,
    menu,
    visible,
    open,
    close,
    xy,
    keys,
    className,
    ...props
  }: ContextMenuProps,
  ref: ForwardedRef<HTMLDivElement>,
): ReactElement => {
  return (
    <div
      className={CSS(CONTEXT_MENU_CONTAINER, className)}
      onContextMenu={open}
      {...props}
    >
      {children}
      {visible && (
        <div
          className={CSS(CSS.B("menu-context"), CSS.bordered())}
          ref={ref}
          style={{ left: xy.x, top: xy.y }}
          onClick={close}
        >
          {menu?.({ keys })}
        </div>
      )}
    </div>
  );
};

/**
 * Menu.ContextMenu wraps a set of children with a context menu. When the user right
 * clicks within wrapped area, the provided menu will be shown. Menu.ContextMenu should
 * be used in conjunction with the Menu.useContextMenu hook.
 *
 * The rendered menu is provided with a set of keys that represent the HTML IDs of the
 * context target elements. The first target is evaluated by traversing the parents
 * of the element that was right clicked until an element with the class "pluto-context-target"
 * is found. If no such element is found, the right clicked element itself is used as
 * the target. Subsequent targets are found by querying all siblings of the first target
 * that have the "pluto-context-selected" class.
 *
 * @example <caption>Example DOM structure</caption>
 *   <div id="pluto-menu-context__container">
 *    <div className="pluto-context-target" id="1">
 *      <span>
 *        <h2>I was right clicked!</h2>
 *      </span>
 *    </div>
 *    <div className="pluto-context-target pluto-context-selected" id="2">
 *    <div className="pluto-context-target" id="3">
 *   </div>
 *
 * In the above example, the keys provided to the menu would be ["1", "2"].
 *
 * The target resolution logic is ideal for both single and multi-select scenarios,
 * such as lists that have several selected rows that should be acted upon together.
 *
 * @param props - Props for the component. Expects all return values from the
 * useContextMenu hook. All non-hook and unlisted props will be spread to the
 * underlying div component acting as the root element.
 * @param props.menu - The menu to show when the user right clicks.
 */
export const ContextMenu = forwardRef(ContextMenuCore);
ContextMenu.displayName = "ContextMenu";
