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
  commitFieldInput,
  createChannelReadOnlyClient,
  createTestChannel,
  deployAndAwaitTask,
  renderTaskFormTab,
  reportTaskStopped,
} from "@/platform/task/testutil";
import { awaitTextEditingElement, commitTextEdit, getIconButton } from "@/testutil";

const client = createTestClient();

// Drafts carry no key; the created row mints its own.
const ZERO_DRAFT: task.New<Modbus.Task.WriteSchemas> = {
  name: "Modbus write task",
  type: Modbus.Task.WRITE_TYPE,
  config: Modbus.Task.WRITE_SCHEMAS.config.parse({}),
};

const createDraft = async (
  client: Synnax,
  config: task.Payload<Modbus.Task.WriteSchemas>["config"],
) => await client.tasks.create({ ...ZERO_DRAFT, config }, Modbus.Task.WRITE_SCHEMAS);

describe("Modbus.Write", () => {
  it("should create command channels and indexes for the built channels on deploy", async () => {
    const dev = await createModbusDevice(client);
    const draft = await createDraft(client, {
      ...Modbus.Task.WRITE_SCHEMAS.config.parse({}),
      device: dev.key,
    });
    const { container } = await renderTaskFormTab(Modbus.Task.Write, {
      client,
      taskKey: draft.key,
    });
    await screen.findByText(dev.name);

    fireEvent.click(getIconButton(container, "add"));
    await screen.findByText("Coil");
    fireEvent.click(getIconButton(container, "add"));
    await waitFor(() => expect(screen.getAllByText("Coil")).toHaveLength(2));

    fireEvent.click(screen.getAllByText("Coil")[1]);
    fireEvent.click(await screen.findByText("Holding register"));
    await screen.findByText("Holding register");

    const created = await deployAndAwaitTask(
      client,
      container,
      draft.key,
      Modbus.Task.WRITE_SCHEMAS,
    );
    expect(created.rack).toBe(dev.rack);
    const config = created.config;
    expect(config.channels).toHaveLength(2);
    const [coil, holding] = config.channels;
    expect(coil.type).toBe("coil");
    expect(coil.address).toBe(0);
    expect(coil.channel).not.toBe(0);
    expect(holding.type).toBe("holding_register");
    expect(holding.address).toBe(1);
    expect(holding.channel).not.toBe(0);

    const coilCmd = await client.channels.retrieve(coil.channel);
    expect(coilCmd.name).toBe(`${dev.name}_coil_output_0_cmd`);
    expect(coilCmd.dataType.toString()).toBe("uint8");
    const coilIndex = await client.channels.retrieve(coilCmd.index);
    expect(coilIndex.name).toBe(`${dev.name}_coil_output_0_cmd_time`);
    expect(coilIndex.isIndex).toBe(true);

    const holdingCmd = await client.channels.retrieve(holding.channel);
    expect(holdingCmd.name).toBe(`${dev.name}_holding_register_output_1_cmd`);
    expect(holdingCmd.dataType.toString()).toBe("uint8");

    const updated = await client.devices.retrieve({
      key: dev.key,
      schemas: Modbus.Device.SCHEMAS,
    });
    expect(updated.properties.write.channels["coil-output-0"]).toBe(coil.channel);
  });

  it("should reuse existing command channels when redeploying", async () => {
    const dev = await createModbusDevice(client);
    const draft = await createDraft(client, {
      ...Modbus.Task.WRITE_SCHEMAS.config.parse({}),
      device: dev.key,
    });
    const first = await renderTaskFormTab(Modbus.Task.Write, {
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
      Modbus.Task.WRITE_SCHEMAS,
    );
    const afterFirst = await client.devices.retrieve({
      key: dev.key,
      schemas: Modbus.Device.SCHEMAS,
    });
    await reportTaskStopped(client, deployed.payload);
    first.unmount();

    const second = await renderTaskFormTab(Modbus.Task.Write, {
      client,
      taskKey: draft.key,
    });
    await screen.findByText("Coil");
    await deployAndAwaitTask(
      client,
      second.container,
      draft.key,
      Modbus.Task.WRITE_SCHEMAS,
    );
    const afterSecond = await client.devices.retrieve({
      key: dev.key,
      schemas: Modbus.Device.SCHEMAS,
    });
    expect(afterSecond.properties.write.channels).toEqual(
      afterFirst.properties.write.channels,
    );
    const matches = await client.channels.retrieve([`${dev.name}_coil_output_0_cmd`]);
    expect(matches).toHaveLength(1);
  });

  it("should rename and remove a channel through the context menu", async () => {
    const dev = await createModbusDevice(client);
    const draft = await createDraft(client, {
      ...Modbus.Task.WRITE_SCHEMAS.config.parse({}),
      device: dev.key,
    });
    const { container } = await renderTaskFormTab(Modbus.Task.Write, {
      client,
      taskKey: draft.key,
    });
    await screen.findByText(dev.name);
    fireEvent.click(getIconButton(container, "add"));
    fireEvent.contextMenu(await screen.findByText("No channel"));
    fireEvent.click(await screen.findByText("Rename"));
    const editable = await awaitTextEditingElement();
    commitTextEdit(editable, "my_cmd_channel");
    await screen.findByText("my_cmd_channel");
    fireEvent.contextMenu(screen.getByText("my_cmd_channel"));
    fireEvent.click(await screen.findByText("Remove"));
    await waitFor(() => expect(screen.queryByText("my_cmd_channel")).toBeNull());
  });

  it("should withhold rename from a subject who cannot update channels", async () => {
    const dev = await createModbusDevice(client);
    const draft = await createDraft(client, {
      ...Modbus.Task.WRITE_SCHEMAS.config.parse({}),
      device: dev.key,
    });
    const { container } = await renderTaskFormTab(Modbus.Task.Write, {
      client,
      taskKey: draft.key,
      as: await createChannelReadOnlyClient(client),
    });
    await screen.findByText(dev.name);
    fireEvent.click(getIconButton(container, "add"));
    fireEvent.contextMenu(await screen.findByText("No channel"));
    // Remove is ungated, so its presence proves the menu resolved before the
    // absence below is read.
    expect(await screen.findByText("Remove")).toBeTruthy();
    expect(screen.queryByText("Rename")).toBeNull();
  });
});

describe("Modbus.Write device map binding", () => {
  const createCoil = (address: number, channel = 0): Modbus.Task.WriteChannel => ({
    ...Modbus.Task.WRITE_CHANNEL_SCHEMAS.coil.parse({ type: "coil" }),
    key: id.create(),
    address,
    channel,
  });

  const createConfig = (device: string, channels: Modbus.Task.WriteChannel[]) => ({
    ...Modbus.Task.WRITE_SCHEMAS.config.parse({}),
    device,
    channels,
  });

  it("should show the channel the device map binds to a row that holds none", async () => {
    const bound = await createTestChannel(client, "mb_cmd");
    const dev = await createModbusDevice(client, {
      properties: { write: { channels: { "coil-output-4": bound.key } } },
    });
    const draft = await createDraft(client, createConfig(dev.key, [createCoil(4)]));
    await renderTaskFormTab(Modbus.Task.Write, { client, taskKey: draft.key });
    await screen.findByText(bound.name);
  });

  it("should drop a row's stale channel when the device map has no entry for its address", async () => {
    const stale = await createTestChannel(client, "mb_cmd");
    const dev = await createModbusDevice(client);
    const draft = await createDraft(
      client,
      createConfig(dev.key, [createCoil(4, stale.key)]),
    );
    await renderTaskFormTab(Modbus.Task.Write, { client, taskKey: draft.key });
    await screen.findByText("No channel");
    expect(screen.queryByText(stale.name)).toBeNull();
  });

  it("should keep a row's channel while no device is selected", async () => {
    const own = await createTestChannel(client, "mb_cmd");
    const draft = await createDraft(client, createConfig("", [createCoil(4, own.key)]));
    await renderTaskFormTab(Modbus.Task.Write, { client, taskKey: draft.key });
    await screen.findByText(own.name);
  });

  it("should rebind a row when its address is edited to one the device map holds", async () => {
    const bound = await createTestChannel(client, "mb_cmd");
    const dev = await createModbusDevice(client, {
      properties: { write: { channels: { "coil-output-7": bound.key } } },
    });
    const draft = await createDraft(client, createConfig(dev.key, [createCoil(4)]));
    await renderTaskFormTab(Modbus.Task.Write, { client, taskKey: draft.key });
    await screen.findByText("No channel");
    commitFieldInput(screen.getByDisplayValue("4"), "7");
    await screen.findByText(bound.name);
  });
});
