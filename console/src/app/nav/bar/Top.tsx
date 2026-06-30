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

import { Cluster } from "@/cluster";
import { Service } from "@/service";
import { Docs } from "@/service/docs";
import { User } from "@/service/user";
import { Version } from "@/service/version";
import { Project } from "@/project";

export const Top = (): ReactElement | null => (
  <Service.Nav.Bar location="top" size="6.5rem">
    <Nav.Bar.Start data-tauri-drag-region gap="large">
      <Service.Window.Controls visibleIfOS="macOS" />
      <Project.Selector />
    </Nav.Bar.Start>
    <Nav.Bar.Content data-tauri-drag-region full="x" />
    <Nav.Bar.End justify="end" align="center" data-tauri-drag-region gap="small">
      <Version.Badge />
      <User.Badge />
      <Cluster.ConnectionBadge />
      <Docs.OpenButton />
      <Service.Window.Controls visibleIfOS="Windows" />
    </Nav.Bar.End>
  </Service.Nav.Bar>
);
