// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/layouts/nav/Nav.css";

import { Logo } from "@synnaxlabs/media";
import { Nav } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { Layout } from "@/layout";
import { BOTTOM_DRAWER_ITEMS, LEFT_DRAWER_ITEMS } from "@/layouts/nav/drawerItems";
import { LOGO_LOCATION } from "@/layouts/nav/logo";
import { Menu } from "@/layouts/nav/Menu";

export const Left = (): ReactElement => (
  <Layout.Nav.Bar location="left" size="8rem">
    {LOGO_LOCATION === "left" && (
      <Nav.Bar.Start bordered>
        <Logo />
      </Nav.Bar.Start>
    )}
    <Nav.Bar.Content>
      <Menu location="left" items={LEFT_DRAWER_ITEMS} dividers={[1]} />
    </Nav.Bar.Content>
    <Nav.Bar.End bordered>
      <Menu location="bottom" items={BOTTOM_DRAWER_ITEMS} />
    </Nav.Bar.End>
  </Layout.Nav.Bar>
);
