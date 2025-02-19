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
import { location } from "@synnaxlabs/x";

import { CSS } from "@/css";
import { Layout } from "@/layout";
import { DRAWER_ITEMS } from "@/layouts/nav/drawerItems";

export interface DrawerProps {
  location: Layout.NavDrawerLocation;
}

export const Drawer = ({ location: loc }: DrawerProps) => {
  const { activeItem, onResize, onSelect } = Layout.useNavDrawer(loc, DRAWER_ITEMS);
  return (
    <Nav.Drawer
      location={loc}
      className={CSS(
        CSS.B("main-nav-drawer"),
        CSS.BM("main-nav-drawer", location.direction(loc)),
      )}
      activeItem={activeItem}
      onResize={onResize}
      onSelect={onSelect}
    />
  );
};
