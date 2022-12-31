// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Space } from "@synnaxlabs/pluto";

import { NavBottom, NavDrawer, NavLeft, NavRight, NavTop } from "./Nav";

import { ClusterProvider } from "@/features/cluster";
import { LayoutMosaic } from "@/features/layout";

import "./MainLayout.css";

/**
 * The center of it all. This is the main layout for the Delta UI. Try to keep this
 * component as simple, presentational, and navigatable as possible.
 */
export const MainLayout = (): JSX.Element => (
  <ClusterProvider>
    <NavTop />
    <Space className="delta-main-fixed--vertical" direction="horizontal" empty>
      <NavLeft />
      <Space
        className="delta-main-content-drawers delta-main-fixed--vertical delta-main-fixed--horizontal"
        empty
      >
        <Space className="delta-main--driven" direction="horizontal" empty>
          <NavDrawer location="left" />
          <div className="delta-main--driven">
            <LayoutMosaic />
          </div>
          <NavDrawer location="right" />
        </Space>
        <NavDrawer location="bottom" />
      </Space>
      <NavRight />
    </Space>
    <NavBottom />
  </ClusterProvider>
);
