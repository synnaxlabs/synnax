// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { calculation, type channel, DataType, TimeSpan } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { uniqueName } from "@synnaxlabs/x/testutil";
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Channel } from "@/platform/channel";
import { findButton, renderModalOpener } from "@/platform/modals/testutil";

const client = createTestClient();

const openEdit = async (channelKey: channel.Key, name: string) => {
  await renderModalOpener(Channel.useCalculatedModal, [{ channelKey }], {
    client,
    onCaughtError: () => {},
  });
  await waitFor(() =>
    expect(screen.getByPlaceholderText<HTMLInputElement>("Channel name").value).toBe(
      name,
    ),
  );
};

const save = async () => {
  await act(async () => {
    fireEvent.click(findButton("Save"));
  });
  await waitFor(() => expect(screen.queryByPlaceholderText("Channel name")).toBeNull());
};

describe("calculated modal operations", () => {
  it("should clear a windowed operation when None is picked", async () => {
    const name = uniqueName("calc_none");
    const calc = await client.calculations.create({
      name,
      expression: "return 2",
      operations: [{ type: "avg", duration: TimeSpan.seconds(2) }],
    });
    await openEdit(calc.output, name);
    await waitFor(() => expect(screen.getByText("Window")).toBeTruthy());
    fireEvent.click(screen.getByText("None"));
    await waitFor(() => expect(screen.queryByText("Window")).toBeNull());
    await save();
    const [stored] = await client.calculations.retrieve({ output: calc.output });
    expect(stored.operations).toEqual([]);
  });

  it("should save a derivative picked for a calculation with no operations", async () => {
    const name = uniqueName("calc_derivative");
    const calc = await client.calculations.create({ name, expression: "return 2" });
    await openEdit(calc.output, name);
    fireEvent.click(screen.getByText("Derivative"));
    await save();
    const [stored] = await client.calculations.retrieve({ output: calc.output });
    expect(stored.operations).toEqual([{ type: "derivative" }]);
  });

  it("should show an operation that another client sets while the modal is open", async () => {
    const reset = await client.channels.create({
      name: uniqueName("calc_reset"),
      dataType: DataType.BOOLEAN,
      virtual: true,
    });
    const name = uniqueName("calc_reload");
    const calc = await client.calculations.create({ name, expression: "return 2" });
    await openEdit(calc.output, name);
    expect(screen.queryByText("Window")).toBeNull();
    const other = createTestClient();
    await other.calculations.retrieve({ key: calc.key });
    await other.calculations.dispatch(
      calc.key,
      calculation.set({
        path: ["operations"],
        value: [{ type: "max", duration: TimeSpan.seconds(4), reset: reset.key }],
      }),
    );
    await waitFor(() => expect(screen.getByText("Window")).toBeTruthy());
    expect(await screen.findByText(reset.name)).toBeTruthy();
    const next = uniqueName("calc_renamed");
    fireEvent.change(screen.getByPlaceholderText("Channel name"), {
      target: { value: next },
    });
    await save();
    const [stored] = await client.calculations.retrieve({ output: calc.output });
    expect(stored.operations).toEqual([
      { type: "max", duration: TimeSpan.seconds(4), reset: reset.key },
    ]);
    expect((await client.channels.retrieve(calc.output)).name).toBe(next);
  });
});
