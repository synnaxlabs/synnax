// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/app/window/Secondary.css";

import { Flex } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { Mosaic } from "@/app/mosaic";
import { Nav } from "@/app/nav";
import { Triggers } from "@/app/triggers";
import { Auth } from "@/feature/auth";
import { Panel } from "@/feature/panel";
import { Project } from "@/feature/project";
import { CSS } from "@/platform/css";

const SideEffect = (): null => {
  Panel.useTearOff();
  Triggers.use();
  return null;
};

/**
 * Secondary is the shell for every non-main window: a panel viewport with the selector
 * strip and the visualization toolbar, and none of the main window's remaining chrome.
 */
export const Secondary = (): ReactElement => (
  <Auth.Guard>
    <Auth.ConnectionGuard nav={false}>
      <Project.Guard>
        <SideEffect />
        <Nav.Bar.Top secondary />
        <Flex.Box gap="tiny" grow className={CSS.B("secondary")}>
          <Mosaic.Mosaic />
          <Nav.Drawer.Bottom />
        </Flex.Box>
      </Project.Guard>
    </Auth.ConnectionGuard>
  </Auth.Guard>
);
