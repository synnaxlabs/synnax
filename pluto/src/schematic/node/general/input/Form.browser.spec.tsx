// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, schematic } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { id } from "@synnaxlabs/x";
import { fireEvent, screen, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { InputForm } from "@/schematic/node/general/input/Form";
import { createConfig } from "@/schematic/node/registry";
import { createAsyncSynnaxWrapper, renderForm } from "@/testutil";

const client = createTestClient();

describe("InputForm", () => {
  it("should offer state channels of variable density and not the others", async () => {
    const term = id.create();
    const [text, number] = await client.channels.create([
      { name: `${term}_text`, dataType: DataType.STRING, virtual: true },
      { name: `${term}_number`, dataType: DataType.FLOAT32, virtual: true },
    ]);
    const { findAllByRole } = renderForm(
      <InputForm tab="control" onTabChange={vi.fn()} />,
      createConfig({ variant: "input" }),
      schematic.inputNodeConfigZ,
      await createAsyncSynnaxWrapper({ client }),
    );
    // The state channel comes before the command channel.
    const [state] = await findAllByRole("button", { name: /channel/i });
    fireEvent.click(state);
    fireEvent.change(await screen.findByPlaceholderText(/search/i), {
      target: { value: term },
    });
    const list = await screen.findByRole("listbox");
    await within(list).findByText(text.name);
    expect(within(list).queryByText(number.name)).toBeNull();
  });
});
