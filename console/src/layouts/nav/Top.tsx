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

import { Cluster } from "@/cluster";
import { Docs } from "@/docs";
import { Layout } from "@/layout";
import { LOGO_LOCATION } from "@/layouts/nav/logo";
import { Palette } from "@/palette";
import { User } from "@/user";
import { Version } from "@/version";
import { Workspace } from "@/workspace";

const PALETTE_TRIGGER_CONFIG: Palette.TriggerConfig = {
  command: [["Control", "Shift", "P"]],
  defaultMode: "command",
  search: [["Control", "P"]],
};

const TopPalette = (): ReactElement => (
  <Palette.Palette commandSymbol=">" triggerConfig={PALETTE_TRIGGER_CONFIG} />
);

export const Top = (): ReactElement | null => {
  const os = OS.use();
  return (
    <Layout.Nav.Bar location="top" size="6.5rem">
      <Nav.Bar.Start data-tauri-drag-region gap="large">
        <Layout.Controls visibleIfOS="macOS" forceOS={os} />
        {LOGO_LOCATION === "top" && <Logo variant="icon" />}
        <Workspace.Selector />
      </Nav.Bar.Start>
      <Nav.Bar.Center grow justify="center" data-tauri-drag-region>
        <TopPalette />
      </Nav.Bar.Center>
      <Nav.Bar.End justify="end" align="center" data-tauri-drag-region gap="small">
        <Version.Badge />
        <User.Badge />
        <Cluster.ConnectionBadge />
        <Docs.OpenButton />
        <Layout.Controls visibleIfOS="Windows" forceOS={os} />
      </Nav.Bar.End>
    </Layout.Nav.Bar>
  );
};
