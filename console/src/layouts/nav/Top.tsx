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
import { Align, Divider, Nav, OS, Text } from "@synnaxlabs/pluto";
import { Size } from "@synnaxlabs/x";
import { type ReactElement, useEffect, useState } from "react";

import { ChannelServices } from "@/channel/services";
import { Cluster } from "@/cluster";
import { ClusterServices } from "@/cluster/services";
import { Docs } from "@/docs";
import { Hardware } from "@/hardware";
import { LabelServices } from "@/label/services";
import { Layout } from "@/layout";
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

/**
 * NavTop is the top navigation bar for the Synnax Console. Try to keep this component
 * presentational.
 */
export const Top = (): ReactElement => {
  const os = OS.use();
  return (
    <Layout.Nav.Bar location="top" size="6.5rem">
      <Nav.Bar.Start data-tauri-drag-region>
        <Layout.Controls visibleIfOS="macOS" />
        {os === "Windows" && <Logo variant="icon" />}
        <Workspace.Selector />
      </Nav.Bar.Start>
      <Nav.Bar.Content grow justify="center" data-tauri-drag-region>
        <TopPalette />
      </Nav.Bar.Content>
      <Nav.Bar.End justify="end" align="center" data-tauri-drag-region size="small">
        <Version.Badge />
        <Align.Pack>
          <Cluster.Dropdown />
          <Cluster.ConnectionBadge />
        </Align.Pack>
        <Docs.OpenButton />
        <Layout.Controls visibleIfOS="Windows" />
      </Nav.Bar.End>
    </Layout.Nav.Bar>
  );
};
