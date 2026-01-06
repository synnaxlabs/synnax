// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/modals/BottomNavBar.css";

import { Nav } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { CSS } from "@/css";

export interface BottomNavBarProps extends Nav.BarProps {}

export const BottomNavBar = ({
  className,
  ...rest
}: BottomNavBarProps): ReactElement => (
  <Nav.Bar
    location="bottom"
    size="8rem"
    className={CSS(CSS.B("bottom-nav-bar"), className)}
    bordered
    {...rest}
  />
);
