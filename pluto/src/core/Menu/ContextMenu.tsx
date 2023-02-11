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
  ComponentPropsWithRef,
  ForwardedRef,
  forwardRef,
  HTMLProps,
  PropsWithChildren,
  RefCallback,
  useRef,
  useState,
} from "react";

import { unique } from "@synnaxlabs/x";

import { useClickOutside } from "@/hooks";
import { ClientXY, toXY, XY, ZERO_XY } from "@/spatial";
import { RenderProp } from "@/util/renderProp";

import "./ContextMenu.css";
import clsx from "clsx";

export interface ContextMenuState {
  visible: boolean;
  keys: string[];
  xy: XY;
}

export type ContextMenuEvent = ClientXY & {
  preventDefault: () => void;
  stopPropagation: () => void;
  target: Element;
};

export type ContextMenuOpen = (
  pos: XY | ClientXY | ContextMenuEvent,
  keys?: string[]
) => void;

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

const CONTEXT_SELECTED = "pluto-context-selected";
const CONTEXT_TARGET = "pluto-context-target";
const MENU_CONTEXT_CONTAINER = "pluto-menu-context__container";

const findTarget = (target: HTMLElement): HTMLElement => {
  let candidate = target;
  while (candidate != null && !candidate.classList.contains(CONTEXT_TARGET)) {
    if (candidate.classList.contains(MENU_CONTEXT_CONTAINER)) return target;
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

export const useContextMenu = (): UseContextMenuReturn => {
  const menuRef = useRef<HTMLDivElement | null>(null);
  const [state, setMenuState] = useState<ContextMenuState>(INITIAL_STATE);

  const handleOpen: ContextMenuOpen = (e, keys) => {
    const xy = toXY(e);
    if ("preventDefault" in e) {
      e.preventDefault();
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
        const [_xy, changed] = positionContextMenu(el, prev.xy);
        if (changed) return { ...prev, xy: _xy };
      }
      return prev;
    });
  };

  const hideMenu = (): void => {
    setMenuState(INITIAL_STATE);
  };

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
      className={clsx(MENU_CONTEXT_CONTAINER, className)}
      onContextMenu={open}
      {...props}
    >
      {children}
      {visible && (
        <div
          className="pluto-menu-context pluto-bordered"
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

const positionContextMenu = (el: HTMLDivElement, xy: XY): [XY, boolean] => {
  const { width, height } = el.getBoundingClientRect();
  const { innerWidth, innerHeight } = window;
  let changed = false;
  if (xy.x + width > innerWidth) {
    xy.x -= width;
    changed = true;
  }
  if (xy.y + height > innerHeight) {
    xy.y -= height;
    changed = true;
  }
  return [xy, changed];
};
