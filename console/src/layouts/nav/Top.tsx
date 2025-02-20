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
import { Button, Nav, OS, Text } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { ChannelServices } from "@/channel/services";
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
import { Workspace } from "@/workspace";
import { WorkspaceServices } from "@/workspace/services";

const DEFAULT_TRIGGER: Palette.TriggerConfig = {
  command: [["Control", "Shift", "P"]],
  defaultMode: "command",
  resource: [["Control", "P"]],
};

const COMMANDS = [
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

const TopPalette = () => (
  <Palette.Palette
    commands={COMMANDS}
    commandSymbol=">"
    services={SERVICES}
    triggers={DEFAULT_TRIGGER}
  />
);

export const Top = () => {
  const placeLayout = Layout.usePlacer();
  const os = OS.use();
  const handleDocs = useCallback(() => {
    placeLayout(Docs.LAYOUT);
  }, [placeLayout]);
  return (
    <Nav.Bar
      location="top"
      size={SIZES.top}
      className={CSS(CSS.B("main-nav"), CSS.B("main-nav-top"))}
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
        data-tauri-drag-region
      >
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
