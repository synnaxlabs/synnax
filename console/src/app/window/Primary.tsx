// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/app/window/Primary.css";

import { Flex, OS } from "@synnaxlabs/pluto";
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
 */
export const Primary = (): ReactElement => {
  const os = OS.use();
  const fullWidthTop = os === "macOS" && Session.Runtime.ENGINE === "tauri";
  return (
    <>
      <SideEffect />
      <Auth.Guard>
        <Auth.ConnectionGuard>
          <Project.Guard>
            <ProjectSideEffect />
            <div
              className={CSS.cls(
                CSS.BE("main", "workspace"),
                fullWidthTop && CSS.M("full-width-top"),
              )}
            >
              <Nav.Bar.Top />
              <Nav.Bar.Left />
              <Flex.Box y gap="small" className={CSS.BE("main", "content")}>
                <Flex.Box x gap="small" grow className={CSS.BE("main", "row")}>
                  <Nav.Drawer.Left />
                  <Mosaic.Mosaic />
                </Flex.Box>
                <Nav.Drawer.Bottom />
              </Flex.Box>
            </div>
          </Project.Guard>
        </Auth.ConnectionGuard>
      </Auth.Guard>
    </>
  );
};
