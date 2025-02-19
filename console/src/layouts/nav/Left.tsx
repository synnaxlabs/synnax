// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/layouts/nav/Nav.css";

import { Logo } from "@synnaxlabs/media";
import { Nav, OS } from "@synnaxlabs/pluto";

import { CSS } from "@/css";
import { Layout } from "@/layout";
import { DRAWER_ITEMS } from "@/layouts/nav/drawerItems";
import { Menu } from "@/layouts/nav/Menu";
import { SIZES } from "@/layouts/nav/sizes";

export const Left = () => {
  const { onSelect, menuItems, activeItem } = Layout.useNavDrawer("left", DRAWER_ITEMS);
  const os = OS.use();
  return (
    <Nav.Bar className={CSS.B("main-nav")} location="left" size={SIZES.side}>
      {os !== "Windows" && (
        <Nav.Bar.Start className="console-main-nav-left__start" bordered>
          <Logo className="console-main-nav-left__logo" />
        </Nav.Bar.Start>
      )}
      <Nav.Bar.Content className="console-main-nav__content">
        <Menu activeItem={activeItem} onChange={onSelect}>
          {menuItems}
        </Menu>
      </Nav.Bar.Content>
    </Nav.Bar>
  );
};
