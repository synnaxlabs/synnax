// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement } from "react";

import { Synnax } from "@synnaxlabs/client";
import { Icon, Logo } from "@synnaxlabs/media";
import {
  Divider,
  Nav,
  Menu as PMenu,
  MenuProps as PMenuProps,
  Button,
  useOS,
  Triggers,
  Client,
  Text,
} from "@synnaxlabs/pluto";
import { Location } from "@synnaxlabs/x";

import { ClusterBadge, ClusterToolbar, ConnectionBadge } from "@/cluster";
import { CLUSTER_COMMANDS } from "@/cluster/palette";
import { Controls } from "@/components";
import { CSS } from "@/css";
import { createDocsLayout } from "@/docs";
import {
  NavDrawerItem,
  NavdrawerLocation,
  useNavDrawer,
  NavMenuItem,
  useLayoutPlacer,
} from "@/layout";
import { LAYOUT_COMMANDS } from "@/layout/palette";
import { NAV_SIZES } from "@/layouts/LayoutMain/constants";
import { LINE_COMMANDS } from "@/line/palette";
import { Palette, PaletteTriggerConfig } from "@/palette/Palette";
import { PID_COMMANDS } from "@/pid/palette";
import { ResourcesToolbar } from "@/resources";
import { resourceTypes } from "@/resources/resources";
import { VersionBadge } from "@/version";
import { VisToolbar } from "@/vis";
import { WorkspaceToolbar } from "@/workspace";
import { WORKSPACE_COMMANDS } from "@/workspace/palettte";

import "@/layouts/LayoutMain/Nav.css";

export const NAV_DRAWERS: NavDrawerItem[] = [
  ClusterToolbar,
  ResourcesToolbar,
  WorkspaceToolbar,
  VisToolbar,
];

const DEFAULT_TRIGGER: PaletteTriggerConfig = {
  resource: [["Meta", "P"]],
  command: [["Meta", "Shift", "P"]],
};

const COMMANDS = [
  ...LINE_COMMANDS,
  ...CLUSTER_COMMANDS,
  ...PID_COMMANDS,
  ...WORKSPACE_COMMANDS,
  ...LAYOUT_COMMANDS,
];

const NavTopPalette = (): ReactElement => {
  const client = Client.use() as Synnax;
  return (
    <Palette
      commands={COMMANDS}
      searcher={client?.ontology}
      triggers={DEFAULT_TRIGGER}
      resourceTypes={resourceTypes}
      commandSymbol=">"
    />
  );
};

/**
 * NavTop is the top navigation bar for the Delta UI. Try to keep this component
 * presentational.
 */
export const NavTop = (): ReactElement => {
  const placer = useLayoutPlacer();

  const os = useOS();
  const handleDocs = (): void => placer(createDocsLayout());

  return (
    <Nav.Bar data-tauri-drag-region location="top" size={NAV_SIZES.top}>
      <Nav.Bar.Start className="delta-main-nav-top__start">
        <Controls className="delta-controls--macos" visibleIfOS="MacOS" />
        {os === "Windows" && <Logo className="delta-main-nav-top__logo" />}
      </Nav.Bar.Start>
      <Nav.Bar.Content
        style={{
          position: "absolute",
          left: "25%",
          width: "50%",
          zIndex: 10,
          height: NAV_SIZES.top,
        }}
      >
        <NavTopPalette />
      </Nav.Bar.Content>
      <Nav.Bar.End className="delta-main-nav-top__end">
        <Button.Icon
          size="small"
          onClick={handleDocs}
          tooltip={<Text level="small">Documentation</Text>}
        >
          <Icon.QuestionMark />
        </Button.Icon>
        <Button.Icon size="small" tooltip={<Text level="small">Settings</Text>}>
          <Icon.Settings />
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
} & Omit<PMenuProps, "children">): ReactElement => (
  <PMenu {...props}>
    {children.map(({ key, tooltip, icon }) => (
      <PMenu.Item.Icon
        key={key}
        itemKey={key}
        tooltip={<Text level="small">{tooltip}</Text>}
      >
        {icon}
      </PMenu.Item.Icon>
    ))}
  </PMenu>
);

/**
 * NavLeft is the left navigation drawer for the Delta UI. Try to keep this component
 * presentational.
 */
export const NavLeft = (): ReactElement => {
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
export const NavRight = (): ReactElement | null => {
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
export const NavBottom = (): ReactElement => {
  return (
    <Nav.Bar location="bottom" size={NAV_SIZES.bottom}>
      <Nav.Bar.End className="delta-main-nav-bottom__end">
        <Triggers.Status variant="info" />
        <Divider />
        <VersionBadge level="p" />
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

export const NavDrawer = ({ location, ...props }: NavDrawerProps): ReactElement => {
  const { activeItem, onResize, onSelect } = useNavDrawer(location, NAV_DRAWERS);
  return (
    <Nav.Drawer
      location={location}
      className={CSS(
        CSS.B("main-nav-drawer"),
        CSS.BM("main-nav-drawer", new Location(location).direction.crude)
      )}
      activeItem={activeItem}
      onResize={onResize}
      onSelect={onSelect}
      {...props}
    />
  );
};
