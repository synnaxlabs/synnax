// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { schematic } from "@synnaxlabs/client";
import { Menu } from "@synnaxlabs/lyra/menu";
import { uniqueName } from "@synnaxlabs/x/testutil";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import type { ReactElement } from "react";
import { describe, expect, it } from "vitest";

import {
  client,
  createSymbolPayload,
  createValveNode,
  preloadTab,
  renderSchematic,
} from "@/feature/schematic/testutil";
import { SymbolMenuItems } from "@/feature/schematic/toolbar/selection";
import { Session } from "@/session";
import { documentIn } from "@/session/window/testutil";

preloadTab();

const Items = (): ReactElement => (
  <Menu.Menu>
    <SymbolMenuItems />
  </Menu.Menu>
);

const renderItems = async (config: schematic.NodeConfig) =>
  await renderSchematic(Items, {
    schematic: { nodes: [{ key: "n1", position: { x: 0, y: 0 }, config }] },
    sessionState: {
      editable: true,
      selected: ["n1"],
      toolbar: {
        selectedTab: "properties",
        selectedSymbolGroup: "general",
        changing: false,
      },
    },
  });

describe("SymbolMenuItems", () => {
  it("should start a change of the selected symbol", async () => {
    const { key, store } = await renderItems(createValveNode());
    fireEvent.click(await screen.findByText("Change symbol"));
    await waitFor(() =>
      expect(
        documentIn(store.getState()[Session.Schematic.SLICE_NAME], key)?.toolbar,
      ).toMatchObject({ selectedTab: "symbols", changing: true }),
    );
  });

  it("should open the editor of the custom symbol that the selection draws", async () => {
    const symbol = await client.schematics.symbols.create(
      createSymbolPayload(uniqueName("custom")),
    );
    await renderItems(createValveNode({ spec: { type: "custom", key: symbol.key } }));
    fireEvent.click(await screen.findByText("Edit symbol"));
    expect(await screen.findByDisplayValue(symbol.name)).toBeDefined();
  });

  it("should offer no edit for a built-in symbol", async () => {
    await renderItems(createValveNode());
    await screen.findByText("Change symbol");
    expect(screen.queryByText("Edit symbol")).toBeNull();
  });
});
