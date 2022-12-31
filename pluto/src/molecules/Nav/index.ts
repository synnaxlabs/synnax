// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Navbar } from "./Navbar";
import { NavDrawer } from "./Navdrawer";
export type { NavbarProps, NavbarContextValue } from "./Navbar";
export type { NavDrawerProps, NavDrawerItem } from "./Navdrawer";

export const Nav = {
  Bar: Navbar,
  Drawer: NavDrawer,
};
