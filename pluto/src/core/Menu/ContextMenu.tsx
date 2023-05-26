// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  ComponentPropsWithoutRef,
  ForwardedRef,
  forwardRef,
  RefCallback,
  useRef,
  useState,
} from "react";

import { unique, ClientXY, toXY, XY, ZERO_XY, positionSoVisible } from "@synnaxlabs/x";

import "@/core/Menu/ContextMenu.css";
import { CSS } from "@/css";
import { useClickOutside } from "@/hooks";
import { RenderProp } from "@/util/renderProp";

interface ContextMenuState {
  visible: boolean;
  keys: string[];
  xy: XY;
}

/** Supported event types for triggering a context menu. */
export type ContextMenuEvent = ClientXY & {
  preventDefault: () => void;
  stopPropagation: () => void;
  target: Element;
};

/** Opens the context menu. See {@link Menu.useContextMenu} for more details. */
export type ContextMenuOpen = (
  pos: XY | ClientXY | ContextMenuEvent,
  keys?: string[]
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
  xy: ZERO_XY,
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

/** Should not be imported directly. Use {@link Menu.useContextMenu} instead. */
export const useContextMenu = (): UseContextMenuReturn => {
  const menuRef = useRef<HTMLDivElement | null>(null);
  const [state, setMenuState] = useState<ContextMenuState>(INITIAL_STATE);

  const handleOpen: ContextMenuOpen = (e, keys) => {
    const xy = toXY(e);
    if ("preventDefault" in e) {
      e.preventDefault();
      // Prevent parent context menus from opening.
      e.stopPropagation();
      keys = keys ?? unique(findSelected(e.target as HTMLElement).map((el) => el.id));
    } else keys = [];
    setMenuState({ visible: true, keys, xy });
  };

  const refCallback = (el: HTMLDivElement): void => {
    menuRef.current = el;
    if (el == null) return;
    setMenuState((prev) => {
      if (prev.visible) {
        const [repositioned, changed] = positionSoVisible(
          el,
          window.document.documentElement
        );
        if (changed) return { ...prev, xy: repositioned.topLeft };
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
  ref: ForwardedRef<HTMLDivElement>
): JSX.Element => {
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

export const ContextMenu = forwardRef(ContextMenuCore);
ContextMenu.displayName = "ContextMenu";
