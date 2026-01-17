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
import { type ReactElement, useEffect } from "react";
import { useDispatch } from "react-redux";

import { Auth } from "@/auth";
import { ChannelServices } from "@/channel/services";
import { Cluster } from "@/cluster";
import { ClusterServices } from "@/cluster/services";
import { Hardware } from "@/hardware";
import { Layout } from "@/layout";
import { Mosaic } from "@/layouts/Mosaic";
import { Nav } from "@/layouts/nav";
import { Notifications } from "@/layouts/Notifications";
import { useTriggers } from "@/layouts/useTriggers";
import { LinePlotServices } from "@/lineplot/services";
import { Link } from "@/link";
import { LogServices } from "@/log/services";
import { Range } from "@/range";
import { RangeServices } from "@/range/services";
import { SchematicServices } from "@/schematic/services";
import { Status } from "@/status";
import { TableServices } from "@/table/services";
import { Version } from "@/version";
import { Workspace } from "@/workspace";
import { WorkspaceServices } from "@/workspace/services";

const LINK_HANDLERS: Record<string, Link.Handler> = {
  channel: ChannelServices.handleLink,
  ...Hardware.LINK_HANDLERS,
  lineplot: LinePlotServices.handleLink,
  log: LogServices.handleLink,
  range: RangeServices.handleLink,
  schematic: SchematicServices.handleLink,
  table: TableServices.handleLink,
  workspace: WorkspaceServices.handleLink,
};

const SideEffect = (): null => {
  const dispatch = useDispatch();
  useEffect(() => {
    dispatch(Layout.maybeCreateGetStartedTab());
  }, []);
  Access.useLoadPermissions({});
  Version.useLoadTauri();
  Cluster.useSyncClusterKey();
  Hardware.Device.useListenForChanges();
  Range.useListenForChanges();
  Workspace.useSyncLayout();
  Workspace.useCheckCore();
  Status.useListenForChanges();
  Link.useDeep(ClusterServices.handleLink, LINK_HANDLERS);
  useTriggers();
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
    </Auth.Guard>
  </>
);
