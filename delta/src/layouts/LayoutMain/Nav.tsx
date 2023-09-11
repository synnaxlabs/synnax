// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { Icon, Logo } from "@synnaxlabs/media";
import {
  Divider,
  Nav,
  Menu as PMenu,
  Button,
  OS,
  Triggers,
  Synnax,
  Text,
} from "@synnaxlabs/pluto";
import { location } from "@synnaxlabs/x";

import { Cluster } from "@/cluster";
import { Controls } from "@/components";
import { CSS } from "@/css";
import { Docs } from "@/docs";
import { Layout } from "@/layout";
import { NAV_SIZES } from "@/layouts/LayoutMain/constants";
import { LinePlot } from "@/lineplot";
import { Palette } from "@/palette/Palette";
import { type TriggerConfig } from "@/palette/types";
import { PID } from "@/pid";
import { Resources } from "@/resources";
import { Version } from "@/version";
import { Vis } from "@/vis";
import { Workspace } from "@/workspace";

import "@/layouts/LayoutMain/Nav.css";

export const NAV_DRAWERS: Layout.NavDrawerItem[] = [
  Cluster.Toolbar,
  Resources.Toolbar,
  Workspace.Toolbar,
  Vis.Toolbar,
];

const DEFAULT_TRIGGER: TriggerConfig = {
  defaultMode: "command",
  resource: [["Control", "P"]],
  command: [["Control", "Shift", "P"]],
};

const COMMANDS = [
  ...LinePlot.COMMANDS,
  ...Layout.COMMANDS,
  ...PID.COMMANDS,
  ...Docs.COMMANDS,
  ...Workspace.COMMANDS,
  ...Cluster.COMMANDS,
];

const NavTopPalette = (): ReactElement => {
  const client = Synnax.use();
  return (
    <Palette
      commands={COMMANDS}
      searcher={client?.ontology}
      triggers={DEFAULT_TRIGGER}
      resourceTypes={Resources.types}
      commandSymbol=">"
    />
  );
};

/**
 * NavTop is the top navigation bar for the Delta UI. Try to keep this component
 * presentational.
 */
export const NavTop = (): ReactElement => {
  const placer = Layout.usePlacer();

  const os = OS.use();
  const handleDocs = (): void => {
    placer(Docs.createLayout());
  };

  return (
    <Nav.Bar data-tauri-drag-region location="top" size={NAV_SIZES.top}>
      <Nav.Bar.Start className="delta-main-nav-top__start">
        <Controls className="delta-controls--macos" visibleIfOS="MacOS" />
        {os === "Windows" && (
          <Logo className="delta-main-nav-top__logo" variant="icon" />
        )}
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
          size="medium"
          onClick={handleDocs}
          tooltip={<Text.Text level="small">Documentation</Text.Text>}
        >
          <Icon.QuestionMark />
        </Button.Icon>
        <Button.Icon
          size="medium"
          tooltip={<Text.Text level="small">Settings</Text.Text>}
        >
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
  children: Layout.NavMenuItem[];
} & Omit<PMenu.MenuProps, "children">): ReactElement => (
  <PMenu.Menu {...props}>
    {children.map(({ key, tooltip, icon }) => (
      <PMenu.Item.Icon
        key={key}
        itemKey={key}
        size="large"
        tooltip={<Text.Text level="small">{tooltip}</Text.Text>}
      >
        {icon}
      </PMenu.Item.Icon>
    ))}
  </PMenu.Menu>
);

/**
 * NavLeft is the left navigation drawer for the Delta UI. Try to keep this component
 * presentational.
 */
export const NavLeft = (): ReactElement => {
  const { onSelect, menuItems } = Layout.useNavDrawer("left", NAV_DRAWERS);
  const os = OS.use();
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
  const { menuItems, onSelect } = Layout.useNavDrawer("right", NAV_DRAWERS);
  const { menuItems: bottomMenuItems, onSelect: onBottomSelect } = Layout.useNavDrawer(
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
      <Nav.Bar.Start>
        <Vis.NavControls />
      </Nav.Bar.Start>
      <Nav.Bar.End className="delta-main-nav-bottom__end">
        <Triggers.Status variant="info" />
        <Divider.Divider />
        <Version.Badge level="p" />
        <Divider.Divider />
        <Cluster.NameBadge />
        <Divider.Divider />
        <Cluster.ConnectionBadge />
      </Nav.Bar.End>
    </Nav.Bar>
  );
};

export interface NavDrawerProps {
  location: Layout.NavdrawerLocation;
}

export const NavDrawer = ({ location: l, ...props }: NavDrawerProps): ReactElement => {
  const { activeItem, onResize, onSelect } = Layout.useNavDrawer(l, NAV_DRAWERS);
  return (
    <Nav.Drawer
      location={l}
      className={CSS(
        CSS.B("main-nav-drawer"),
        CSS.BM("main-nav-drawer", location.direction(l))
      )}
      activeItem={activeItem}
      onResize={onResize}
      onSelect={onSelect}
      {...props}
    />
  );
};
