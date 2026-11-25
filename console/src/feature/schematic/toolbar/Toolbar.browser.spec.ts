// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import {
  ACTIVE_BINDING,
  createValveNode,
  pickSymbol,
  preloadTab,
  renderSelected,
  retrieveConfigs,
} from "@/feature/schematic/testutil";
import { findLastButton } from "@/platform/modals/testutil";

const LOSS = "Valve loses the Active binding.";

preloadTab();

describe("Toolbar symbol change", () => {
  it("should ask before it removes the bindings the new symbol lacks", async () => {
    const { key } = await renderSelected({
      n1: createValveNode({ states: [ACTIVE_BINDING] }),
    });
    await pickSymbol("Cap");
    await screen.findByText(LOSS);
    fireEvent.click(findLastButton("Change"));
    await waitFor(async () =>
      expect((await retrieveConfigs(key)).n1).toMatchObject({
        spec: { type: "builtin", name: "cap" },
        states: [],
      }),
    );
  });

  it("should ask before a change to an item that takes no bindings", async () => {
    const { key } = await renderSelected({
      n1: createValveNode({ states: [ACTIVE_BINDING] }),
    });
    await pickSymbol("Value");
    await screen.findByText(LOSS);
    fireEvent.click(findLastButton("Change"));
    await waitFor(async () =>
      expect((await retrieveConfigs(key)).n1.variant).toBe("value"),
    );
  });

  it("should keep the symbol and bindings when the confirm is canceled", async () => {
    const { key } = await renderSelected({
      n1: createValveNode({ states: [ACTIVE_BINDING] }),
    });
    await pickSymbol("Cap");
    await screen.findByText(LOSS);
    fireEvent.click(findLastButton("Cancel"));
    await waitFor(() => expect(screen.queryByText(LOSS)).toBeNull());
    expect((await retrieveConfigs(key)).n1).toMatchObject({
      spec: { type: "builtin", name: "valve" },
      states: [ACTIVE_BINDING],
    });
  });
});
