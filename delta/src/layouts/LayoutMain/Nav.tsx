// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon, Logo } from "@synnaxlabs/media";
import {
  Divider,
  Nav,
  Menu as PMenu,
  MenuProps as PMenuProps,
  Button,
  useOS,
} from "@synnaxlabs/pluto";
import { locToDir } from "@synnaxlabs/x";

import { NAV_SIZES } from "./constants";

import { Controls } from "@/components";
import { CSS } from "@/css";
import { ClusterBadge, ClusterToolbar, ConnectionBadge } from "@/features/cluster";
import { createDocsLayout } from "@/features/docs";
import {
  NavDrawerItem,
  NavdrawerLocation,
  useNavDrawer,
  NavMenuItem,
  useLayoutPlacer,
} from "@/features/layout";
import { ResourcesToolbar } from "@/features/resources";
import { VersionBadge } from "@/features/version";
import { VisToolbar } from "@/features/vis";
import { WorkspaceToolbar } from "@/features/workspace";

import "./Nav.css";

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
export const NavTop = (): JSX.Element => {
  const placer = useLayoutPlacer();

  const os = useOS();
  const handleDocs = (): void => placer(createDocsLayout());

  return (
    <Nav.Bar data-tauri-drag-region location="top" size={NAV_SIZES.top}>
      <Nav.Bar.Start className="delta-main-nav-top__start">
        <Controls className="delta-controls--macos" visibleIfOS="MacOS" />
        {os === "Windows" && <Logo className="delta-main-nav-top__logo" />}
      </Nav.Bar.Start>
      <Nav.Bar.End className="delta-main-nav-top__end">
        <Button.Icon size="small" onClick={handleDocs}>
          <Icon.QuestionMark />
        </Button.Icon>
        <Controls className="delta-controls--windows" visibleIfOS="Windows" />
      </Nav.Bar.End>
    </Nav.Bar>
  );
};

export const NavMenu = ({
  children,
  ...props
}: {
  children: NavMenuItem[];
} & Omit<PMenuProps, "children">): JSX.Element => (
  <PMenu {...props}>
    {children.map((item) => (
      <PMenu.Item.Icon key={item.key} itemKey={item.key}>
        {item.icon}
      </PMenu.Item.Icon>
    ))}
  </PMenu>
);

/**
 * NavLeft is the left navigation drawer for the Delta UI. Try to keep this component
 * presentational.
 */
export const NavLeft = (): JSX.Element => {
  const { onSelect, menuItems } = useNavDrawer("left", NAV_DRAWERS);
  const os = useOS();
  return (
    <Nav.Bar location="left" size={NAV_SIZES.side}>
      {os !== "Windows" && (
        <Nav.Bar.Start className="delta-main-nav-left__start" bordered>
          <Logo className="delta-main-nav-left__logo" />
        </Nav.Bar.Start>
      )}
      <Nav.Bar.Content className="delta-main-nav__content">
        <NavMenu onChange={onSelect}>{menuItems}</NavMenu>
      </Nav.Bar.Content>
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
      <Nav.Bar.Content className="delta-main-nav__content" size="small">
        <NavMenu onChange={onSelect}>{menuItems}</NavMenu>
      </Nav.Bar.Content>
      {bottomMenuItems.length > 0 && (
        <Nav.Bar.End className="delta-main-nav__content" bordered>
          <NavMenu onChange={onBottomSelect}>{bottomMenuItems}</NavMenu>
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
      <Nav.Bar.End className="delta-main-nav-bottom__end" bordered>
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

export const NavDrawer = ({ location, ...props }: NavDrawerProps): JSX.Element => {
  const { activeItem, onResize, onSelect } = useNavDrawer(location, NAV_DRAWERS);
  return (
    <Nav.Drawer
      location={location}
      className={CSS(
        CSS.B("main-nav-drawer"),
        CSS.BM("main-nav-drawer", locToDir(location))
      )}
      activeItem={activeItem}
      onResize={onResize}
      onSelect={onSelect}
      {...props}
    />
  );
};
