// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Drift } from "@synnaxlabs/drift";
import { Access, Flex } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { useLinks } from "@/app/links";
import { Mosaic } from "@/app/Mosaic";
import { Nav } from "@/app/nav";
import { Notifications } from "@/app/Notifications";
import { useTriggers } from "@/app/useTriggers";
import { Auth } from "@/feature/auth";
import { Device } from "@/feature/device";
import { Project } from "@/feature/project";
import { Cluster } from "@/platform/cluster";
import { Layout } from "@/platform/layout";
import { Project as PlatformProject } from "@/platform/project";
import { Range } from "@/platform/range";
import { Status } from "@/platform/status";

const SideEffect = (): null => {
  Access.useLoadPermissions({});
  Cluster.useSyncClusterKey();
  Device.useListenForChanges();
  Range.useListenForChanges();
  PlatformProject.useCheckCore();
  Status.useListenForChanges();
  useLinks();
  useTriggers();
  return null;
};

// ProjectSideEffect holds effects that only make sense with an active project. It is
// rendered inside PlatformProject.Guard, so it mounts only once a project is active - layout
// sync and the file-drop importer never run against the select-or-create screen.
const ProjectSideEffect = (): null => {
  Project.useSyncLayout();
  Layout.useDropOutside();
  return null;
};

export const MAIN_LAYOUT_TYPE = Drift.MAIN_WINDOW;

/**
 * The center of it all. This is the main layout for the Synnax Console. Try to keep this
 * component as simple, presentational, and navigable as possible.
 */
export const Main = (): ReactElement => (
  <>
    {/* We need to place notifications here so they are in the proper stacking context */}
    <Notifications />
    <SideEffect />
    <Auth.Guard>
      <PlatformProject.Guard>
        <ProjectSideEffect />
        <Nav.Bar.Top />
        <Flex.Box
          x
          gap="tiny"
          grow
          style={{ paddingRight: "1rem", paddingBottom: "1rem" }}
        >
          <Nav.Bar.Left />
          <Flex.Box gap="tiny" grow style={{ width: 0 }}>
            <Flex.Box x gap="tiny" grow style={{ height: 0 }}>
              <Nav.Drawer.Left />
              <Mosaic />
            </Flex.Box>
            <Nav.Drawer.Bottom />
          </Flex.Box>
        </Flex.Box>
      </PlatformProject.Guard>
    </Auth.Guard>
  </>
);
