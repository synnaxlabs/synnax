// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Nav, OS } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { Cluster } from "@/cluster";
import { Docs } from "@/docs";
import { Service } from "@/layered/service";
import { Project } from "@/project";
import { User } from "@/user";
import { Version } from "@/version";

export const Top = (): ReactElement | null => {
  const os = OS.use();
  return (
    <Service.Nav.Bar location="top" size="6.5rem">
      <Nav.Bar.Start data-tauri-drag-region gap="large">
        <Service.Window.Controls visibleIfOS="macOS" forceOS={os} />
        <Project.Selector />
      </Nav.Bar.Start>
      <Nav.Bar.End justify="end" align="center" data-tauri-drag-region gap="small">
        <Version.Badge />
        <User.Badge />
        <Cluster.ConnectionBadge />
        <Docs.OpenButton />
        <Service.Window.Controls visibleIfOS="Windows" forceOS={os} />
      </Nav.Bar.End>
    </Service.Nav.Bar>
  );
};
