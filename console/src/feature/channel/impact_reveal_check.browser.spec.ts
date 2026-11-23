// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel as channelClient, DataType, query } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { uuid } from "@synnaxlabs/x";
import { uniqueName } from "@synnaxlabs/x/testutil";
import { act, fireEvent, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Channel } from "@/feature/channel";
import { Tree } from "@/platform/tree";
import {
  createHostItems,
  HOST_ID,
  openTreeRowContextMenu,
  renderLiveTree,
} from "@/platform/tree/testutil";

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

const hostChannels = (...channels: { key: channelClient.Key }[]) =>
  createHostItems((c) => [
    Tree.createSource(
      c.channels,
      query.keysRequest(channels.map((ch) => ch.key)),
      (ch) => Tree.toResource("channel")(ch.payload),
    ),
  ]);

describe("impact dialog reveal check", () => {
  it("should keep focus out of the tree while the dialog waits for its plan", async () => {
    const ch = await createChannel();
    await createReadTask(ch.key);
    await renderLiveTree({
      client,
      root: HOST_ID,
      items: { ...Channel.TREE_ITEMS, ...hostChannels(ch) },
    });
    const tree = screen.getByRole("tree");
    // A right-click focuses the tree, so its menu gives the focus back to it.
    act(() => tree.focus());
    await openTreeRowContextMenu(ch.name);
    fireEvent.click(await screen.findByText("Delete"));
    const dialog = screen.getByRole("dialog", { hidden: true });
    const focused: string[] = [];
    for (let frame = 0; frame < 60; frame++) {
      if (tree.contains(document.activeElement)) focused.push(`frame ${frame}`);
      if (frame > 0 && getComputedStyle(dialog).visibility === "visible") break;
      await new Promise((resolve) => requestAnimationFrame(resolve));
    }
    expect(focused).toEqual([]);
  });
});
