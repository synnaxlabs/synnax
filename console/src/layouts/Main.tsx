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

import { ArcServices } from "@/arc/services";
import { Auth } from "@/auth";
import { ChannelServices } from "@/channel/services";
import { Cluster } from "@/cluster";
import { ClusterServices } from "@/cluster/services";
import { Hardware } from "@/hardware";
import { Schematic } from "@/layered/service/schematic";
import { Layout } from "@/layout";
import { Mosaic } from "@/layouts/Mosaic";
import { Nav } from "@/layouts/nav";
import { Notifications } from "@/layouts/Notifications";
import { useTriggers } from "@/layouts/useTriggers";
import { LinePlotServices } from "@/lineplot/services";
import { Link } from "@/link";
import { LogServices } from "@/log/services";
import { Project } from "@/project";
import { ProjectServices } from "@/project/services";
import { Range } from "@/range";
import { RangeServices } from "@/range/services";
import { Status } from "@/status";
import { TableServices } from "@/table/services";

const LINK_HANDLERS: Record<string, Link.Handler> = {
  arc: ArcServices.handleLink,
  channel: ChannelServices.handleLink,
  ...Hardware.LINK_HANDLERS,
  lineplot: LinePlotServices.handleLink,
  log: LogServices.handleLink,
  range: RangeServices.handleLink,
  schematic: Schematic.handleLink,
  table: TableServices.handleLink,
  project: ProjectServices.handleLink,
};

const SideEffect = (): null => {
  Access.useLoadPermissions({});
  Cluster.useSyncClusterKey();
  Hardware.Device.useListenForChanges();
  Range.useListenForChanges();
  Project.useCheckCore();
  Status.useListenForChanges();
  Link.useDeep(ClusterServices.handleLink, LINK_HANDLERS);
  useTriggers();
  return null;
};

// ProjectSideEffect holds effects that only make sense with an active project. It is
// rendered inside Project.Guard, so it mounts only once a project is active - layout
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
      <Project.Guard>
        <ProjectSideEffect />
        <Nav.Top />
        <Flex.Box
          x
          gap="tiny"
          grow
          style={{ paddingRight: "1rem", paddingBottom: "1rem" }}
        >
          <Nav.Left />
          <Flex.Box gap="tiny" grow style={{ width: 0 }}>
            <Flex.Box x gap="tiny" grow style={{ height: 0 }}>
              <Layout.Nav.Drawer location="left" menuItems={Nav.DRAWER_ITEMS} />
              <Mosaic />
            </Flex.Box>
            <Layout.Nav.Drawer location="bottom" menuItems={Nav.DRAWER_ITEMS} />
          </Flex.Box>
        </Flex.Box>
      </Project.Guard>
    </Auth.Guard>
  </>
);
