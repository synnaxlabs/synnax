// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/layouts/nav/Nav.css";

import { Nav } from "@synnaxlabs/pluto";

import { CSS } from "@/css";
import { Layout } from "@/layout";
import { DRAWER_ITEMS } from "@/layouts/nav/drawerItems";
import { Menu } from "@/layouts/nav/Menu";
import { SIZES } from "@/layouts/nav/sizes";

export const Right = () => {
  const { activeItem, menuItems, onSelect } = Layout.useNavDrawer(
    "right",
    DRAWER_ITEMS,
  );
  const {
    menuItems: bottomMenuItems,
    activeItem: bottomActiveItem,
    onSelect: onBottomSelect,
  } = Layout.useNavDrawer("bottom", DRAWER_ITEMS);
  return (
    <Nav.Bar className={CSS.B("main-nav")} location="right" size={SIZES.side}>
      <Nav.Bar.Content className="console-main-nav__content" size="medium">
        <Menu activeItem={activeItem} onChange={onSelect}>
          {menuItems}
        </Menu>
      </Nav.Bar.Content>
      {bottomMenuItems.length > 0 && (
        <Nav.Bar.End className="console-main-nav__content" bordered>
          <Menu activeItem={bottomActiveItem} onChange={onBottomSelect}>
            {bottomMenuItems}
          </Menu>
        </Nav.Bar.End>
      )}
    </Nav.Bar>
  );
};
