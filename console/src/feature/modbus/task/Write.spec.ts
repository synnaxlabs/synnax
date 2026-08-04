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
import { awaitTextEditingElement, commitTextEdit, getIconButton } from "@/testutil";

const client = createTestClient();

// Draft creates mint their own key; the zero payload's empty key must not be sent.
const { key: _key, ...ZERO_DRAFT } = Modbus.Task.ZERO_WRITE_PAYLOAD;

const createDraft = async (
  client: Synnax,
  config: task.Payload<Modbus.Task.WriteSchemas>["config"],
) => await client.tasks.create({ ...ZERO_DRAFT, config }, Modbus.Task.WRITE_SCHEMAS);

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
  return await client.tasks.retrieve({ key, schemas: Modbus.Task.WRITE_SCHEMAS });
};

describe("Modbus.Write", () => {
  it("should create command channels and indexes for the built channels on deploy", async () => {
    const dev = await createModbusDevice(client);
    const draft = await createDraft(client, {
      ...Modbus.Task.ZERO_WRITE_PAYLOAD.config,
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
    fireEvent.click(await screen.findByText("Holding Register"));
    await screen.findByText("Holding Register");

    const created = await deployAndAwaitTask(client, container, draft.key);
    expect(created.rack).toBe(dev.rack);
    const config = created.config;
    expect(config.channels).toHaveLength(2);
    const [coil, holding] = config.channels;
    expect(coil.type).toBe("coil_output");
    expect(coil.address).toBe(0);
    expect(coil.channel).not.toBe(0);
    expect(holding.type).toBe("holding_register_output");
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
      ...Modbus.Task.ZERO_WRITE_PAYLOAD.config,
      device: dev.key,
    });
    const first = await renderTaskFormTab(Modbus.Task.Write, {
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

    const second = await renderTaskFormTab(Modbus.Task.Write, {
      client,
      taskKey: draft.key,
    });
    await screen.findByText("Coil");
    await deployAndAwaitTask(client, second.container, draft.key);
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
      ...Modbus.Task.ZERO_WRITE_PAYLOAD.config,
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
});
