// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, DataType, task as taskClient } from "@synnaxlabs/client";
import {
  createTestClient,
  createTestClientWithPolicy,
} from "@synnaxlabs/client/testutil";
import { uuid } from "@synnaxlabs/x";
import { assertDefined, uniqueName } from "@synnaxlabs/x/testutil";
import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Channel } from "@/feature/channel";
import { findLastButton } from "@/platform/modals/testutil";
import { createResource, renderTreeContextMenu } from "@/platform/tree/testutil";

const client = createTestClient();

const CHANGED = "These records changed. Check them again.";
const UPDATE_FAILED = "Failed to update the records";
const HIDDEN_CHECK = "1 record you cannot see Delete too";

const createIndex = async (): Promise<channel.Channel> =>
  await client.channels.create({
    name: uniqueName("index"),
    dataType: DataType.TIMESTAMP,
    isIndex: true,
  });

const createData = async (index: channel.Channel): Promise<channel.Channel> =>
  await client.channels.create({
    name: uniqueName("data"),
    dataType: DataType.UINT8,
    index: index.key,
  });

const createReadTask = async (channels: channel.Key[]) => {
  const rack = await client.racks.create({ name: uniqueName("rack") });
  const device = await client.devices.create({
    key: uuid.create(),
    rack: rack.key,
    location: "127.0.0.1:502",
    name: uniqueName("plc"),
    make: "modbus",
    model: "modbus",
    properties: {},
  });
  return await rack.createTask({
    name: uniqueName("read"),
    type: "modbus_read",
    config: {
      device: device.key,
      channels: channels.map((ch, i) => ({
        type: "coil",
        key: `coil-${i}`,
        channel: ch,
      })),
    },
  });
};

const openDelete = async (ch: channel.Channel, as: typeof client = client) => {
  const menu = Channel.TREE_ITEMS.channel.ContextMenu;
  assertDefined(menu);
  await renderTreeContextMenu(menu, {
    client: as,
    resources: [createResource(channel.resourceID(ch.key), ch.name)],
  });
  fireEvent.click(await screen.findByText("Delete"));
};

const deleteEnabled = (): boolean =>
  findLastButton("Delete").getAttribute("aria-disabled") == null;

/** Returns the text of the dialog's footer status. */
const footer = (): string =>
  within(screen.getByRole("dialog")).getByRole("status").textContent;

/** Waits until the plan in flight lands or fails. */
const settle = async () =>
  await waitFor(
    () => expect(deleteEnabled() || footer().includes(UPDATE_FAILED)).toBe(true),
    { timeout: 5000 },
  );

const pickOf = (name: string): HTMLElement =>
  screen.getByRole("checkbox", { name: `${name} Delete too` });

const findPickOf = async (name: string): Promise<HTMLElement> =>
  await screen.findByRole("checkbox", { name: `${name} Delete too` });

describe("impact dialog round 2 checks", () => {
  // The task first shows in the plan after the refused token, which no pick asked for.
  it("drops the pick of a task from a changed plan with its channel", async () => {
    const index = await createIndex();
    const data = await createData(index);
    await openDelete(index);
    fireEvent.click(await findPickOf(data.name));
    await settle();
    const task = await createReadTask([data.key]);
    fireEvent.click(findLastButton("Delete"));
    await screen.findByText(CHANGED);
    await findPickOf(task.name);
    await settle();
    fireEvent.click(pickOf(task.name));
    await settle();
    fireEvent.click(pickOf(data.name));
    await settle();
    expect(footer()).not.toContain(UPDATE_FAILED);
    expect(screen.queryByText(task.name)).toBeNull();
  });

  // The task reads both data channels, so it still blocks through the second one.
  it("keeps the pick of a task that another picked channel still brings", async () => {
    const index = await createIndex();
    const first = await createData(index);
    const second = await createData(index);
    const task = await createReadTask([first.key, second.key]);
    await openDelete(index);
    fireEvent.click(await findPickOf(first.name));
    await findPickOf(task.name);
    await settle();
    fireEvent.click(pickOf(second.name));
    await settle();
    fireEvent.click(pickOf(task.name));
    await settle();
    fireEvent.click(pickOf(first.name));
    await settle();
    expect(footer()).not.toContain(UPDATE_FAILED);
    expect(pickOf(task.name)).toBeChecked();
  });

  // The user cannot see the data channel, so only the hidden pick deletes it and brings
  // the task that reads it.
  it("drops the pick of a task that the hidden pick brings with it", async () => {
    const index = await createIndex();
    const data = await createData(index);
    const task = await createReadTask([data.key]);
    const userClient = await createTestClientWithPolicy(client, {
      name: uniqueName("impact"),
      objects: [channel.resourceID(index.key), taskClient.resourceID(task.key)],
      actions: ["retrieve", "delete"],
    });
    await openDelete(index, userClient);
    const blocks = await screen.findByRole("region", { name: "Blocks the delete" });
    fireEvent.click(within(blocks).getByRole("checkbox", { name: HIDDEN_CHECK }));
    await findPickOf(task.name);
    await settle();
    fireEvent.click(pickOf(task.name));
    await settle();
    fireEvent.click(screen.getByRole("checkbox", { name: HIDDEN_CHECK }));
    await settle();
    expect(footer()).not.toContain(UPDATE_FAILED);
    expect(screen.queryByText(task.name)).toBeNull();
  });

  // Both picks go out before the first plan lands, as two clicks inside one round trip
  // to a remote Core do.
  it("drops the pick of a task that the first of two quick picks brings", async () => {
    const index = await createIndex();
    const first = await createData(index);
    const second = await createData(index);
    const task = await createReadTask([first.key]);
    await openDelete(index);
    await findPickOf(first.name);
    await settle();
    fireEvent.click(pickOf(first.name));
    fireEvent.click(pickOf(second.name));
    await findPickOf(task.name);
    await settle();
    fireEvent.click(pickOf(task.name));
    await settle();
    fireEvent.click(pickOf(first.name));
    await settle();
    expect(footer()).not.toContain(UPDATE_FAILED);
    expect(screen.queryByText(task.name)).toBeNull();
  });
});
