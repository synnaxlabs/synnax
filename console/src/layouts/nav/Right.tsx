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
import { type ReactElement } from "react";

import { CSS } from "@/css";
import { Layout } from "@/layout";
import { NAV_DRAWER_ITEMS } from "@/layouts/nav/drawerItems";
import { Menu } from "@/layouts/nav/Menu";
import { SIZES } from "@/layouts/nav/sizes";

export const Right = (): ReactElement | null => {
  const { activeItem, menuItems, onSelect } = Layout.useNavDrawer(
    "right",
    NAV_DRAWER_ITEMS,
  );

  if (menuItems.length === 0) return null;

  return (
    <Nav.Bar className={CSS.B("main-nav")} location="right" size={SIZES.side}>
      <Nav.Bar.Content className="console-main-nav__content" size="medium">
        <Menu activeItem={activeItem} onChange={onSelect}>
          {menuItems}
        </Menu>
      </Nav.Bar.Content>
    </Nav.Bar>
  );
};
