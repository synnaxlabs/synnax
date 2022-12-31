// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Nav, Theming } from "@synnaxlabs/pluto";

import { NAV_SIZES } from "./constants";

import { Logo } from "@/components";
import { ClusterToolbar } from "@/features/cluster";
import { ResourcesToolbar } from "@/features/resources";

import "./NavLeft.css";

/**
 * NavLeft is the left navigation drawer for the Delta UI. Try to keep this component
 * presentational.
 */
export const NavLeft = (): JSX.Element => (
  <Nav.Drawer
    location="left"
    size={NAV_SIZES.side}
    items={[ClusterToolbar, ResourcesToolbar]}
  >
    <Nav.Bar.Start className="delta-main-nav-left__start" bordered>
      <Logo className="delta-main-nav-left__logo" />
    </Nav.Bar.Start>
    <Nav.Bar.End className="delta-main-nav-left__end" bordered>
      <Theming.Switch />
    </Nav.Bar.End>
  </Nav.Drawer>
);
