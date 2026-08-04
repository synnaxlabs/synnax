// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax, task } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Modbus } from "@/feature/modbus";
import { createModbusDevice } from "@/feature/modbus/testutil";
import { awaitCommand, clickDeploy, renderTaskFormTab } from "@/platform/task/testutil";
import { getIconButton } from "@/testutil";

const client = createTestClient();

// Draft creates mint their own key; the zero payload's empty key must not be sent.
const { key: _key, ...ZERO_DRAFT } = Modbus.Task.ZERO_READ_PAYLOAD;

const createDraft = async (
  client: Synnax,
  config: task.Payload<Modbus.Task.ReadSchemas>["config"],
) => await client.tasks.create({ ...ZERO_DRAFT, config }, Modbus.Task.READ_SCHEMAS);

const deployAndAwaitTask = async (
  client: Synnax,
  container: ParentNode,
  key: task.Key,
) => {
  const streamer = await client.openStreamer(task.COMMAND_CHANNEL_NAME);
  try {
    await clickDeploy(container);
    await awaitCommand(streamer, key);
  } finally {
    streamer.close();
  }
  return await client.tasks.retrieve({ key, schemas: Modbus.Task.READ_SCHEMAS });
};

describe("Modbus.Read", () => {
  it("should build channels in the form and create them on the cluster on deploy", async () => {
    const dev = await createModbusDevice(client);
    const draft = await createDraft(client, {
      ...Modbus.Task.ZERO_READ_PAYLOAD.config,
      device: dev.key,
    });
    const { container } = await renderTaskFormTab(Modbus.Task.Read, {
      client,
      taskKey: draft.key,
    });
    await screen.findByText(dev.name);

    fireEvent.click(getIconButton(container, "add"));
    await screen.findByText("Coil");
    fireEvent.click(getIconButton(container, "add"));
    await waitFor(() => expect(screen.getAllByText("Coil")).toHaveLength(2));

    fireEvent.click(screen.getAllByText("Coil")[1]);
    fireEvent.click(await screen.findByText("Register"));
    await screen.findByText("Register");

    const created = await deployAndAwaitTask(client, container, draft.key);
    expect(created.rack).toBe(dev.rack);
    const config = created.config;
    expect(config.device).toBe(dev.key);
    expect(config.channels).toHaveLength(2);
    const [coil, register] = config.channels;
    expect(coil.type).toBe("coil_input");
    expect(coil.address).toBe(0);
    expect(coil.channel).not.toBe(0);
    expect(register.type).toBe("register_input");
    expect(register.address).toBe(1);
    expect(register.channel).not.toBe(0);

    const updated = await client.devices.retrieve({
      key: dev.key,
      schemas: Modbus.Device.SCHEMAS,
    });
    expect(updated.properties.read.index).not.toBe(0);
    expect(updated.properties.read.channels["coil-input-0"]).toBe(coil.channel);
    const index = await client.channels.retrieve(updated.properties.read.index);
    expect(index.name).toBe(`${dev.name}_time`);
    expect(index.isIndex).toBe(true);

    const coilCh = await client.channels.retrieve(coil.channel);
    expect(coilCh.name).toBe(`${dev.name}_coil_input_0`);
    expect(coilCh.index).toBe(index.key);
    const registerCh = await client.channels.retrieve(register.channel);
    expect(registerCh.name.startsWith(`${dev.name}_register_input_1`)).toBe(true);
    expect(registerCh.dataType.toString()).toBe("uint8");
  });

  it("should reuse the existing index and channels when redeploying", async () => {
    const dev = await createModbusDevice(client);
    const draft = await createDraft(client, {
      ...Modbus.Task.ZERO_READ_PAYLOAD.config,
      device: dev.key,
    });
    const first = await renderTaskFormTab(Modbus.Task.Read, {
      client,
      taskKey: draft.key,
    });
    await screen.findByText(dev.name);
    fireEvent.click(getIconButton(first.container, "add"));
    await screen.findByText("Coil");
    await deployAndAwaitTask(client, first.container, draft.key);
    const afterFirst = await client.devices.retrieve({
      key: dev.key,
      schemas: Modbus.Device.SCHEMAS,
    });
    first.unmount();

    const second = await renderTaskFormTab(Modbus.Task.Read, {
      client,
      taskKey: draft.key,
    });
    await screen.findByText("Coil");
    await deployAndAwaitTask(client, second.container, draft.key);
    const afterSecond = await client.devices.retrieve({
      key: dev.key,
      schemas: Modbus.Device.SCHEMAS,
    });
    expect(afterSecond.properties.read.index).toBe(afterFirst.properties.read.index);
    expect(afterSecond.properties.read.channels).toEqual(
      afterFirst.properties.read.channels,
    );
    const matches = await client.channels.retrieve([`${dev.name}_coil_input_0`]);
    expect(matches).toHaveLength(1);
  });
});
