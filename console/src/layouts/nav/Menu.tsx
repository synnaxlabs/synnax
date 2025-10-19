// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/layouts/nav/Nav.css";

import { List, Select } from "@synnaxlabs/pluto";
import { xy } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useRef } from "react";

import { CSS } from "@/css";
import { Layout } from "@/layout";
import { type NavDrawerItem } from "@/layout/useNavDrawer";
import { DRAWER_ITEMS } from "@/layouts/nav/drawerItems";

export interface MenuProps {
  location: Layout.NavDrawerLocation;
}

export const Menu = ({ location }: MenuProps): ReactElement => {
  const positionRef = useRef<xy.XY>({ ...xy.ZERO });
  const timeoutRef = useRef<NodeJS.Timeout>(undefined);
  const { onSelect, menuItems, activeItem, onStartHover, onStopHover } =
    Layout.useNavDrawer(location, DRAWER_ITEMS);

  const getItem = useCallback(
    (keys: string | string[]) => {
      if (typeof keys === "string")
        return menuItems.find(({ key: itemKey }) => itemKey === keys);
      return menuItems.filter(({ key: itemKey }) => keys.includes(itemKey));
    },
    [menuItems],
  ) as List.GetItem<string, NavDrawerItem>;

  return (
    <Select.Frame<string, NavDrawerItem>
      data={menuItems.map(({ key }) => key)}
      onChange={onSelect}
      getItem={getItem}
    >
      <List.Items<string, NavDrawerItem> className={CSS.BE("main-nav", "menu")}>
        {(p) => {
          const { getItem } = List.useUtilContext<string, NavDrawerItem>();
          const item = getItem?.(p.key);
          if (item == null) return null;
          return (
            <Select.ListItem
              key={item.key}
              itemKey={item.key}
              onClick={() => {
                clearTimeout(timeoutRef.current);
              }}
              onMouseEnter={(e) => {
                timeoutRef.current = setTimeout(() => {
                  onStartHover(item.key);
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
                clearTimeout(timeoutRef.current);
              }}
              triggerIndicator={item.trigger}
              contrast={2}
              variant="text"
              gap="medium"
              index={p.index}
              size="large"
              selected={activeItem?.key === item.key}
            >
              {item.icon}
            </Select.ListItem>
          );
        }}
      </List.Items>
    </Select.Frame>
  );
};
