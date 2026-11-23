// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  channel,
  DataType,
  type lineplot,
  type Synnax,
  type task,
} from "@synnaxlabs/client";
import { uuid } from "@synnaxlabs/x";
import { assertDefined, uniqueName } from "@synnaxlabs/x/testutil";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { expect } from "vitest";

import { Channel } from "@/feature/channel";
import { findLastButton } from "@/platform/modals/testutil";
import { createResource, renderTreeContextMenu } from "@/platform/tree/testutil";

/** Creates a virtual float channel, which no index or data holds. */
export const createVirtualChannel = async (client: Synnax): Promise<channel.Channel> =>
  await client.channels.create({
    name: uniqueName("ch"),
    dataType: DataType.FLOAT32,
    virtual: true,
  });

/** Creates a Modbus read task, on a rack and device of its own, that reads channels. */
export const createReadTask = async (
  client: Synnax,
  channels: channel.Key[],
): Promise<task.Task> => {
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

/** Creates a line plot, in a project of its own, that plots ch. */
export const createPlot = async (
  client: Synnax,
  ch: channel.Key,
  name: string = uniqueName("plot"),
): Promise<lineplot.LinePlot> => {
  const proj = await client.projects.create({ name: uniqueName("proj") });
  return await client.lineplots.create({
    project: proj.key,
    name,
    channels: { y1: [ch] },
  });
};

/** Opens the delete dialog of channels from the channel tree menu, as client. */
export const openDelete = async (
  client: Synnax,
  channels: channel.Channel[],
): Promise<void> => {
  const menu = Channel.TREE_ITEMS.channel.ContextMenu;
  assertDefined(menu);
  await renderTreeContextMenu(menu, {
    client,
    resources: channels.map((ch) =>
      createResource(channel.resourceID(ch.key), ch.name),
    ),
  });
  fireEvent.click(await screen.findByText("Delete"));
};

/** Waits until the dialog's Delete button takes a click. */
export const waitForDelete = async (): Promise<void> =>
  await waitFor(() =>
    expect(findLastButton("Delete").getAttribute("aria-disabled")).toBeNull(),
  );
