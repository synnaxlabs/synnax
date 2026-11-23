// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { schematic } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { act, fireEvent, render, waitFor } from "@testing-library/react";
import type { FC, PropsWithChildren } from "react";
import { beforeAll, describe, expect, it, vi } from "vitest";

import { TestNode } from "@/schematic/node/symbol/testutil";
import { createAsyncSynnaxWrapper } from "@/testutil";
import { symbolNode } from "@/vis/symbolNode/aether";

const BUTTON = "button.pluto-symbol";
const SET: schematic.symbol.Command = {
  key: "set",
  name: "Set",
  state: "s",
  value: { type: "ask" },
};

describe("Symbol node ask", () => {
  const client = createTestClient();
  let Wrapper: FC<PropsWithChildren>;

  beforeAll(async () => {
    Wrapper = await createAsyncSynnaxWrapper({
      client,
      additionalRegistry: symbolNode.REGISTRY,
    });
  });

  const createSymbol = async (state: schematic.symbol.State) =>
    await client.schematics.symbols.create({
      name: "asks",
      data: {
        drawing: {
          type: "svg",
          markup:
            '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10">' +
            '<rect width="10" height="10" /></svg>',
        },
        states: [state],
        commands: [SET],
        interactions: [{ type: "fire", key: "click", command: "set" }],
        variant: "actuator",
      },
    });

  const renderAsk = async (state: schematic.symbol.State, value: number) => {
    const symbol = await createSymbol(state);
    const onConfigChange = vi.fn();
    const utils = render(
      <Wrapper>
        <TestNode
          config={{
            spec: { type: "custom", key: symbol.key },
            states: [{ key: "s", source: { type: "local", value } }],
          }}
          onConfigChange={onConfigChange}
        />
      </Wrapper>,
    );
    await waitFor(() => expect(utils.container.querySelector(BUTTON)).not.toBeNull());
    fireEvent.click(utils.container.querySelector(BUTTON)!);
    return { ...utils, onConfigChange };
  };

  const sent = (value: number) => ({
    states: [expect.objectContaining({ source: { type: "local", value } })],
  });

  it("should send the entry that the menu of an enumeration state picks", async () => {
    const e = await client.enumerations.create({
      name: "Mode",
      entries: [
        { key: "off", value: 0, name: "Off" },
        { key: "manual", value: 2, name: "Manual" },
      ],
    });
    const { findByRole, onConfigChange } = await renderAsk(
      {
        type: "enumeration",
        key: "s",
        name: "Mode",
        enumeration: e.key,
        match: { type: "exact" },
        default: 0,
      },
      0,
    );
    fireEvent.click(await findByRole("menuitem", { name: "Manual" }));
    expect(onConfigChange).toHaveBeenCalledWith(sent(2));
  });

  it("should send a typed number only on Send", async () => {
    const { findByRole, getByRole, onConfigChange } = await renderAsk(
      { type: "number", key: "s", name: "Level", bounds: { lower: 0, upper: 100 } },
      10,
    );
    const field = await findByRole("textbox");
    expect(field).toHaveProperty("value", "10");
    await act(async () => {
      field.focus();
      fireEvent.change(field, { target: { value: "500" } });
      fireEvent.keyDown(field, { key: "Enter", code: "Enter" });
    });
    expect(onConfigChange).not.toHaveBeenCalled();
    fireEvent.click(getByRole("button", { name: "Send" }));
    await waitFor(() => expect(onConfigChange).toHaveBeenCalledWith(sent(100)));
  });
});
