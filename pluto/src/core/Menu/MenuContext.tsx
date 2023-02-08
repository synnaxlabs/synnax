// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.
//

import { MouseEventHandler, PropsWithChildren, useRef, useState } from "react";

import { unique } from "@synnaxlabs/x";

import { useClickOutside } from "@/hooks";
import { toXY, XY, ZERO_XY } from "@/spatial";
import { RenderProp } from "@/util/renderProp";

import "./MenuContext.css";

export interface MenuContextProps extends PropsWithChildren {
  menu?: RenderProp<MenuContextMenuProps>;
}

interface MenuState extends XY {
  open: boolean;
  keys: string[];
}

export interface MenuContextMenuProps {
  keys: string[];
}

const INITIAL_STATE: MenuState = {
  open: false,
  keys: [],
  ...ZERO_XY
};

export const MenuContext = ({ children, menu }: MenuContextProps): JSX.Element => {
  const menuRef = useRef<HTMLDivElement>(null);
  const [menuState, setMenuState] = useState<MenuState>(INITIAL_STATE);

  const handleContextMenu: MouseEventHandler<HTMLDivElement> = (e) => {
    // Traverse the parent of the element until we find one
    // with the class 'pluto-context-target'
    e.preventDefault();
    e.stopPropagation();
    let target = e.target as HTMLElement;
    while (target != null && !target.classList.contains('pluto-context-target')) {
      // if we reach the top of the tree, we're done
      if (target.classList.contains('pluto-menu-context__container')) {
        target = e.target as HTMLElement;
        break;
      } 
      target = target.parentElement as HTMLElement;
    }

    const selected = target.parentElement?.querySelectorAll('.pluto-context-selected') ?? [];
    const keys = unique([target.id, ...Array.from(selected).map((el) => el.id)]);
    setMenuState({open: true, keys, ...toXY(e) });
  }

  const handleMenuClick = (): void => {
    setMenuState(INITIAL_STATE);
  }

  useClickOutside(menuRef, () => setMenuState(INITIAL_STATE));

  return (
    <div 
        className="pluto-menu-context__container" 
        onContextMenu={handleContextMenu} 
    >
        {children}
        {menuState.open && 
        <div 
          className="pluto-menu-context pluto-bordered" 
          ref={menuRef} 
          style={{ left: menuState.x, top: menuState.y }}
          onClick={handleMenuClick}
        >
          {menu?.({ keys: menuState.keys })}
        </div>}
    </div>
  );
};
