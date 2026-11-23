// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, DataType, NotFoundError } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
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

const createIndex = async (): Promise<channel.Channel> =>
  await client.channels.create({
    name: uniqueName("index"),
    dataType: DataType.TIMESTAMP,
    isIndex: true,
  });

const createReadTask = async (ch: channel.Key) => {
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

const openDelete = async (ch: channel.Channel) => {
  const menu = Channel.TREE_ITEMS.channel.ContextMenu;
  assertDefined(menu);
  await renderTreeContextMenu(menu, {
    client,
    resources: [createResource(channel.resourceID(ch.key), ch.name)],
  });
  fireEvent.click(await screen.findByText("Delete"));
};

const deleteEnabled = (): boolean =>
  findLastButton("Delete").getAttribute("aria-disabled") == null;

/** Returns the text of the dialog's footer status. */
const footer = (): string =>
  within(screen.getByRole("dialog")).getByRole("status").textContent;

const taskDeleted = async (key: string): Promise<boolean> =>
  await client.tasks.retrieve({ key }).then(
    () => false,
    (e: unknown) => NotFoundError.matches(e),
  );

describe("impact dialog checks", () => {
  // The task blocks only the data channel's delete, so the plan without that pick no
  // longer lists it.
  it("plans again when the user unchecks a channel whose delete added a blocker", async () => {
    const index = await createIndex();
    const data = await client.channels.create({
      name: uniqueName("data"),
      dataType: DataType.UINT8,
      index: index.key,
    });
    const task = await createReadTask(data.key);
    await openDelete(index);
    fireEvent.click(
      await screen.findByRole("checkbox", { name: `${data.name} Delete too` }),
    );
    fireEvent.click(
      await screen.findByRole("checkbox", { name: `${task.name} Delete too` }),
    );
    await waitFor(() => expect(deleteEnabled()).toBe(true));
    fireEvent.click(screen.getByRole("checkbox", { name: `${data.name} Delete too` }));
    await waitFor(() =>
      expect(deleteEnabled() || footer().includes(UPDATE_FAILED)).toBe(true),
    );
    expect(footer()).not.toContain(UPDATE_FAILED);
    expect(screen.queryByText(task.name)).toBeNull();
  });

  it("shows the new plan when a picked blocker goes before the delete", async () => {
    const ch = await createIndex();
    const task = await createReadTask(ch.key);
    await openDelete(ch);
    fireEvent.click(
      await screen.findByRole("checkbox", { name: `${task.name} Delete too` }),
    );
    await waitFor(() => expect(deleteEnabled()).toBe(true));
    await client.tasks.delete(task.key);
    fireEvent.click(findLastButton("Delete"));
    await waitFor(() =>
      expect(screen.queryByText(CHANGED) != null || footer().includes("Failed")).toBe(
        true,
      ),
    );
    expect(footer()).not.toContain("Failed");
    screen.getByText(CHANGED);
    expect((await client.channels.retrieve(ch.key)).key).toBe(ch.key);
  });

  // A disabled check takes no click, key, or label click, so the pick that the apply
  // carries stays shown until it ends.
  it("locks every check while the delete applies", async () => {
    const ch = await createIndex();
    const task = await createReadTask(ch.key);
    await openDelete(ch);
    const check = await screen.findByRole("checkbox", {
      name: `${task.name} Delete too`,
    });
    fireEvent.click(check);
    await waitFor(() => expect(deleteEnabled()).toBe(true));
    fireEvent.click(findLastButton("Delete"));
    for (const box of within(screen.getByRole("dialog")).getAllByRole("checkbox"))
      expect(box).toBeDisabled();
    expect(check).toBeChecked();
    await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
    expect(await taskDeleted(task.key)).toBe(true);
  });
});
