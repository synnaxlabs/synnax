// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { Mosaic } from "@/layouts/Mosaic";
import { BOTTOM_DRAWER_ITEM } from "@/layouts/nav/drawerItems";
import { Nav } from "@/nav";

/**
 * Aux is the content of an auxiliary window: just the panel mosaic and the bottom
 * toolbar. Unlike {@link Main}, it has no top or left navigation, drawers, or guards.
 */
export const Aux = (): ReactElement => (
  <Flex.Box y gap="tiny" grow style={{ paddingRight: "1rem", paddingBottom: "1rem" }}>
    <Mosaic />
    <Nav.BottomDrawer item={BOTTOM_DRAWER_ITEM} />
  </Flex.Box>
);
