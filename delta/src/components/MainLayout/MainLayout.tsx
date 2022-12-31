// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Space } from "@synnaxlabs/pluto";

import { NavBottom } from "./NavBottom";
import { NavLeft } from "./NavLeft";
import { NavRight } from "./NavRight";
import { NavTop } from "./NavTop";

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
    <Space direction="horizontal" size="large" className="delta-main__middle" empty>
      <NavLeft />
      <div className="delta-main__content">
        <LayoutMosaic />
      </div>
      <NavRight />
    </Space>
    <NavBottom />
  </ClusterProvider>
);
