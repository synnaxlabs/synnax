// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, DataType, TimeSpan } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { uniqueName } from "@synnaxlabs/x/testutil";
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Channel } from "@/platform/channel";
import { findButton, renderModalOpener } from "@/platform/modals/testutil";

const client = createTestClient();

const createCalculated = async (reset?: channel.Key) => {
  const name = uniqueName("reset_calc");
  const calc = await client.calculations.create({
    name,
    expression: "return 2",
    operations: [{ type: "avg", duration: TimeSpan.seconds(5), reset }],
  });
  return { name, calc };
};

const createReset = async (): Promise<channel.Channel> =>
  await client.channels.create({
    name: uniqueName("reset_bool"),
    dataType: DataType.BOOLEAN,
    virtual: true,
  });

const openModal = async (channelKey: channel.Key, name: string): Promise<void> => {
  await renderModalOpener(Channel.useCalculatedModal, [{ channelKey }], {
    client,
    onCaughtError: () => {},
  });
  await waitFor(() =>
    expect(screen.getByPlaceholderText<HTMLInputElement>("Channel name").value).toBe(
      name,
    ),
  );
  await waitFor(() => expect(screen.getByText("Reset channel")).toBeTruthy());
};

const save = async (): Promise<void> => {
  await act(async () => {
    fireEvent.click(findButton("Save"));
  });
};

describe("calculated modal reset channel check", () => {
  it("should save a picked reset channel as the window's reset", async () => {
    const reset = await createReset();
    const { name, calc } = await createCalculated();
    await openModal(calc.output, name);
    fireEvent.click(screen.getByText("Select channel"));
    fireEvent.change(await screen.findByPlaceholderText("Search channels..."), {
      target: { value: reset.name },
    });
    fireEvent.click(await screen.findByText(reset.name));
    await save();
    await waitFor(async () => {
      const [stored] = (await client.calculations.retrieve({ key: calc.key }))
        .operations;
      expect(stored).toEqual({
        type: "avg",
        duration: TimeSpan.seconds(5),
        reset: reset.key,
      });
    });
  });

  it("should save a cleared reset channel as no reset", async () => {
    const reset = await createReset();
    const { name, calc } = await createCalculated(reset.key);
    await openModal(calc.output, name);
    const trigger = (await screen.findByText(reset.name)).closest("button");
    expect(trigger).not.toBeNull();
    fireEvent.keyDown(trigger as HTMLButtonElement, { key: "Backspace" });
    await waitFor(() => expect(screen.getByText("Select channel")).toBeTruthy());
    await save();
    await waitFor(async () => {
      const [stored] = (await client.calculations.retrieve({ key: calc.key }))
        .operations;
      expect(stored).toEqual({ type: "avg", duration: TimeSpan.seconds(5) });
    });
  });
});
