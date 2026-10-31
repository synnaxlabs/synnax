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
  awaitEditableForm,
  commitFieldInput,
  createTaskReadOnlyClient,
  createWithHook,
  deployAndAwaitTask,
  renderTaskFormTab,
  type RenderTaskFormTabOptions,
  reportTaskStopped,
  selectFromDropdown,
} from "@/platform/task/testutil";
import { getIconButton } from "@/testutil";

const client = createTestClient();

// The form renders read-only until the update grant lands, and a preview field renders
// no input, so wait for it to become editable before querying fields.
const renderRead = async (options: RenderTaskFormTabOptions) => {
  const rendered = await renderTaskFormTab(Modbus.Task.Read, options);
  await awaitEditableForm();
  return rendered;
};

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

describe("Read", () => {
  it("should build channels in the form and create them on the Core on deploy", async () => {
    const dev = await createModbusDevice(client);
    const draft = await createDraft(client, {
      ...Modbus.Task.READ_SCHEMAS.config.parse({}),
      device: dev.key,
    });
    const { container } = await renderRead({
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
    expect(registerCh.dataType.toString()).toBe("uint16");
  });

  it("should bind a new entry to the channel the device already maps", async () => {
    const dev = await createModbusDevice(client);
    const config = { ...Modbus.Task.READ_SCHEMAS.config.parse({}), device: dev.key };
    const firstDraft = await createDraft(client, config);
    const first = await renderRead({ client, taskKey: firstDraft.key });
    await screen.findByText(dev.name);
    fireEvent.click(getIconButton(first.container, "add"));
    await screen.findByText("Coil");
    const firstTask = await deployAndAwaitTask(
      client,
      first.container,
      firstDraft.key,
      Modbus.Task.READ_SCHEMAS,
    );
    first.unmount();
    const existing = await client.channels.retrieve(
      firstTask.config.channels[0].channel,
    );
    const secondDraft = await createDraft(client, config);
    const second = await renderRead({ client, taskKey: secondDraft.key });
    await screen.findByText(dev.name);
    fireEvent.click(getIconButton(second.container, "add"));
    await screen.findByText(existing.name);
    await waitFor(async () => {
      const saved = await client.tasks.retrieve({
        key: secondDraft.key,
        schemas: Modbus.Task.READ_SCHEMAS,
      });
      expect(saved.config.channels[0].channel).toBe(existing.key);
    });
  });

  it("should reuse the existing index and channels when redeploying", async () => {
    const dev = await createModbusDevice(client);
    const draft = await createDraft(client, {
      ...Modbus.Task.READ_SCHEMAS.config.parse({}),
      device: dev.key,
    });
    const first = await renderRead({
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

    const second = await renderRead({
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

  it("should deploy a string register to a string channel", async () => {
    const dev = await createModbusDevice(client);
    const draft = await createDraft(client, {
      ...Modbus.Task.READ_SCHEMAS.config.parse({}),
      device: dev.key,
      channels: [
        Modbus.Task.READ_CHANNEL_SCHEMAS.input_register.parse({
          type: "input_register",
          key: id.create(),
          address: 305,
        }),
      ],
    });
    const { container } = await renderRead({ client, taskKey: draft.key });
    await selectFromDropdown("uint16", "String");
    commitFieldInput(
      await screen.findByLabelText<HTMLInputElement>("String length"),
      "6",
    );
    const created = await deployAndAwaitTask(
      client,
      container,
      draft.key,
      Modbus.Task.READ_SCHEMAS,
    );
    const [str] = created.config.channels;
    expect(str).toMatchObject({ dataType: "string", stringLength: 6 });
    const ch = await client.channels.retrieve(str.channel);
    expect(ch.name).toBe(`${dev.name}_register_input_305_str`);
    expect(ch.dataType.toString()).toBe("string");
  });

  it("should deploy a register bit to a uint8 channel named by its bit", async () => {
    const dev = await createModbusDevice(client);
    const draft = await createDraft(client, {
      ...Modbus.Task.READ_SCHEMAS.config.parse({}),
      device: dev.key,
      channels: [
        Modbus.Task.READ_CHANNEL_SCHEMAS.input_register_bit.parse({
          type: "input_register_bit",
          key: id.create(),
          address: 141,
        }),
      ],
    });
    const { container } = await renderRead({ client, taskKey: draft.key });
    commitFieldInput(await screen.findByLabelText<HTMLInputElement>("Bit"), "5");
    const created = await deployAndAwaitTask(
      client,
      container,
      draft.key,
      Modbus.Task.READ_SCHEMAS,
    );
    const [bit] = created.config.channels;
    expect(bit).toMatchObject({ address: 141, bit: 5 });
    const ch = await client.channels.retrieve(bit.channel);
    expect(ch.name).toBe(`${dev.name}_register_bit_input_141_5`);
    expect(ch.dataType.toString()).toBe("uint8");
  });

  it("should store a six-digit Modicon address of the channel's table", async () => {
    const dev = await createModbusDevice(client);
    const draft = await createDraft(client, {
      ...Modbus.Task.READ_SCHEMAS.config.parse({}),
      device: dev.key,
      channels: [
        Modbus.Task.READ_CHANNEL_SCHEMAS.input_register.parse({
          type: "input_register",
          key: id.create(),
        }),
      ],
    });
    await renderRead({ client, taskKey: draft.key });
    await screen.findByText("300001");
    commitFieldInput(
      await screen.findByLabelText<HTMLInputElement>("Address"),
      "300004",
    );
    await screen.findByText("300004");
    await waitFor(async () => {
      const saved = await client.tasks.retrieve({
        key: draft.key,
        schemas: Modbus.Task.READ_SCHEMAS,
      });
      expect(saved.config.channels[0].address).toBe(3);
    });
  });

  it("should mark an address of another table until a valid entry", async () => {
    const dev = await createModbusDevice(client);
    const draft = await createDraft(client, {
      ...Modbus.Task.READ_SCHEMAS.config.parse({}),
      device: dev.key,
      channels: [
        Modbus.Task.READ_CHANNEL_SCHEMAS.input_register.parse({
          type: "input_register",
          key: id.create(),
        }),
      ],
    });
    await renderRead({ client, taskKey: draft.key });
    const address = await screen.findByLabelText<HTMLInputElement>("Address");
    commitFieldInput(address, "400004");
    await waitFor(() => expect(address.getAttribute("aria-invalid")).toBe("true"));
    expect(screen.getByText("300001")).toBeTruthy();
    commitFieldInput(address, "300004");
    await screen.findByText("300004");
    expect(address.getAttribute("aria-invalid")).toBeNull();
  });

  describe("byte and word order", () => {
    const createRegisterDraft = async (dataType: string, swapWords: boolean) => {
      const dev = await createModbusDevice(client, {
        properties: {
          connection: { host: "localhost", port: 502, swapBytes: false, swapWords },
        },
      });
      const register = Modbus.Task.READ_CHANNEL_SCHEMAS.holding_register.parse({
        type: "holding_register",
        key: id.create(),
        dataType,
      });
      const draft = await createDraft(client, {
        ...Modbus.Task.READ_SCHEMAS.config.parse({}),
        device: dev.key,
        channels: [register],
      });
      return { dev, draft };
    };

    const deployRegister = async (container: HTMLElement, key: task.Key) => {
      const created = await deployAndAwaitTask(
        client,
        container,
        key,
        Modbus.Task.READ_SCHEMAS,
      );
      const [register] = created.config.channels;
      if (register.type !== "holding_register")
        throw new Error(`expected a holding register, got ${register.type}`);
      return register;
    };

    it("should show the device's order and leave the channel unset on deploy", async () => {
      const { draft } = await createRegisterDraft("float32", true);
      const { container } = await renderRead({ client, taskKey: draft.key });
      await screen.findByText("Words: device (swapped)");
      screen.getByText("Bytes: device (not swapped)");
      const register = await deployRegister(container, draft.key);
      expect(register.bytesSwapped).toBeUndefined();
      expect(register.wordsSwapped).toBeUndefined();
    });

    it("should store the override picked on a register channel", async () => {
      const { draft } = await createRegisterDraft("float32", true);
      const { container } = await renderRead({ client, taskKey: draft.key });
      fireEvent.click(await screen.findByText("Bytes: device (not swapped)"));
      fireEvent.click(await screen.findByText("Bytes: swapped"));
      fireEvent.click(screen.getByText("Words: device (swapped)"));
      fireEvent.click(await screen.findByText("Words: not swapped"));
      const register = await deployRegister(container, draft.key);
      expect(register.bytesSwapped).toBe(true);
      expect(register.wordsSwapped).toBe(false);
    });

    it("should not open the order options without permission to update the task", async () => {
      const { draft } = await createRegisterDraft("float32", true);
      await renderTaskFormTab(Modbus.Task.Read, {
        client,
        taskKey: draft.key,
        as: await createTaskReadOnlyClient(client),
      });
      fireEvent.click(await screen.findByText("Bytes: device (not swapped)"));
      expect(screen.queryByText("Bytes: swapped")).toBeNull();
    });

    it("should show only the byte order for a type that fits in one register", async () => {
      const { draft } = await createRegisterDraft("uint16", true);
      await renderRead({ client, taskKey: draft.key });
      await screen.findByText("Bytes: device (not swapped)");
      expect(screen.queryByText(/^Words:/)).toBeNull();
    });
  });

  it("should create the task with one channel", async () => {
    const tsk = await createWithHook(
      Modbus.Task.useCreateRead,
      Modbus.Task.READ_SCHEMAS,
    );
    expect(tsk.config.channels).toHaveLength(1);
  });

  describe("add channel", () => {
    it.each([
      { dataType: "uint16", address: 11 },
      { dataType: "float32", address: 12 },
      { dataType: "float64", address: 14 },
    ])(
      "should place a new channel after the last register of a $dataType channel",
      async ({ dataType, address }) => {
        const dev = await createModbusDevice(client);
        const register = Modbus.Task.READ_CHANNEL_SCHEMAS.holding_register.parse({
          type: "holding_register",
          key: id.create(),
          address: 10,
          dataType,
        });
        const draft = await createDraft(client, {
          ...Modbus.Task.READ_SCHEMAS.config.parse({}),
          device: dev.key,
          channels: [register],
        });
        const { container } = await renderRead({ client, taskKey: draft.key });
        await screen.findByText(dev.name);
        fireEvent.click(getIconButton(container, "add"));
        await waitFor(async () => {
          const saved = await client.tasks.retrieve({
            key: draft.key,
            schemas: Modbus.Task.READ_SCHEMAS,
          });
          expect(saved.config.channels[1]?.address).toBe(address);
        });
      },
    );

    it.each([
      {
        last: "bit 15 of a register",
        channel: Modbus.Task.READ_CHANNEL_SCHEMAS.input_register_bit.parse({
          type: "input_register_bit",
          address: 10,
          bit: 15,
        }),
        next: { address: 11, bit: 0 },
      },
      {
        last: "a string of seven characters",
        channel: Modbus.Task.READ_CHANNEL_SCHEMAS.input_register.parse({
          type: "input_register",
          address: 10,
          dataType: "string",
          stringLength: 7,
        }),
        next: { address: 14 },
      },
    ])("should place a new channel after $last", async ({ channel, next }) => {
      const dev = await createModbusDevice(client);
      const draft = await createDraft(client, {
        ...Modbus.Task.READ_SCHEMAS.config.parse({}),
        device: dev.key,
        channels: [{ ...channel, key: id.create() }],
      });
      const { container } = await renderRead({ client, taskKey: draft.key });
      await screen.findByText(dev.name);
      fireEvent.click(getIconButton(container, "add"));
      await waitFor(async () => {
        const saved = await client.tasks.retrieve({
          key: draft.key,
          schemas: Modbus.Task.READ_SCHEMAS,
        });
        expect(saved.config.channels[1]).toMatchObject(next);
      });
    });
  });
});
