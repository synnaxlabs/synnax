// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/app/window/Primary.css";

import { Access, Flex } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { Link } from "@/app/link";
import { Mosaic } from "@/app/mosaic";
import { Nav } from "@/app/nav";
import { Triggers } from "@/app/triggers";
import { Auth } from "@/feature/auth";
import { Cluster } from "@/feature/cluster";
import { Device } from "@/feature/device";
import { Project } from "@/feature/project";
import { CSS } from "@/platform/css";
import { Range } from "@/platform/range";
import { Status } from "@/platform/status";

const SideEffect = (): null => {
  Access.useLoadPermissions({});
  Cluster.useSyncClusterKey();
  Device.useListenForChanges();
  Range.useListenForChanges();
  Project.useCheckCore();
  Status.useListenForChanges();
  Link.useDeep();
  Triggers.use();
  return null;
};

/**
 * The center of it all. This is the main layout for the Synnax Console. Try to keep this
 * component as simple, presentational, and navigable as possible.
 */
export const Primary = (): ReactElement => (
  <>
    <SideEffect />
    <Auth.Guard>
      <Project.Guard>
        <Nav.Bar.Top />
        <Flex.Box x gap="tiny" grow className={CSS.BE("main", "content")}>
          <Nav.Bar.Left />
          <Flex.Box gap="tiny" grow className={CSS.BE("main", "column")}>
            <Flex.Box x gap="tiny" grow className={CSS.BE("main", "row")}>
              <Nav.Drawer.Left />
              <Mosaic.Mosaic />
            </Flex.Box>
            <Nav.Drawer.Bottom />
          </Flex.Box>
        </Flex.Box>
      </Project.Guard>
    </Auth.Guard>
  </>
);
