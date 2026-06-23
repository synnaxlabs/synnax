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

import { View } from "@/layered/view";
import { Version } from "@/version";

export const LoginNav = (): ReactElement => {
  const os = OS.use();
  return (
    <View.Nav.Bar location="top" size="6.5rem" bordered data-tauri-drag-region>
      <Nav.Bar.Start data-tauri-drag-region>
        <View.Window.Controls visibleIfOS="macOS" forceOS={os} />
      </Nav.Bar.Start>
      <Nav.Bar.End data-tauri-drag-region justify="end">
        <Version.Badge />
        <View.Window.Controls visibleIfOS="Windows" forceOS={os} />
      </Nav.Bar.End>
    </View.Nav.Bar>
  );
};
