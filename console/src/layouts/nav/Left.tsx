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
import { type ReactElement } from "react";

import { CSS } from "@/css";
import { Layout } from "@/layout";
import { NAV_DRAWER_ITEMS } from "@/layouts/nav/drawerItems";
import { Menu } from "@/layouts/nav/Menu";
import { SIZES } from "@/layouts/nav/sizes";

/**
 * NavLeft is the left navigation drawer for the Synnax Console. Try to keep this component
 * presentational.
 */
export const Left = (): ReactElement => {
  const { onSelect, menuItems, activeItem, onStartHover, onStopHover } =
    Layout.useNavDrawer("left", NAV_DRAWER_ITEMS);
  const os = OS.use();
  const {
    menuItems: bottomMenuItems,
    activeItem: bottomActiveItem,
    onSelect: onBottomSelect,
    onStartHover: onBottomStartHover,
    onStopHover: onBottomStopHover,
  } = Layout.useNavDrawer("bottom", NAV_DRAWER_ITEMS);
  return (
    <Nav.Bar
      className={CSS.B("main-nav")}
      location="left"
      size={SIZES.side}
      bordered={false}
    >
      {os !== "Windows" && (
        <Nav.Bar.Start className="console-main-nav-left__start" bordered>
          <Logo className="console-main-nav-left__logo" />
        </Nav.Bar.Start>
      )}
      <Nav.Bar.Content className="console-main-nav__content">
        <Menu
          activeItem={activeItem}
          onChange={onSelect}
          onStartHover={onStartHover}
          onStopHover={onStopHover}
        >
          {menuItems}
        </Menu>
      </Nav.Bar.Content>
      {bottomMenuItems.length > 0 && (
        <Nav.Bar.End className="console-main-nav__content" bordered>
          <Menu
            activeItem={bottomActiveItem}
            onChange={onBottomSelect}
            onStartHover={onBottomStartHover}
            onStopHover={onBottomStopHover}
          >
            {bottomMenuItems}
          </Menu>
        </Nav.Bar.End>
      )}
    </Nav.Bar>
  );
};
