// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/layouts/nav/Nav.css";

import { Nav } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { Layout } from "@/layout";
import { Menu } from "@/layouts/nav/Menu";
import { Palette } from "@/palette";

const PALETTE_TRIGGER_CONFIG: Palette.TriggerConfig = {
  command: [["Control", "Shift", "P"]],
  defaultMode: "command",
  search: [["Control", "P"]],
};

const SearchAndCommandPalette = (): ReactElement => (
  <Palette.Palette commandSymbol=">" triggerConfig={PALETTE_TRIGGER_CONFIG} />
);

export const Left = (): ReactElement => (
  <Layout.Nav.Bar location="left" size="8rem">
    <Nav.Bar.Content align="center">
      <SearchAndCommandPalette />
      <Menu location="left" />
    </Nav.Bar.Content>
    <Nav.Bar.End bordered>
      <Menu location="bottom" />
    </Nav.Bar.End>
  </Layout.Nav.Bar>
);
