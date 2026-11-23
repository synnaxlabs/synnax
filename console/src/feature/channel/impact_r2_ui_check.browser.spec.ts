// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel as channelClient, DataType } from "@synnaxlabs/client";
import {
  createTestClient,
  createTestClientWithPolicy,
} from "@synnaxlabs/client/testutil";
import { uuid } from "@synnaxlabs/x";
import { assertDefined, uniqueName } from "@synnaxlabs/x/testutil";
import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { userEvent } from "vitest/browser";

import { Channel } from "@/feature/channel";
import { findLastButton } from "@/platform/modals/testutil";
import { createResource, renderTreeContextMenu } from "@/platform/tree/testutil";

const client = createTestClient();

const createChannel = async () =>
  await client.channels.create({
    name: uniqueName("ch"),
    dataType: DataType.FLOAT32,
    virtual: true,
  });

const createReadTask = async (ch: channelClient.Key) => {
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
      channels: [{ type: "coil", key: "coil", channel: ch }],
    },
  });
};

const openDelete = async (
  channels: channelClient.Channel[],
  as: typeof client = client,
): Promise<void> => {
  const menu = Channel.TREE_ITEMS.channel.ContextMenu;
  assertDefined(menu);
  await renderTreeContextMenu(menu, {
    client: as,
    resources: channels.map((ch) =>
      createResource(channelClient.resourceID(ch.key), ch.name),
    ),
  });
  fireEvent.click(await screen.findByText("Delete"));
};

const waitForDelete = async () =>
  await waitFor(() =>
    expect(findLastButton("Delete").getAttribute("aria-disabled")).toBeNull(),
  );

describe("impact dialog round 2 ui check", () => {
  it("should keep focus in the dialog when an apply from a check fails", async () => {
    const ch = await createChannel();
    await createReadTask(ch.key);
    const userClient = await createTestClientWithPolicy(client, {
      name: uniqueName("impact"),
      objects: [channelClient.resourceID(ch.key)],
      actions: ["retrieve", "delete"],
    });
    await openDelete([ch], userClient);
    const blocks = await screen.findByRole("region", { name: "Blocks the delete" });
    const hidden = within(blocks).getByRole("checkbox", {
      name: "1 record you cannot see Delete too",
    });
    hidden.focus();
    await userEvent.keyboard(" ");
    await waitForDelete();
    expect(hidden).toHaveFocus();
    await userEvent.keyboard("{Control>}{Enter}{/Control}");
    await screen.findByText("Failed to delete channel");
    expect(screen.getByRole("dialog").contains(document.activeElement)).toBe(true);
  });

  it("should dim the label of a check while the delete applies", async () => {
    const ch = await createChannel();
    const task = await createReadTask(ch.key);
    await openDelete([ch]);
    const check = await screen.findByRole("checkbox", {
      name: `${task.name} Delete too`,
    });
    fireEvent.click(check);
    await waitForDelete();
    const label = screen.getByText("Delete too", { selector: "label" });
    const before = getComputedStyle(label).color;
    fireEvent.click(findLastButton("Delete"));
    expect(check).toBeDisabled();
    expect(getComputedStyle(label).color).not.toBe(before);
  });
});
