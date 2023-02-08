// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  MouseEventHandler,
  PropsWithChildren,
  RefObject,
  useRef,
  useState,
} from "react";

import { unique } from "@synnaxlabs/x";

import { useClickOutside } from "@/hooks";
import { toXY, XY, ZERO_XY } from "@/spatial";
import { RenderProp } from "@/util/renderProp";

import "./MenuContext.css";

export interface MenuContextProps extends PropsWithChildren {
  menu?: RenderProp<MenuContextMenuProps>;
}

interface MenuState {
  open: boolean;
  keys: string[];
  xy: XY;
}

export interface MenuContextMenuProps {
  keys: string[];
}

const INITIAL_STATE: MenuState = {
  open: false,
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
    candidate = target.parentElement as HTMLElement;
  }
  return candidate;
};

const findSelected = (target_: HTMLElement): HTMLElement[] => {
  const target = findTarget(target_);
  const selected = (target.parentElement?.querySelectorAll(`.${CONTEXT_SELECTED}`) ??
    []) as HTMLElement[];
  return [target, ...Array.from(selected)];
};

export const MenuContext = ({ children, menu }: MenuContextProps): JSX.Element => {
  const menuRef = useRef<HTMLDivElement | null>(null);
  const [{ keys, xy, open }, setMenuState] = useState<MenuState>(INITIAL_STATE);

  const handleContextMenu: MouseEventHandler<HTMLDivElement> = (e) => {
    e.preventDefault();
    e.stopPropagation();
    const selected = findSelected(e.target as HTMLElement);
    const keys = unique(selected.map((el) => el.id));
    setMenuState({ open: true, keys, xy: toXY(e) });
  };

  const hideMenu = (): void => setMenuState(INITIAL_STATE);

  useClickOutside(menuRef, hideMenu);

  const refCallback = (el: HTMLDivElement): void => {
    menuRef.current = el;
    if (el == null) return;
    if (open) {
      const [_xy, changed] = positionContextMenu(el, xy);
      if (changed) setMenuState({ open: true, keys, xy: _xy });
    }
  };

  return (
    <div className={MENU_CONTEXT_CONTAINER} onContextMenu={handleContextMenu}>
      {children}
      {open && (
        <div
          className="pluto-menu-context pluto-bordered"
          ref={refCallback}
          style={{ left: xy.x, top: xy.y }}
          onClick={hideMenu}
        >
          {menu?.({ keys })}
        </div>
      )}
    </div>
  );
};

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
