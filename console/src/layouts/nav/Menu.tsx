// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/layouts/nav/Nav.css";

import { CSS as PCSS, Menu as PMenu } from "@synnaxlabs/pluto";
import { xy } from "@synnaxlabs/x";
import { type ReactElement, useRef } from "react";

import { CSS } from "@/css";
import { Layout } from "@/layout";
import { DRAWER_ITEMS } from "@/layouts/nav/drawerItems";

export interface MenuProps extends Omit<PMenu.MenuProps, "children" | "onChange"> {
  location: Layout.NavDrawerLocation;
}

export const Menu = ({ location, ...rest }: MenuProps): ReactElement => {
  const positionRef = useRef<xy.XY>({ ...xy.ZERO });
  const timeoutRef = useRef<NodeJS.Timeout | null>(null);
  const { onSelect, menuItems, activeItem, onStartHover, onStopHover } =
    Layout.useNavDrawer(location, DRAWER_ITEMS);

  return (
    <PMenu.Menu {...rest} onChange={onSelect}>
      {menuItems.map(({ key, icon, trigger }) => (
        <PMenu.Item
          className={CSS(
            CSS.BE("main-nav", "item"),
            PCSS.selected(activeItem?.key === key),
          )}
          onClick={() => {
            if (timeoutRef.current != null) {
              clearTimeout(timeoutRef.current);
              timeoutRef.current = null;
            }
          }}
          onMouseEnter={(e) => {
            timeoutRef.current = setTimeout(() => {
              timeoutRef.current = null;
              onStartHover(key);
              positionRef.current = xy.construct(e);
              const lis = (e: MouseEvent) => {
                const delta = xy.translation(xy.construct(e), positionRef.current);
                if (Math.abs(delta.y) > 75 && Math.abs(delta.x) < 30) {
                  onStopHover();
                  window.removeEventListener("mousemove", lis);
                }
              };
              window.addEventListener("mousemove", lis);
            }, 350);
          }}
          onMouseLeave={() => {
            if (timeoutRef.current != null) {
              clearTimeout(timeoutRef.current);
              timeoutRef.current = null;
            }
          }}
          key={key}
          itemKey={key}
          size="large"
          contrast={2}
          triggerIndicator={trigger}
        >
          {icon}
        </PMenu.Item>
      ))}
    </PMenu.Menu>
  );
};
