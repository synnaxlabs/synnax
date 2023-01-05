// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Navbar } from "./Navbar";
import { Navdrawer, useNavDrawer } from "./Navdrawer";
import { NavMenu } from "./NavMenu";
export type { NavbarProps, NavbarContextValue } from "./Navbar";
export type {
  NavDrawerProps,
  NavDrawerContent,
  UseNavDrawerReturn,
  UseNavDrawerProps,
  NavDrawerItem,
} from "./Navdrawer";
export type { NavMenuItem } from "./NavMenu";

export const Nav = {
  Bar: Navbar,
  Drawer: Navdrawer,
  useDrawer: useNavDrawer,
  Menu: NavMenu,
};
