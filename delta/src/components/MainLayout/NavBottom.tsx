// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Nav, Divider } from "@synnaxlabs/pluto";

import { VersionBadge } from "../VersionBadge";

import { NAV_SIZES } from "./constants";

import { ClusterBadge, ConnectionBadge } from "@/features/cluster";

import "./NavBottom.css";

/**
 * NavBottom is the bottom navigation bar for the Delta UI. Try to keep this component
 * presentational.
 */
export const NavBottom = (): JSX.Element => (
  <Nav.Bar location="bottom" size={NAV_SIZES.bottom}>
    <Nav.Bar.End className="delta-main-nav-bottom__end">
      <Divider direction="vertical" />
      <VersionBadge />
      <Divider direction="vertical" />
      <ClusterBadge />
      <Divider direction="vertical" />
      <ConnectionBadge />
    </Nav.Bar.End>
  </Nav.Bar>
);
