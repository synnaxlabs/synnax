// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Space } from "@synnaxlabs/pluto";

import { ClusterProvider } from "@/features/cluster";

import { NavBottom, NavDrawer, NavLeft, NavRight, NavTop } from "./Nav";

import { LayoutMosaic } from "@/features/layout";
import { VisCanvas } from "@/features/vis";

import "./MainLayout.css";

/**
 * The center of it all. This is the main layout for the Delta UI. Try to keep this
 * component as simple, presentational, and navigatable as possible.
 */
export const MainLayout = (): JSX.Element => (
  <ClusterProvider>
    <NavTop />
    <Space className="delta-main-fixed--y" direction="x" empty>
      <NavLeft />
      <Space
        className="delta-main-content-drawers delta-main-fixed--y delta-main-fixed--x"
        empty
      >
        <Space className="delta-main--driven" direction="x" empty>
          <NavDrawer location="left" className="delta-main-nav-drawer--y" />
          <div className="delta-main--driven" style={{ position: "relative" }}>
            <VisCanvas>
              <LayoutMosaic />
            </VisCanvas>
          </div>
          <NavDrawer location="right" className="delta-main-nav-drawer--y" />
        </Space>
        <NavDrawer location="bottom" />
      </Space>
      <NavRight />
    </Space>
    <NavBottom />
  </ClusterProvider>
);
