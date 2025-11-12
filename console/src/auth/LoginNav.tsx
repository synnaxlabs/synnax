// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Nav, OS } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { Layout } from "@/layout";

export const LoginNav = (): ReactElement => {
  const os = OS.use();
  return (
    <Layout.Nav.Bar location="top" size="6.5rem" bordered data-tauri-drag-region>
      <Nav.Bar.Start data-tauri-drag-region>
        <Layout.Controls visibleIfOS="macOS" forceOS={os} />
      </Nav.Bar.Start>
      <Nav.Bar.End data-tauri-drag-region justify="end">
        <Layout.Controls visibleIfOS="Windows" forceOS={os} />
      </Nav.Bar.End>
    </Layout.Nav.Bar>
  );
};
