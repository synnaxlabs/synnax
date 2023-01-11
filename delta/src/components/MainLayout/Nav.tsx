// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Divider, Nav, Theming } from "@synnaxlabs/pluto";
import type { NavDrawerItem } from "@synnaxlabs/pluto";

import { Logo } from "../Logo";

import { ClusterBadge, ClusterToolbar, ConnectionBadge } from "@/features/cluster";
import { NavdrawerLocation, useNavDrawer } from "@/features/layout";
import { ResourcesToolbar } from "@/features/resources";
import { VersionBadge } from "@/features/version";
import { VisToolbar, WarpModeToggle } from "@/features/vis";
import { WorkspaceToolbar } from "@/features/workspace";

import "./Nav.css";

export const NAV_SIZES = {
  side: 48,
  top: 42,
  bottom: 32,
};

export const NAV_DRAWERS: NavDrawerItem[] = [
  ClusterToolbar,
  ResourcesToolbar,
  WorkspaceToolbar,
  VisToolbar,
];

/**
 * NavTop is the top navigation bar for the Delta UI. Try to keep this component
 * presentational.
 */
export const NavTop = (): JSX.Element => (
  <Nav.Bar data-tauri-drag-region location="top" size={NAV_SIZES.top}>
    <Nav.Bar.End style={{ padding: "0 2rem" }}>
      <WarpModeToggle />
    </Nav.Bar.End>
  </Nav.Bar>
);

/**
 * NavLeft is the left navigation drawer for the Delta UI. Try to keep this component
 * presentational.
 */
export const NavLeft = (): JSX.Element => {
  const { onSelect, menuItems } = useNavDrawer("left", NAV_DRAWERS);
  return (
    <Nav.Bar location="left" size={NAV_SIZES.side}>
      <Nav.Bar.Start className="delta-main-nav-left__start" bordered>
        <Logo className="delta-main-nav-left__logo" />
      </Nav.Bar.Start>
      <Nav.Bar.Content>
        <Nav.Menu onSelect={onSelect} items={menuItems} />
      </Nav.Bar.Content>
      <Nav.Bar.End className="delta-main-nav-left__end" bordered>
        <Theming.Switch />
      </Nav.Bar.End>
    </Nav.Bar>
  );
};

/**
 * NavRight is the right navigation bar for the Delta UI. Try to keep this component
 * presentational.
 */
export const NavRight = (): JSX.Element | null => {
  const { menuItems, onSelect } = useNavDrawer("right", NAV_DRAWERS);
  const { menuItems: bottomMenuItems, onSelect: onBottomSelect } = useNavDrawer(
    "bottom",
    NAV_DRAWERS
  );
  return (
    <Nav.Bar location="right" size={NAV_SIZES.side}>
      <Nav.Bar.Content>
        <Nav.Menu items={menuItems} onSelect={onSelect} />
      </Nav.Bar.Content>
      {bottomMenuItems.length > 0 && (
        <Nav.Bar.End>
          <Nav.Menu items={bottomMenuItems} onSelect={onBottomSelect} />
        </Nav.Bar.End>
      )}
    </Nav.Bar>
  );
};

/**
 * NavBottom is the bottom navigation bar for the Delta UI. Try to keep this component
 * presentational.
 */
export const NavBottom = (): JSX.Element => {
  return (
    <Nav.Bar location="bottom" size={NAV_SIZES.bottom}>
      <Nav.Bar.End className="delta-main-nav-bottom__end">
        <Divider />
        <VersionBadge />
        <Divider />
        <ClusterBadge />
        <Divider />
        <ConnectionBadge />
      </Nav.Bar.End>
    </Nav.Bar>
  );
};

export interface NavDrawerProps {
  location: NavdrawerLocation;
}

export const NavDrawer = ({ location }: NavDrawerProps): JSX.Element => {
  const { activeItem } = useNavDrawer(location, NAV_DRAWERS);
  return (
    <Nav.Drawer
      location={location}
      activeItem={activeItem}
      style={{ flexShrink: 0, flexGrow: 0 }}
    />
  );
};
