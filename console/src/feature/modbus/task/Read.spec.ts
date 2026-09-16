// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax, type task } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { id } from "@synnaxlabs/x";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Modbus } from "@/feature/modbus";
import { createModbusDevice } from "@/feature/modbus/testutil";
import {
  createTestChannel,
  deployAndAwaitTask,
  renderTaskFormTab,
  reportTaskStopped,
} from "@/platform/task/testutil";
import { getIconButton } from "@/testutil";

const client = createTestClient();

// Drafts carry no key; the created row mints its own.
const ZERO_DRAFT: task.New<Modbus.Task.ReadSchemas> = {
  name: "Modbus read task",
  type: Modbus.Task.READ_TYPE,
  config: Modbus.Task.READ_SCHEMAS.config.parse({}),
};

const createDraft = async (
  client: Synnax,
  config: task.Payload<Modbus.Task.ReadSchemas>["config"],
) => await client.tasks.create({ ...ZERO_DRAFT, config }, Modbus.Task.READ_SCHEMAS);

describe("Modbus.Read", () => {
  it("should build channels in the form and create them on the Core on deploy", async () => {
    const dev = await createModbusDevice(client);
    const draft = await createDraft(client, {
      ...Modbus.Task.READ_SCHEMAS.config.parse({}),
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

    const created = await deployAndAwaitTask(
      client,
      container,
      draft.key,
      Modbus.Task.READ_SCHEMAS,
    );
    expect(created.rack).toBe(dev.rack);
    const config = created.config;
    expect(config.device).toBe(dev.key);
    expect(config.channels).toHaveLength(2);
    const [coil, register] = config.channels;
    expect(coil.type).toBe("coil");
    expect(coil.address).toBe(0);
    expect(coil.channel).not.toBe(0);
    expect(register.type).toBe("input_register");
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
      ...Modbus.Task.READ_SCHEMAS.config.parse({}),
      device: dev.key,
    });
    const first = await renderTaskFormTab(Modbus.Task.Read, {
      client,
      taskKey: draft.key,
    });
    await screen.findByText(dev.name);
    fireEvent.click(getIconButton(first.container, "add"));
    await screen.findByText("Coil");
    const deployed = await deployAndAwaitTask(
      client,
      first.container,
      draft.key,
      Modbus.Task.READ_SCHEMAS,
    );
    const afterFirst = await client.devices.retrieve({
      key: dev.key,
      schemas: Modbus.Device.SCHEMAS,
    });
    await reportTaskStopped(client, deployed.payload);
    first.unmount();

    const second = await renderTaskFormTab(Modbus.Task.Read, {
      client,
      taskKey: draft.key,
    });
    await screen.findByText("Coil");
    await deployAndAwaitTask(
      client,
      second.container,
      draft.key,
      Modbus.Task.READ_SCHEMAS,
    );
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

describe("Modbus.Read device map binding", () => {
  const createCoil = (address: number, channel = 0): Modbus.Task.ReadChannel => ({
    ...Modbus.Task.READ_CHANNEL_SCHEMAS.coil.parse({ type: "coil" }),
    key: id.create(),
    address,
    channel,
  });

  const createConfig = (device: string, channels: Modbus.Task.ReadChannel[]) => ({
    ...Modbus.Task.READ_SCHEMAS.config.parse({}),
    device,
    channels,
  });

  it("should show the channel the device map binds to a row that holds none", async () => {
    const bound = await createTestChannel(client, "mb");
    const dev = await createModbusDevice(client, {
      properties: { read: { index: 0, channels: { "coil-input-4": bound.key } } },
    });
    const draft = await createDraft(client, createConfig(dev.key, [createCoil(4)]));
    await renderTaskFormTab(Modbus.Task.Read, { client, taskKey: draft.key });
    await screen.findByText(bound.name);
  });

  it("should drop a row's stale channel when the device map has no entry for its address", async () => {
    const stale = await createTestChannel(client, "mb");
    const dev = await createModbusDevice(client);
    const draft = await createDraft(
      client,
      createConfig(dev.key, [createCoil(4, stale.key)]),
    );
    await renderTaskFormTab(Modbus.Task.Read, { client, taskKey: draft.key });
    await screen.findByText("No channel");
    expect(screen.queryByText(stale.name)).toBeNull();
  });

  it("should keep a row's channel while no device is selected", async () => {
    const own = await createTestChannel(client, "mb");
    const draft = await createDraft(client, createConfig("", [createCoil(4, own.key)]));
    await renderTaskFormTab(Modbus.Task.Read, { client, taskKey: draft.key });
    await screen.findByText(own.name);
  });
});
