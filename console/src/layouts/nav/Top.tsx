// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/layouts/nav/Nav.css";

import { Nav as PNav, OS } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { Cluster } from "@/cluster";
import { Docs } from "@/docs";
import { Controls } from "@/layouts/Controls";
import { Nav } from "@/nav";
import { Panel } from "@/panel";
import { Project } from "@/project";
import { User } from "@/user";
import { Version } from "@/version";

export const Top = (): ReactElement | null => {
  const os = OS.use();
  return (
    <Nav.Bar location="top" size="6.5rem">
      <PNav.Bar.Start data-tauri-drag-region gap="large">
        <Controls visibleIfOS="macOS" forceOS={os} />
        <Project.Selector />
        <Panel.PanelTabs />
      </PNav.Bar.Start>
      <PNav.Bar.End justify="end" align="center" data-tauri-drag-region gap="small">
        <Version.Badge />
        <User.Badge />
        <Cluster.ConnectionBadge />
        <Docs.OpenButton />
        <Controls visibleIfOS="Windows" forceOS={os} />
      </PNav.Bar.End>
    </Nav.Bar>
  );
};
