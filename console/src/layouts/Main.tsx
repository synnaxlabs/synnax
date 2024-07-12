// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/layouts/Main.css";

import { Drift } from "@synnaxlabs/drift";
import { Align } from "@synnaxlabs/pluto";
import { type ReactElement, useEffect } from "react";
import { useDispatch } from "react-redux";

import { ChannelServices } from "@/channel/services";
import { NavDrawer } from "@/components/nav/Nav";
import { Device } from "@/hardware/device";
import { Layout } from "@/layout";
import { Mosaic } from "@/layouts/Mosaic";
import { NavBottom, NavLeft, NavRight, NavTop } from "@/layouts/Nav";
import { LinePlotServices } from "@/lineplot/services";
import { Link } from "@/link";
import { Notifications } from "@/notifications";
import { RangeServices } from "@/range/services";
import { SchematicServices } from "@/schematic/services";
import { Version } from "@/version";
import { Workspace } from "@/workspace";

const SideEffect = (): null => {
  const dispatch = useDispatch();
  useEffect(() => {
    dispatch(Layout.maybeCreateGetStartedTab());
  }, []);
  Version.useLoadTauri();
  Device.useListenForChanges();
  Workspace.useSyncLayout();
  Link.useDeep({ handlers: HANDLERS });
  return null;
};

export const MAIN_TYPE = Drift.MAIN_WINDOW;

/**
 * The center of it all. This is the main layout for the Synnax Console. Try to keep this
 * component as simple, presentational, and navigatable as possible.
 */
export const Main = (): ReactElement => {
  return (
    <>
      {/* We need to place notifications here so they are in the proper stacking context */}
      <Notifications.Notifications />
      <SideEffect />
      <NavTop />
      <Layout.Modals />
      <Align.Space className="console-main-fixed--y" direction="x" empty>
        <NavLeft />
        <Align.Space
          className="console-main-content-drawers console-main-fixed--y console-main-fixed--x"
          empty
        >
          <Align.Space className="console-main--driven" direction="x" empty>
            <NavDrawer location="left" />
            <main className="console-main--driven" style={{ position: "relative" }}>
              <Mosaic />
            </main>
            <NavDrawer location="right" />
          </Align.Space>
          <NavDrawer location="bottom" />
        </Align.Space>
        <NavRight />
      </Align.Space>
      <NavBottom />
    </>
  );
};

export const HANDLERS: Link.Handler[] = [
  ChannelServices.linkHandler,
  LinePlotServices.linkHandler,
  RangeServices.linkHandler,
  SchematicServices.linkHandler,
  Workspace.linkHandler,
];
