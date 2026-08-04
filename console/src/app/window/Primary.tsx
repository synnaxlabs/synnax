// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/app/window/Primary.css";

import { Access, Flex, OS } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { Link } from "@/app/link";
import { Mosaic } from "@/app/mosaic";
import { Nav } from "@/app/nav";
import { Triggers } from "@/app/triggers";
import { Auth } from "@/feature/auth";
import { Device } from "@/feature/device";
import { Panel } from "@/feature/panel";
import { Project } from "@/feature/project";
import { CSS } from "@/platform/css";
import { Session } from "@/session";

const SideEffect = (): null => {
  Access.useLoadPermissions({});
  Device.useListenForChanges();
  Link.useDeep();
  Triggers.use();
  return null;
};

// Tear-off reads the required project selectors, so it mounts inside Project.Guard.
const ProjectSideEffect = (): null => {
  Panel.useTearOff();
  return null;
};

/**
 * The center of it all. This is the main layout for the Synnax Console. Try to keep this
 * component as simple, presentational, and navigable as possible.
 *
 * On macOS desktop the top bar spans the full window so the traffic lights sit above
 * the left rail. Everywhere else the rail runs to the window's top edge and the top
 * bar starts to its right.
 */
export const Primary = (): ReactElement => {
  const os = OS.use();
  const fullWidthTop = os === "macOS" && Session.Runtime.ENGINE === "tauri";
  const column = (
    <Flex.Box gap="tiny" grow className={CSS.BE("main", "column")}>
      <Flex.Box x gap="tiny" grow className={CSS.BE("main", "row")}>
        <Nav.Drawer.Left />
        <Mosaic.Mosaic />
      </Flex.Box>
      <Nav.Drawer.Bottom />
    </Flex.Box>
  );
  return (
    <>
      <SideEffect />
      <Auth.Guard>
        <Auth.ConnectionGuard>
          <Project.Guard>
            <ProjectSideEffect />
            {fullWidthTop ? (
              <>
                <Nav.Bar.Top />
                <Flex.Box x gap="tiny" grow className={CSS.BE("main", "content")}>
                  <Nav.Bar.Left />
                  {column}
                </Flex.Box>
              </>
            ) : (
              <Flex.Box x empty grow>
                <Nav.Bar.Left />
                <Flex.Box y empty grow>
                  <Nav.Bar.Top />
                  <Flex.Box x gap="tiny" grow className={CSS.BE("main", "content")}>
                    {column}
                  </Flex.Box>
                </Flex.Box>
              </Flex.Box>
            )}
          </Project.Guard>
        </Auth.ConnectionGuard>
      </Auth.Guard>
    </>
  );
};
