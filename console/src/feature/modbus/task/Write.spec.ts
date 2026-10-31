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
  createChannelReadOnlyClient,
  createTaskReadOnlyClient,
  createWithHook,
  deployAndAwaitTask,
  renderTaskFormTab,
  type RenderTaskFormTabOptions,
  reportTaskStopped,
} from "@/platform/task/testutil";
import { awaitTextEditingElement, commitTextEdit, getIconButton } from "@/testutil";

const client = createTestClient();

// The channels of the task, while no menu is open.
const getItems = () => screen.queryAllByRole("option");

// The form renders read-only until the update grant lands, and a preview field renders
// no input, so wait for it to become editable before querying fields.
const renderWrite = async (options: RenderTaskFormTabOptions) => {
  const rendered = await renderTaskFormTab(Modbus.Task.Write, options);
  await awaitEditableForm();
  return rendered;
};

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

describe("Write", () => {
  it("should create command channels and indexes for the built channels on deploy", async () => {
    const dev = await createModbusDevice(client);
    const draft = await createDraft(client, {
      ...Modbus.Task.WRITE_SCHEMAS.config.parse({}),
      device: dev.key,
    });
    const { container } = await renderWrite({
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
    fireEvent.click(await screen.findByText("Holding register"));
    await waitFor(() => within(getItems()[1]).getByText("Holding register"));

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
    expect(holdingCmd.name).toBe(`${dev.name}_holding_register_output_1_u16_cmd`);
    expect(holdingCmd.dataType.toString()).toBe("uint16");

    const updated = await client.devices.retrieve({
      key: dev.key,
      schemas: Modbus.Device.SCHEMAS,
    });
    expect(updated.properties.write.channels["coil-output-0"]).toBe(coil.channel);
  });

  it("should bind a new entry to the channel the device already maps", async () => {
    const dev = await createModbusDevice(client);
    const config = { ...Modbus.Task.WRITE_SCHEMAS.config.parse({}), device: dev.key };
    const firstDraft = await createDraft(client, config);
    const first = await renderWrite({ client, taskKey: firstDraft.key });
    await screen.findByText(dev.name);
    fireEvent.click(getIconButton(first.container, "add"));
    await waitFor(() => expect(getItems()).toHaveLength(1));
    const firstTask = await deployAndAwaitTask(
      client,
      first.container,
      firstDraft.key,
      Modbus.Task.WRITE_SCHEMAS,
    );
    first.unmount();
    const cmd = await client.channels.retrieve(firstTask.config.channels[0].channel);
    const secondDraft = await createDraft(client, config);
    const second = await renderWrite({ client, taskKey: secondDraft.key });
    await screen.findByText(dev.name);
    fireEvent.click(getIconButton(second.container, "add"));
    await screen.findByText(cmd.name);
    await waitFor(async () => {
      const saved = await client.tasks.retrieve({
        key: secondDraft.key,
        schemas: Modbus.Task.WRITE_SCHEMAS,
      });
      expect(saved.config.channels[0].channel).toBe(cmd.key);
    });
  });

  it("should reuse existing command channels when redeploying", async () => {
    const dev = await createModbusDevice(client);
    const draft = await createDraft(client, {
      ...Modbus.Task.WRITE_SCHEMAS.config.parse({}),
      device: dev.key,
    });
    const first = await renderWrite({
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
      Modbus.Task.WRITE_SCHEMAS,
    );
    const afterFirst = await client.devices.retrieve({
      key: dev.key,
      schemas: Modbus.Device.SCHEMAS,
    });
    await reportTaskStopped(client, deployed.payload);
    first.unmount();

    const second = await renderWrite({
      client,
      taskKey: draft.key,
    });
    await waitFor(() => expect(getItems()).toHaveLength(1));
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

  it("should reuse a holding register command channel when redeploying", async () => {
    const dev = await createModbusDevice(client);
    const draft = await createDraft(client, {
      ...Modbus.Task.WRITE_SCHEMAS.config.parse({}),
      device: dev.key,
      channels: [
        Modbus.Task.WRITE_CHANNEL_SCHEMAS.holding_register.parse({
          type: "holding_register",
          key: id.create(),
          dataType: "uint16",
        }),
      ],
    });
    for (let i = 0; i < 2; i++) {
      const { container, unmount } = await renderWrite({ client, taskKey: draft.key });
      const deployed = await deployAndAwaitTask(
        client,
        container,
        draft.key,
        Modbus.Task.WRITE_SCHEMAS,
      );
      await reportTaskStopped(client, deployed.payload);
      unmount();
    }
    const updated = await client.devices.retrieve({
      key: dev.key,
      schemas: Modbus.Device.SCHEMAS,
    });
    expect(Object.keys(updated.properties.write.channels)).toEqual([
      "holding-register-output-0-uint16",
    ]);
    const matches = await client.channels.retrieve([
      `${dev.name}_holding_register_output_0_u16_cmd`,
    ]);
    expect(matches).toHaveLength(1);
  });

  describe("legacy command channel keys", () => {
    const createCommand = async (dataType: string) => {
      const index = await client.channels.create({
        name: id.create(),
        dataType: "timestamp",
        isIndex: true,
      });
      return await client.channels.create({
        name: id.create(),
        dataType,
        index: index.key,
      });
    };

    const deployFloat = async (legacyKey: string, cmdKey: number) => {
      const dev = await createModbusDevice(client, {
        properties: { write: { channels: { [legacyKey]: cmdKey } } },
      });
      const draft = await createDraft(client, {
        ...Modbus.Task.WRITE_SCHEMAS.config.parse({}),
        device: dev.key,
        channels: [
          Modbus.Task.WRITE_CHANNEL_SCHEMAS.holding_register.parse({
            type: "holding_register",
            key: id.create(),
            dataType: "float32",
          }),
        ],
      });
      const { container, unmount } = await renderWrite({ client, taskKey: draft.key });
      const deployed = await deployAndAwaitTask(
        client,
        container,
        draft.key,
        Modbus.Task.WRITE_SCHEMAS,
      );
      await reportTaskStopped(client, deployed.payload);
      unmount();
      const updated = await client.devices.retrieve({
        key: dev.key,
        schemas: Modbus.Device.SCHEMAS,
      });
      return { channel: deployed.config.channels[0].channel, updated };
    };

    it.each([
      ["Python", "holding-register-output-0"],
      ["an older Console", "holding-register_output-0"],
    ])(
      "should adopt a command channel that %s saved without a data type",
      async (_, legacyKey) => {
        const cmd = await createCommand("float32");
        const { channel, updated } = await deployFloat(legacyKey, cmd.key);
        expect(channel).toBe(cmd.key);
        expect(updated.properties.write.channels).toEqual({
          "holding-register-output-0-float32": cmd.key,
        });
      },
    );

    it("should not adopt a legacy command channel of another data type", async () => {
      const cmd = await createCommand("uint16");
      const { channel, updated } = await deployFloat(
        "holding-register-output-0",
        cmd.key,
      );
      expect(channel).not.toBe(cmd.key);
      expect(updated.properties.write.channels["holding-register-output-0"]).toBe(
        cmd.key,
      );
    });

    it("should create a command channel when a legacy key names a deleted channel", async () => {
      const cmd = await createCommand("float32");
      await client.channels.delete(cmd.key);
      const { channel, updated } = await deployFloat(
        "holding-register-output-0",
        cmd.key,
      );
      expect(channel).not.toBe(cmd.key);
      expect(
        updated.properties.write.channels["holding-register-output-0-float32"],
      ).toBe(channel);
    });
  });

  it("should create a new command channel when the data type changes", async () => {
    const dev = await createModbusDevice(client);
    const deployRegister = async (dataType: string) => {
      const draft = await createDraft(client, {
        ...Modbus.Task.WRITE_SCHEMAS.config.parse({}),
        device: dev.key,
        channels: [
          Modbus.Task.WRITE_CHANNEL_SCHEMAS.holding_register.parse({
            type: "holding_register",
            key: id.create(),
            dataType,
          }),
        ],
      });
      const { container, unmount } = await renderWrite({ client, taskKey: draft.key });
      const deployed = await deployAndAwaitTask(
        client,
        container,
        draft.key,
        Modbus.Task.WRITE_SCHEMAS,
      );
      await reportTaskStopped(client, deployed.payload);
      unmount();
      return deployed.config.channels[0].channel;
    };
    const first = await deployRegister("uint16");
    const second = await deployRegister("float32");
    expect(second).not.toBe(first);
    const cmd = await client.channels.retrieve(second);
    expect(cmd.dataType.toString()).toBe("float32");
    expect(await deployRegister("uint16")).toBe(first);
  });

  it("should rename and remove a channel through the context menu", async () => {
    const dev = await createModbusDevice(client);
    const draft = await createDraft(client, {
      ...Modbus.Task.WRITE_SCHEMAS.config.parse({}),
      device: dev.key,
    });
    const { container } = await renderWrite({
      client,
      taskKey: draft.key,
    });
    await screen.findByText(dev.name);
    fireEvent.click(getIconButton(container, "add"));
    fireEvent.contextMenu(await screen.findByText(`${dev.name}_coil_output_0_cmd`));
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
    const { container } = await renderWrite({
      client,
      taskKey: draft.key,
      as: await createChannelReadOnlyClient(client),
    });
    await screen.findByText(dev.name);
    fireEvent.click(getIconButton(container, "add"));
    fireEvent.contextMenu(await screen.findByText(`${dev.name}_coil_output_0_cmd`));
    // Remove is ungated, so its presence proves the menu resolved before the
    // absence below is read.
    expect(await screen.findByText("Remove")).toBeTruthy();
    expect(screen.queryByText("Rename")).toBeNull();
  });

  it("should store a byte order override on a holding register channel", async () => {
    const dev = await createModbusDevice(client, {
      properties: {
        connection: { host: "localhost", port: 502, swapBytes: true, swapWords: false },
      },
    });
    const draft = await createDraft(client, {
      ...Modbus.Task.WRITE_SCHEMAS.config.parse({}),
      device: dev.key,
      channels: [
        Modbus.Task.WRITE_CHANNEL_SCHEMAS.holding_register.parse({
          type: "holding_register",
          key: id.create(),
          dataType: "float32",
        }),
      ],
    });
    const { container } = await renderWrite({ client, taskKey: draft.key });
    const trigger = await screen.findByRole("button", { name: /^Byte order/ });
    within(trigger).getByText("Little-endian");
    fireEvent.click(trigger);
    fireEvent.click(await screen.findByText("Word swap"));
    const created = await deployAndAwaitTask(
      client,
      container,
      draft.key,
      Modbus.Task.WRITE_SCHEMAS,
    );
    const [register] = created.config.channels;
    if (register.type !== "holding_register")
      throw new Error(`expected a holding register, got ${register.type}`);
    expect(register.bytesSwapped).toBe(false);
    expect(register.wordsSwapped).toBe(false);
  });

  it("should deploy a holding register bit to a uint8 command channel", async () => {
    const dev = await createModbusDevice(client);
    const draft = await createDraft(client, {
      ...Modbus.Task.WRITE_SCHEMAS.config.parse({}),
      device: dev.key,
      channels: [
        Modbus.Task.WRITE_CHANNEL_SCHEMAS.holding_register_bit.parse({
          type: "holding_register_bit",
          key: id.create(),
          address: 4,
          bit: 3,
        }),
      ],
    });
    const { container } = await renderWrite({ client, taskKey: draft.key });
    await screen.findByText(dev.name);
    const created = await deployAndAwaitTask(
      client,
      container,
      draft.key,
      Modbus.Task.WRITE_SCHEMAS,
    );
    const [bit] = created.config.channels;
    const cmd = await client.channels.retrieve(bit.channel);
    expect(cmd.name).toBe(`${dev.name}_holding_register_bit_output_4_3_cmd`);
    expect(cmd.dataType.toString()).toBe("uint8");
    const updated = await client.devices.retrieve({
      key: dev.key,
      schemas: Modbus.Device.SCHEMAS,
    });
    expect(Object.values(updated.properties.write.channels)).toEqual([bit.channel]);
  });

  describe("address", () => {
    const renderRegister = async () => {
      const dev = await createModbusDevice(client);
      const draft = await createDraft(client, {
        ...Modbus.Task.WRITE_SCHEMAS.config.parse({}),
        device: dev.key,
        channels: [
          Modbus.Task.WRITE_CHANNEL_SCHEMAS.holding_register.parse({
            type: "holding_register",
            key: id.create(),
          }),
        ],
      });
      const { container } = await renderWrite({ client, taskKey: draft.key });
      await screen.findByText("400001");
      const input = container.querySelector<HTMLInputElement>(
        'input[aria-label="Address"]',
      );
      if (input == null) throw new Error("expected an address input");
      return { draft, input };
    };

    const enter = (input: HTMLInputElement, value: string) => {
      fireEvent.focus(input);
      fireEvent.change(input, { target: { value } });
      fireEvent.blur(input);
    };

    it("should store a six-digit Modicon address as an offset", async () => {
      const { draft, input } = await renderRegister();
      enter(input, "400004");
      await screen.findByText("400004");
      await waitFor(async () => {
        const saved = await client.tasks.retrieve({
          key: draft.key,
          schemas: Modbus.Task.WRITE_SCHEMAS,
        });
        expect(saved.config.channels[0].address).toBe(3);
      });
    });

    it("should keep the address and mark the field for another table's prefix", async () => {
      const { input } = await renderRegister();
      enter(input, "300004");
      await waitFor(() =>
        expect(input.closest(".lyra-input")?.classList.contains("lyra--error")).toBe(
          true,
        ),
      );
      expect(input.value).toBe("0");
      screen.getByText("400001");
    });

    it("should clear the mark when a valid entry keeps the stored address", async () => {
      const { input } = await renderRegister();
      const marked = () =>
        input.closest(".lyra-input")?.classList.contains("lyra--error");
      enter(input, "300004");
      await waitFor(() => expect(marked()).toBe(true));
      enter(input, "400001");
      await waitFor(() => expect(marked()).toBe(false));
    });
  });

  describe("channels", () => {
    const renderChannels = async (channels: Modbus.Task.WriteChannel[]) => {
      const dev = await createModbusDevice(client);
      const draft = await createDraft(client, {
        ...Modbus.Task.WRITE_SCHEMAS.config.parse({}),
        device: dev.key,
        channels,
      });
      const rendered = await renderWrite({ client, taskKey: draft.key });
      await waitFor(() => expect(getItems()).toHaveLength(channels.length));
      return rendered;
    };

    const holding = (address: number, dataType: string) =>
      Modbus.Task.WRITE_CHANNEL_SCHEMAS.holding_register.parse({
        type: "holding_register",
        key: id.create(),
        address,
        dataType,
      });

    it("should show a heading for each column", async () => {
      await renderChannels([holding(0, "uint16")]);
      for (const name of ["Type", "Address", "Data type", "Byte order", "Channel"])
        expect(screen.getByText(name)).toBeTruthy();
    });

    it("should offer only the register types a task can write", async () => {
      await renderChannels([holding(0, "uint16")]);
      fireEvent.click(
        within(getItems()[0]).getByRole("button", { name: /^Register type/ }),
      );
      await screen.findByText("Holding register bit");
      expect(screen.queryByText("Input register")).toBeNull();
      expect(screen.queryByText("Discrete input")).toBeNull();
    });

    it("should mark the addresses of channels that write the same register", async () => {
      await renderChannels([holding(10, "float32"), holding(11, "uint16")]);
      const marked = getItems().map(
        (item) => item.querySelector(".console-modbus__address--overlap") != null,
      );
      expect(marked).toEqual([true, true]);
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
      const item = getItems()[2];
      expect(within(item).getByText("Holding register")).toBeTruthy();
      const address = within(item).getByRole<HTMLInputElement>("textbox", {
        name: "Address",
      });
      expect(address.value).toBe("1");
    });

    it("should not add a channel from the map of a read-only task", async () => {
      const dev = await createModbusDevice(client);
      const draft = await createDraft(client, {
        ...Modbus.Task.WRITE_SCHEMAS.config.parse({}),
        device: dev.key,
        channels: [holding(0, "uint16"), holding(3, "uint16")],
      });
      const { container } = await renderTaskFormTab(Modbus.Task.Write, {
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
  });

  it("should create the task with one channel", async () => {
    const tsk = await createWithHook(
      Modbus.Task.useCreateWrite,
      Modbus.Task.WRITE_SCHEMAS,
    );
    expect(tsk.config.channels).toHaveLength(1);
  });

  describe("add channel", () => {
    it.each([
      { bit: 3, address: 10, nextBit: 4 },
      { bit: 15, address: 11, nextBit: 0 },
    ])(
      "should place a new channel after bit $bit of a register",
      async ({ bit, address, nextBit }) => {
        const dev = await createModbusDevice(client);
        const draft = await createDraft(client, {
          ...Modbus.Task.WRITE_SCHEMAS.config.parse({}),
          device: dev.key,
          channels: [
            Modbus.Task.WRITE_CHANNEL_SCHEMAS.holding_register_bit.parse({
              type: "holding_register_bit",
              key: id.create(),
              address: 10,
              bit,
            }),
          ],
        });
        const { container } = await renderWrite({ client, taskKey: draft.key });
        await screen.findByText(dev.name);
        fireEvent.click(getIconButton(container, "add"));
        await waitFor(async () => {
          const saved = await client.tasks.retrieve({
            key: draft.key,
            schemas: Modbus.Task.WRITE_SCHEMAS,
          });
          expect(saved.config.channels[1]).toMatchObject({ address, bit: nextBit });
        });
      },
    );

    it.each([
      { dataType: "uint16", address: 11 },
      { dataType: "float32", address: 12 },
      { dataType: "float64", address: 14 },
    ])(
      "should place a new channel after the last register of a $dataType channel",
      async ({ dataType, address }) => {
        const dev = await createModbusDevice(client);
        const register = Modbus.Task.WRITE_CHANNEL_SCHEMAS.holding_register.parse({
          type: "holding_register",
          key: id.create(),
          address: 10,
          dataType,
        });
        const draft = await createDraft(client, {
          ...Modbus.Task.WRITE_SCHEMAS.config.parse({}),
          device: dev.key,
          channels: [register],
        });
        const { container } = await renderWrite({ client, taskKey: draft.key });
        await screen.findByText(dev.name);
        fireEvent.click(getIconButton(container, "add"));
        await waitFor(async () => {
          const saved = await client.tasks.retrieve({
            key: draft.key,
            schemas: Modbus.Task.WRITE_SCHEMAS,
          });
          expect(saved.config.channels[1]?.address).toBe(address);
        });
      },
    );
  });
});
