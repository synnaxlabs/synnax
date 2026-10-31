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
import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

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

// The channels of the task, while no menu is open.
const getItems = () => screen.queryAllByRole("option");

// A trigger's label comes from its menu, so a click on the label text does not reach
// the trigger in jsdom.
const findByteOrder = () => screen.findByRole("button", { name: /^Byte order/ });

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
    await waitFor(() => expect(getItems()).toHaveLength(1));
    fireEvent.click(getIconButton(container, "add"));
    await waitFor(() => expect(getItems()).toHaveLength(2));

    fireEvent.click(
      within(getItems()[1]).getByRole("button", { name: /^Register type/ }),
    );
    fireEvent.click(await screen.findByText("Input register"));
    await waitFor(() => within(getItems()[1]).getByText("Input register"));

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
    await waitFor(() => expect(getItems()).toHaveLength(1));
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
    await waitFor(() => expect(getItems()).toHaveLength(1));
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
    await waitFor(() => expect(getItems()).toHaveLength(1));
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
    const address = await screen.findByLabelText<HTMLInputElement>("Address");
    commitFieldInput(address, "300004");
    await waitFor(() => expect(address.value).toBe("3"));
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
      const trigger = await findByteOrder();
      within(trigger).getByText("From device");
      within(trigger).getByText("Big-endian");
      const register = await deployRegister(container, draft.key);
      expect(register.bytesSwapped).toBeUndefined();
      expect(register.wordsSwapped).toBeUndefined();
    });

    it("should name a device without swaps a word swap, as the Driver reads the low word first", async () => {
      const { draft } = await createRegisterDraft("float32", false);
      await renderRead({ client, taskKey: draft.key });
      const trigger = await findByteOrder();
      within(trigger).getByText("Word swap");
      fireEvent.click(trigger);
      await screen.findByText("CDAB");
    });

    it("should store the override picked on a register channel", async () => {
      const { draft } = await createRegisterDraft("float32", true);
      const { container } = await renderRead({ client, taskKey: draft.key });
      fireEvent.click(await findByteOrder());
      fireEvent.click(await screen.findByText("Byte swap"));
      const register = await deployRegister(container, draft.key);
      expect(register.bytesSwapped).toBe(true);
      expect(register.wordsSwapped).toBe(true);
    });

    it("should not open the order options without permission to update the task", async () => {
      const { draft } = await createRegisterDraft("float32", true);
      await renderTaskFormTab(Modbus.Task.Read, {
        client,
        taskKey: draft.key,
        as: await createTaskReadOnlyClient(client),
      });
      fireEvent.click(await findByteOrder());
      expect(screen.queryByText("Byte swap")).toBeNull();
    });

    it("should offer only the two byte orders for a type that fits in one register", async () => {
      const { draft } = await createRegisterDraft("uint16", true);
      await renderRead({ client, taskKey: draft.key });
      fireEvent.click(await findByteOrder());
      await screen.findByText("Little-endian");
      expect(screen.queryByText("Byte swap")).toBeNull();
      expect(screen.queryByText("Word swap")).toBeNull();
    });
  });

  describe("channels", () => {
    const renderChannels = async (channels: Modbus.Task.ReadChannel[]) => {
      const dev = await createModbusDevice(client);
      const draft = await createDraft(client, {
        ...Modbus.Task.READ_SCHEMAS.config.parse({}),
        device: dev.key,
        channels,
      });
      const rendered = await renderRead({ client, taskKey: draft.key });
      await waitFor(() => expect(getItems()).toHaveLength(channels.length));
      return rendered;
    };

    const holding = (address: number, dataType: string) =>
      Modbus.Task.READ_CHANNEL_SCHEMAS.holding_register.parse({
        type: "holding_register",
        key: id.create(),
        address,
        dataType,
      });

    const coil = (address: number) =>
      Modbus.Task.READ_CHANNEL_SCHEMAS.coil.parse({
        type: "coil",
        key: id.create(),
        address,
      });

    it("should show a heading for each column", async () => {
      await renderChannels([coil(0)]);
      for (const name of ["Type", "Address", "Data type", "Byte order", "Channel"])
        expect(screen.getByText(name)).toBeTruthy();
    });

    it("should show the last address of a channel that spans several", async () => {
      await renderChannels([holding(10, "float32")]);
      expect(within(getItems()[0]).getByText("–11")).toBeTruthy();
    });

    it("should mark the addresses of channels that read the same register", async () => {
      await renderChannels([holding(10, "float32"), holding(11, "uint16"), coil(11)]);
      const marked = getItems().map(
        (item) => item.querySelector(".console-modbus__address--overlap") != null,
      );
      expect(marked).toEqual([true, true, false]);
    });

    it("should map each register type in use on its own lane", async () => {
      const { container } = await renderChannels([holding(10, "uint16"), coil(0)]);
      const lanes = [...container.querySelectorAll(".console-modbus__lane-name")].map(
        (lane) => lane.textContent,
      );
      expect(lanes).toEqual(["Coil", "Holding register"]);
    });

    it("should draw one band where many channels read the same addresses", async () => {
      const { container } = await renderChannels([
        holding(10, "float32"),
        holding(10, "float32"),
        holding(10, "float32"),
      ]);
      expect(container.querySelectorAll(".console-modbus__overlap")).toHaveLength(1);
    });

    it("should show the Modicon address after the offset", async () => {
      await renderChannels([holding(10, "float32")]);
      expect(within(getItems()[0]).getByText("400011")).toBeTruthy();
    });

    it("should add a channel at the free address the map is clicked at", async () => {
      const { container } = await renderChannels([
        holding(0, "uint16"),
        holding(3, "uint16"),
      ]);
      const track = container.querySelector<HTMLElement>(".console-modbus__track");
      if (track == null) throw new Error("no track");
      vi.spyOn(track, "getBoundingClientRect").mockReturnValue(
        new DOMRect(0, 0, 400, 8),
      );
      fireEvent.pointerMove(track, { clientX: 150 });
      fireEvent.click(track);
      await waitFor(() => expect(getItems()).toHaveLength(3));
      const address = within(getItems()[2]).getByRole<HTMLInputElement>("textbox", {
        name: "Address",
      });
      expect(address.value).toBe("1");
    });

    it("should not add a channel from the map of a read-only task", async () => {
      const dev = await createModbusDevice(client);
      const draft = await createDraft(client, {
        ...Modbus.Task.READ_SCHEMAS.config.parse({}),
        device: dev.key,
        channels: [holding(0, "uint16"), holding(3, "uint16")],
      });
      const { container } = await renderTaskFormTab(Modbus.Task.Read, {
        client,
        taskKey: draft.key,
        as: await createTaskReadOnlyClient(client),
      });
      await waitFor(() => expect(getItems()).toHaveLength(2));
      const track = container.querySelector<HTMLElement>(".console-modbus__track");
      if (track == null) throw new Error("no track");
      vi.spyOn(track, "getBoundingClientRect").mockReturnValue(
        new DOMRect(0, 0, 400, 8),
      );
      fireEvent.pointerMove(track, { clientX: 150 });
      expect(container.querySelector(".console-modbus__free")).toBeNull();
      fireEvent.click(track);
      expect(getItems()).toHaveLength(2);
    });

    it("should not add a channel when a used address is clicked", async () => {
      const { container } = await renderChannels([coil(0), coil(1)]);
      const segment = container.querySelector(".console-modbus__segment");
      if (segment == null) throw new Error("no segment");
      fireEvent.click(segment);
      expect(getItems()).toHaveLength(2);
    });

    it("should open a channel's menu from its segment in the map", async () => {
      const { container } = await renderChannels([coil(0), coil(1)]);
      fireEvent.contextMenu(container.querySelectorAll(".console-modbus__segment")[1]);
      fireEvent.click(await screen.findByText("Disable"));
      await waitFor(() =>
        expect(
          getItems().map((item) =>
            item.classList.contains("console-list-channel-item--disabled"),
          ),
        ).toEqual([false, true]),
      );
      expect(
        container.querySelectorAll(".console-modbus__segment--disabled"),
      ).toHaveLength(1);
    });

    it("should not select a channel when a field is edited", async () => {
      await renderChannels([coil(0)]);
      fireEvent.click(within(getItems()[0]).getByRole("textbox", { name: "Address" }));
      expect(getItems()[0].getAttribute("aria-selected")).toBe("false");
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
