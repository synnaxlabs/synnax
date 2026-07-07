// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Nav } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { Cluster } from "@/feature/cluster";
import { Docs } from "@/feature/docs";
import { Panel } from "@/feature/panel";
import { Nav as PlatformNav } from "@/platform/nav";
import { Project } from "@/platform/project";
import { User } from "@/platform/user";
import { Version } from "@/platform/version";
import { Window } from "@/platform/window";

export const Top = (): ReactElement | null => (
  <PlatformNav.Bar location="top" size="6.5rem">
    <Nav.Bar.Start data-tauri-drag-region gap="large">
      <Window.Controls visibleIfOS="macOS" />
      <Project.Selector />
      <Panel.Selector />
    </Nav.Bar.Start>
    <Nav.Bar.Content data-tauri-drag-region full="x" />
    <Nav.Bar.End justify="end" align="center" data-tauri-drag-region gap="small">
      <Version.Badge />
      <User.Badge />
      <Cluster.ConnectionBadge />
      <Docs.OpenButton />
      <Window.Controls visibleIfOS="Windows" />
    </Nav.Bar.End>
  </PlatformNav.Bar>
);
