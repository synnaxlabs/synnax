// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/layouts/nav/Nav.css";

import { Icon, Logo } from "@synnaxlabs/media";
import { Align, Button, Divider, Nav, OS, Text } from "@synnaxlabs/pluto";
import { Size } from "@synnaxlabs/x";
import { type ReactElement, useEffect, useState } from "react";

import { ChannelServices } from "@/channel/services";
import { Cluster } from "@/cluster";
import { ClusterServices } from "@/cluster/services";
import { CSS } from "@/css";
import { Docs } from "@/docs";
import { Hardware } from "@/hardware";
import { LabelServices } from "@/label/services";
import { Layout } from "@/layout";
import { SIZES } from "@/layouts/nav/sizes";
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
import { Workspace } from "@/workspace";
import { WorkspaceServices } from "@/workspace/services";

const PALETTE_TRIGGER_CONFIG: Palette.TriggerConfig = {
  command: [["Control", "Shift", "P"]],
  defaultMode: "command",
  search: [["Control", "P"]],
};

const COMMANDS: Palette.Command[] = [
  ...ChannelServices.COMMANDS,
  ...ClusterServices.COMMANDS,
  ...Docs.COMMANDS,
  ...Hardware.COMMANDS,
  ...LabelServices.COMMANDS,
  ...Layout.COMMANDS,
  ...LinePlotServices.COMMANDS,
  ...LogServices.COMMANDS,
  ...Persist.COMMANDS,
  ...RangeServices.COMMANDS,
  ...SchematicServices.COMMANDS,
  ...TableServices.COMMANDS,
  ...UserServices.COMMANDS,
  ...WorkspaceServices.COMMANDS,
];

const TopPalette = (): ReactElement => (
  <Palette.Palette
    commands={COMMANDS}
    commandSymbol=">"
    services={SERVICES}
    triggerConfig={PALETTE_TRIGGER_CONFIG}
  />
);

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
  const hasMemoryDisplayed = "memory" in performance;
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
    return () => clearInterval(interval);
  });
  return !hasMemoryDisplayed ? null : (
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
 * NavTop is the top navigation bar for the Synnax Console. Try to keep this component
 * presentational.
 */
export const Top = (): ReactElement => {
  const placeLayout = Layout.usePlacer();

  const os = OS.use();
  const handleDocs = (): void => {
    placeLayout(Docs.LAYOUT);
  };

  return (
    <Nav.Bar
      location="top"
      size={SIZES.top}
      className={CSS(CSS.B("main-nav"), CSS.B("main-nav-top"))}
      bordered={false}
    >
      <Nav.Bar.Start className="console-main-nav-top__start" data-tauri-drag-region>
        <Layout.Controls className="console-controls--macos" visibleIfOS="macOS" />
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
        <TopPalette />
      </Nav.Bar.Content>
      <Nav.Bar.End
        className="console-main-nav-top__end"
        justify="end"
        align="center"
        data-tauri-drag-region
        size="small"
      >
        <MemoryBadge />
        <Version.Badge />
        <Align.Pack style={{ "--pluto-pack-br": "0.5rem" }}>
          <Cluster.Dropdown />
          <Cluster.ConnectionBadge />
        </Align.Pack>
        <Button.Icon
          size="medium"
          onClick={handleDocs}
          tooltip={<Text.Text level="small">Documentation</Text.Text>}
        >
          <Icon.QuestionMark />
        </Button.Icon>
        <Layout.Controls className="console-controls--windows" visibleIfOS="Windows" />
      </Nav.Bar.End>
    </Nav.Bar>
  );
};
