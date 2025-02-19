// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/layouts/Main.css";

import {
  channel,
  linePlot,
  log,
  ranger,
  schematic,
  table,
  workspace,
} from "@synnaxlabs/client";
import { Drift } from "@synnaxlabs/drift";
import { Align } from "@synnaxlabs/pluto";
import { useEffect } from "react";
import { useDispatch } from "react-redux";

import { Channel } from "@/channel";
import { ChannelServices } from "@/channel/services";
import { Cluster } from "@/cluster";
import { ClusterServices } from "@/cluster/services";
import { Hardware } from "@/hardware";
import { Layout } from "@/layout";
import { Mosaic } from "@/layouts/Mosaic";
import { Nav } from "@/layouts/nav";
import { LinePlotServices } from "@/lineplot/services";
import { Link } from "@/link";
import { LogServices } from "@/log/services";
import { Notifications } from "@/notifications";
import { Permissions } from "@/permissions";
import { RangeServices } from "@/range/services";
import { SchematicServices } from "@/schematic/services";
import { TableServices } from "@/table/services";
import { Version } from "@/version";
import { Workspace } from "@/workspace";
import { WorkspaceServices } from "@/workspace/services";

const NOTIFICATION_ADAPTERS = [
  ...Cluster.NOTIFICATION_ADAPTERS,
  ...Hardware.NOTIFICATION_ADAPTERS,
  ...Version.NOTIFICATION_ADAPTERS,
];

const LINK_HANDLERS: Record<string, Link.Handler> = {
  [channel.ONTOLOGY_TYPE]: ChannelServices.handleLink,
  ...Hardware.LINK_HANDLERS,
  [linePlot.ONTOLOGY_TYPE]: LinePlotServices.handleLink,
  [log.ONTOLOGY_TYPE]: LogServices.handleLink,
  [ranger.ONTOLOGY_TYPE]: RangeServices.handleLink,
  [schematic.ONTOLOGY_TYPE]: SchematicServices.handleLink,
  [table.ONTOLOGY_TYPE]: TableServices.handleLink,
  [workspace.ONTOLOGY_TYPE]: WorkspaceServices.handleLink,
};

const SideEffect = () => {
  const dispatch = useDispatch();
  useEffect(() => {
    dispatch(Layout.maybeCreateGetStartedTab());
  }, []);
  Version.useLoadTauri();
  Cluster.useSyncClusterKey();
  Hardware.Device.useListenForChanges();
  Channel.useListenForCalculationState();
  Workspace.useSyncLayout();
  Link.useDeep(ClusterServices.handleLink, LINK_HANDLERS);
  Layout.useTriggers();
  Permissions.useFetchPermissions();
  Layout.useDropOutside();
  return null;
};

export const MAIN_LAYOUT_TYPE = Drift.MAIN_WINDOW;

/**
 * The center of it all. This is the main layout for the Synnax Console. Try to keep this
 * component as simple, presentational, and navigable as possible.
 */
export const Main = () => (
  <>
    {/* We need to place notifications here so they are in the proper stacking context */}
    <Notifications.Notifications adapters={NOTIFICATION_ADAPTERS} />
    <SideEffect />
    <Nav.Top />
    <Layout.Modals />
    <Align.Space className="console-main-fixed--y" direction="x" empty>
      <Nav.Left />
      <Align.Space
        className="console-main-content-drawers console-main-fixed--y console-main-fixed--x"
        empty
      >
        <Align.Space className="console-main--driven" direction="x" empty>
          <Nav.Drawer location="left" />
          <main className="console-main--driven" style={{ position: "relative" }}>
            <Mosaic />
          </main>
          <Nav.Drawer location="right" />
        </Align.Space>
        <Nav.Drawer location="bottom" />
      </Align.Space>
      <Nav.Right />
    </Align.Space>
    <Nav.Bottom />
  </>
);
