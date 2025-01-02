// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/layouts/Nav.css";

import { Icon, Logo } from "@synnaxlabs/media";
import { Button, Divider, Nav, OS, Text } from "@synnaxlabs/pluto";
import { Size } from "@synnaxlabs/x";
import { type ReactElement, useEffect, useState } from "react";

import { Channel } from "@/channel";
import { Cluster } from "@/cluster";
import { ClusterServices } from "@/cluster/services";
import { Controls } from "@/components";
import { NAV_DRAWERS, NavMenu } from "@/components/nav/Nav";
import { CSS } from "@/css";
import { Docs } from "@/docs";
import { LabJack } from "@/hardware/labjack";
import { NI } from "@/hardware/ni";
import { OPC } from "@/hardware/opc";
import { LabelServices } from "@/label/services";
import { Layout } from "@/layout";
import { NAV_SIZES } from "@/layouts/constants";
import { LinePlotServices } from "@/lineplot/services";
import { LogServices } from "@/log/services";
import { Palette } from "@/palette";
import { Persist } from "@/persist";
import { RangeServices } from "@/range/services";
import { SchematicServices } from "@/schematic/services";
import { SERVICES } from "@/services";
import { TableServices } from "@/table/services";
import { UserServices } from "@/user/services";
import { Version } from "@/version";
import { Vis } from "@/vis";
import { Workspace } from "@/workspace";

const DEFAULT_TRIGGER: Palette.TriggerConfig = {
  defaultMode: "command",
  resource: [["Control", "P"]],
  command: [["Control", "Shift", "P"]],
};

const COMMANDS = [
  ...LinePlotServices.COMMANDS,
  ...Layout.COMMANDS,
  ...SchematicServices.COMMANDS,
  ...Docs.COMMANDS,
  ...Workspace.COMMANDS,
  ...ClusterServices.COMMANDS,
  ...RangeServices.COMMANDS,
  ...LabJack.COMMANDS,
  ...OPC.COMMANDS,
  ...Persist.COMMANDS,
  ...NI.COMMANDS,
  ...Channel.COMMANDS,
  ...LabelServices.COMMANDS,
  ...UserServices.COMMANDS,
  ...LogServices.COMMANDS,
  ...TableServices.COMMANDS,
];

const NavTopPalette = (): ReactElement => (
  <Palette.Palette
    commands={COMMANDS}
    triggers={DEFAULT_TRIGGER}
    services={SERVICES}
    commandSymbol=">"
  />
);

/**
 * NavTop is the top navigation bar for the Synnax Console. Try to keep this component
 * presentational.
 */
export const NavTop = (): ReactElement => {
  const place = Layout.usePlacer();

  const os = OS.use();
  const handleDocs = (): void => {
    place(Docs.createLayout());
  };

  return (
    <Nav.Bar
      location="top"
      size={NAV_SIZES.top}
      className={CSS(CSS.B("main-nav"), CSS.B("main-nav-top"))}
    >
      <Nav.Bar.Start className="console-main-nav-top__start" data-tauri-drag-region>
        <Controls className="console-controls--macos" visibleIfOS="MacOS" />
        {os === "Windows" && (
          <Logo className="console-main-nav-top__logo" variant="icon" />
        )}
        <Workspace.Selector />
      </Nav.Bar.Start>
      <Nav.Bar.Content
        grow
        justify="center"
        className="console-main-nav-top__center"
        data-tauri-drag-region
      >
        <NavTopPalette />
      </Nav.Bar.Content>
      <Nav.Bar.End
        className="console-main-nav-top__end"
        justify="end"
        data-tauri-drag-region
      >
        <Button.Icon
          size="medium"
          onClick={handleDocs}
          tooltip={<Text.Text level="small">Documentation</Text.Text>}
        >
          <Icon.QuestionMark />
        </Button.Icon>
        <Controls className="console-controls--windows" visibleIfOS="Windows" />
      </Nav.Bar.End>
    </Nav.Bar>
  );
};

/**
 * NavLeft is the left navigation drawer for the Synnax Console. Try to keep this component
 * presentational.
 */
export const NavLeft = (): ReactElement => {
  const { onSelect, menuItems, activeItem } = Layout.useNavDrawer("left", NAV_DRAWERS);
  const os = OS.use();
  return (
    <Nav.Bar className={CSS.B("main-nav")} location="left" size={NAV_SIZES.side}>
      {os !== "Windows" && (
        <Nav.Bar.Start className="console-main-nav-left__start" bordered>
          <Logo className="console-main-nav-left__logo" />
        </Nav.Bar.Start>
      )}
      <Nav.Bar.Content className="console-main-nav__content">
        <NavMenu activeItem={activeItem} onChange={onSelect}>
          {menuItems}
        </NavMenu>
      </Nav.Bar.Content>
    </Nav.Bar>
  );
};

/**
 * NavRight is the right navigation bar for the Synnax Console. Try to keep this component
 * presentational.
 */
export const NavRight = (): ReactElement | null => {
  const { activeItem, menuItems, onSelect } = Layout.useNavDrawer("right", NAV_DRAWERS);
  const {
    menuItems: bottomMenuItems,
    activeItem: bottomActiveItem,
    onSelect: onBottomSelect,
  } = Layout.useNavDrawer("bottom", NAV_DRAWERS);
  return (
    <Nav.Bar className={CSS.B("main-nav")} location="right" size={NAV_SIZES.side}>
      <Nav.Bar.Content className="console-main-nav__content" size="medium">
        <NavMenu activeItem={activeItem} onChange={onSelect}>
          {menuItems}
        </NavMenu>
      </Nav.Bar.Content>
      {bottomMenuItems.length > 0 && (
        <Nav.Bar.End className="console-main-nav__content" bordered>
          <NavMenu activeItem={bottomActiveItem} onChange={onBottomSelect}>
            {bottomMenuItems}
          </NavMenu>
        </Nav.Bar.End>
      )}
    </Nav.Bar>
  );
};

interface MemoryUsage {
  used: Size;
  total: Size;
}

interface PerformanceAPI {
  memory: MemoryInfo;
}

interface MemoryInfo {
  usedJSHeapSize: number;
  jsHeapSizeLimit: number;
}

const MemoryBadge = (): ReactElement | null => {
  const [memory, setMemory] = useState<MemoryUsage>({
    used: Size.ZERO,
    total: Size.ZERO,
  });
  const displayMemory = "memory" in performance;
  useEffect(() => {
    const interval = setInterval(() => {
      if ("memory" in performance) {
        const { memory } = performance as PerformanceAPI;
        setMemory({
          used: Size.bytes(memory.usedJSHeapSize),
          total: Size.bytes(memory.jsHeapSizeLimit),
        });
      }
    }, 1000);
    return (): void => clearInterval(interval);
  });
  if (displayMemory === false) return null;
  return (
    <>
      <Divider.Divider />
      <Text.Text level="p" style={{ padding: "0 2rem" }}>
        {memory.used.truncate(Size.MEGABYTE).toString()} /
        {memory.total.truncate(Size.MEGABYTE).toString()}
      </Text.Text>
    </>
  );
};

/**
 * NavBottom is the bottom navigation bar for the Synnax Console. Try to keep this component
 * presentational.
 */
export const NavBottom = (): ReactElement => (
  <Nav.Bar className={CSS.B("main-nav")} location="bottom" size={NAV_SIZES.bottom}>
    <Nav.Bar.Start>
      <Vis.NavControls />
    </Nav.Bar.Start>
    <Nav.Bar.End className="console-main-nav-bottom__end" empty>
      <MemoryBadge />
      <Divider.Divider />
      <Version.Badge />
      <Divider.Divider />
      <Cluster.Dropdown />
      <Divider.Divider />
      <Cluster.ConnectionBadge />
    </Nav.Bar.End>
  </Nav.Bar>
);
