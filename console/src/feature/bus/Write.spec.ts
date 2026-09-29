// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type bus, can, tcp } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { DataType } from "@synnaxlabs/x";
import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import {
  createBusDevice,
  createBusLibrary,
  createBusTask,
  createMessage,
  findFieldRow,
  renderBusTask,
} from "@/feature/bus/testutil";
import { CAN } from "@/feature/can";
import { TCP } from "@/feature/tcp";
import { deployAndAwaitTask } from "@/platform/task/testutil";
import { uniqueName } from "@/testutil";

const client = createTestClient();

const setup = async () => {
  const lib = await createBusLibrary(client);
  const dev = await createBusDevice(
    client,
    TCP.Device.MAKE,
    tcp.propertiesZ.parse({ host: "localhost", port: 5025 }),
  );
  const draft = await createBusTask(
    client,
    TCP.Task.WRITE_TYPE,
    TCP.Task.WRITE_SCHEMAS.config.parse({
      library: lib.library.key,
      device: dev.key,
      messages: [createMessage(lib.status)],
    }),
  );
  return { ...lib, dev, draft };
};

const createCommandChannel = async () => {
  const index = await client.channels.create({
    name: uniqueName("cmd_time"),
    dataType: DataType.TIMESTAMP,
    isIndex: true,
  });
  return await client.channels.create({
    name: uniqueName("cmd"),
    dataType: DataType.FLOAT64,
    index: index.key,
  });
};

/** Expects the command channels of a deployed message to hold the names on index. */
const expectCommands = async (m: bus.WriteMessage, index: string, fields: string[]) => {
  const channels = await client.channels.retrieve(m.fields.map((f) => f.channel));
  expect(channels.map((c) => c.name).sort()).toEqual(fields);
  const [first] = channels;
  expect((await client.channels.retrieve(first.index)).name).toBe(index);
  channels.forEach((c) => expect(c.index).toBe(first.index));
};

describe("bus write task", () => {
  it("should create a command channel for each unmapped field on deploy", async () => {
    const { dev, draft } = await setup();
    const { container } = await renderBusTask(TCP.Task.Write, client, draft.key);
    await findFieldRow("Volts");
    const deployed = await deployAndAwaitTask(
      client,
      container,
      draft.key,
      TCP.Task.WRITE_SCHEMAS,
    );
    const channels = await client.channels.retrieve(
      deployed.config.messages[0].fields.map((f) => f.channel),
    );
    expect(channels.map((c) => c.name).sort()).toEqual([
      `${dev.name}_Status_Amps_cmd`,
      `${dev.name}_Status_Volts_cmd`,
    ]);
    const index = await client.channels.retrieve(channels[0].index);
    expect(index.name).toBe(`${dev.name}_Status_cmd_time`);
    channels.forEach((c) => expect(c.index).toBe(index.key));
  });

  it("should put the command channels of each message on its own index", async () => {
    const { library, engine, brake } = await createBusLibrary(client);
    const dev = await createBusDevice(
      client,
      CAN.Device.MAKE,
      can.propertiesZ.parse({}),
    );
    const draft = await createBusTask(
      client,
      CAN.Task.WRITE_TYPE,
      CAN.Task.WRITE_SCHEMAS.config.parse({
        library: library.key,
        device: dev.key,
        messages: [createMessage(engine), createMessage(brake)],
      }),
    );
    const { container } = await renderBusTask(CAN.Task.Write, client, draft.key);
    await findFieldRow("Rpm");
    const deployed = await deployAndAwaitTask(
      client,
      container,
      draft.key,
      CAN.Task.WRITE_SCHEMAS,
    );
    const [engineMsg, brakeMsg] = deployed.config.messages;
    await expectCommands(engineMsg, `${dev.name}_Engine_cmd_time`, [
      `${dev.name}_Engine_Rpm_cmd`,
      `${dev.name}_Engine_Temp_cmd`,
    ]);
    await expectCommands(brakeMsg, `${dev.name}_Brake_cmd_time`, [
      `${dev.name}_Brake_Pressure_cmd`,
    ]);
  });

  it("should send a field to the command channel the user maps it to", async () => {
    const { status, draft } = await setup();
    const cmd = await createCommandChannel();
    const { container } = await renderBusTask(TCP.Task.Write, client, draft.key);
    const row = await findFieldRow("Volts");
    fireEvent.click(within(row).getByRole("button", { name: "Channel" }));
    fireEvent.change(await screen.findByPlaceholderText("Search channels..."), {
      target: { value: cmd.name },
    });
    fireEvent.click(await screen.findByText(cmd.name));
    await waitFor(async () => {
      const saved = await client.tasks.retrieve({
        key: draft.key,
        schemas: TCP.Task.WRITE_SCHEMAS,
      });
      expect(saved.config.messages[0].fields[0].channel).toBe(cmd.key);
    });

    const deployed = await deployAndAwaitTask(
      client,
      container,
      draft.key,
      TCP.Task.WRITE_SCHEMAS,
    );
    const [volts, amps] = deployed.config.messages[0].fields;
    expect(volts).toEqual({ field: status.payload.fields[0].key, channel: cmd.key });
    expect(amps.channel).not.toBe(0);
    expect(amps.channel).not.toBe(cmd.key);
  });

  it("should send an unchecked field as zero by leaving it unmapped", async () => {
    const { status, draft } = await setup();
    await renderBusTask(TCP.Task.Write, client, draft.key);
    fireEvent.click(await screen.findByRole("checkbox", { name: "Amps" }));
    await waitFor(async () => {
      const saved = await client.tasks.retrieve({
        key: draft.key,
        schemas: TCP.Task.WRITE_SCHEMAS,
      });
      expect(saved.config.messages[0].fields).toEqual([
        { field: status.payload.fields[0].key, channel: 0 },
      ]);
    });
  });
});
