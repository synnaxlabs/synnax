// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Nav as PNav, OS } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { Layout } from "@/layout";
import { Nav } from "@/nav";
import { Version } from "@/version";

export const LoginNav = (): ReactElement => {
  const os = OS.use();
  return (
    <Nav.Bar location="top" size="6.5rem" bordered data-tauri-drag-region>
      <PNav.Bar.Start data-tauri-drag-region>
        <Layout.Controls visibleIfOS="macOS" forceOS={os} />
      </PNav.Bar.Start>
      <PNav.Bar.End data-tauri-drag-region justify="end">
        <Version.Badge />
        <Layout.Controls visibleIfOS="Windows" forceOS={os} />
      </PNav.Bar.End>
    </Nav.Bar>
  );
};
