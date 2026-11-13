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
import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { type z } from "zod";

import { LabJack } from "@/feature/labjack";
import {
  createAnalogWriteChannel,
  createDigitalWriteChannel,
  createLabJackDevice,
} from "@/feature/labjack/testutil";
import {
  awaitEditableForm,
  createWithHook,
  deployAndAwaitTask,
  findChannelListItem,
  renderTaskFormTab,
  type RenderTaskFormTabOptions,
} from "@/platform/task/testutil";
import { findDialogTriggerByText, uniqueName } from "@/testutil";

const client = createTestClient();

const typeTriggers = (): HTMLElement[] =>
  screen.getAllByRole("button", { name: /^Channel type/ });

const chooseType = async (index: number, type: "Analog" | "Digital") => {
  fireEvent.click(typeTriggers()[index]);
  fireEvent.click(await screen.findByRole("option", { name: type }));
};

// The form renders read-only until the update grant lands, and a preview field renders
// no input, so wait for it to become editable before querying fields.
const renderWrite = async (options: RenderTaskFormTabOptions = {}) => {
  const rendered = await renderTaskFormTab(LabJack.Task.Write, {
    task: ZERO_DRAFT,
    ...options,
  });
  await awaitEditableForm();
  return rendered;
};

const createConfig = (
  device: string,
  channels: LabJack.Task.WriteChannel[],
): z.infer<LabJack.Task.WriteSchemas["config"]> => ({
  ...LabJack.Task.WRITE_SCHEMAS.config.parse({}),
  device,
  channels,
});

// Drafts carry no key; the created row mints its own.
const ZERO_DRAFT: task.New<LabJack.Task.WriteSchemas> = {
  name: "LabJack write task",
  type: LabJack.Task.WRITE_TYPE,
  config: LabJack.Task.WRITE_SCHEMAS.config.parse({}),
};

const createDraft = async (
  client: Synnax,
  config: z.infer<LabJack.Task.WriteSchemas["config"]>,
) => await client.tasks.create({ ...ZERO_DRAFT, config }, LabJack.Task.WRITE_SCHEMAS);

const savedPorts = async (key: string) => {
  const saved = await client.tasks.retrieve({
    key,
    schemas: LabJack.Task.WRITE_SCHEMAS,
  });
  return saved.config.channels.map(({ type, port }) => [type, port]);
};

describe("Write", () => {
  it("should render output channels with port and type selects", async () => {
    const dev = await createLabJackDevice(client);
    const draft = await createDraft(
      client,
      createConfig(dev.key, [
        createDigitalWriteChannel("DIO4"),
        createAnalogWriteChannel("DAC0"),
      ]),
    );
    await renderWrite({ client, taskKey: draft.key });
    await waitFor(() => expect(screen.getByText("FIO4")).toBeTruthy());
    expect(screen.getByText("DAC0")).toBeTruthy();
    expect(typeTriggers().map((t) => t.textContent)).toEqual(["Digital", "Analog"]);
  });

  it("should swap the channel to the analog port space when Analog is chosen", async () => {
    const dev = await createLabJackDevice(client);
    const draft = await createDraft(
      client,
      createConfig(dev.key, [createDigitalWriteChannel("DIO4")]),
    );
    await renderWrite({ client, taskKey: draft.key });
    await waitFor(() => expect(screen.getByText("FIO4")).toBeTruthy());
    await chooseType(0, "Analog");
    await waitFor(() => expect(screen.getByText("DAC0")).toBeTruthy());
    expect(screen.queryByText("FIO4")).toBeNull();
  });

  it("should add a zero channel when the list is empty and Add is pressed", async () => {
    const dev = await createLabJackDevice(client);
    const draft = await createDraft(client, createConfig(dev.key, []));
    await renderWrite({ client, taskKey: draft.key });
    await waitFor(() => expect(screen.getByText("No channels in task")).toBeTruthy());
    fireEvent.click(screen.getByRole("button", { name: "Add channel" }));
    await waitFor(() => expect(screen.getByText("FIO4")).toBeTruthy());
  });

  it("should add the next open port of the same type when Add is pressed", async () => {
    const dev = await createLabJackDevice(client);
    const draft = await createDraft(
      client,
      createConfig(dev.key, [createDigitalWriteChannel("DIO4")]),
    );
    await renderWrite({ client, taskKey: draft.key });
    await waitFor(() => expect(screen.getByText("FIO4")).toBeTruthy());
    fireEvent.click(screen.getByRole("button", { name: "Add channel" }));
    await waitFor(() => expect(screen.getByText("FIO5")).toBeTruthy());
  });

  it("should copy the selected channel, not the last one, when Add is pressed", async () => {
    const dev = await createLabJackDevice(client);
    const draft = await createDraft(
      client,
      createConfig(dev.key, [
        createDigitalWriteChannel("DIO4"),
        createAnalogWriteChannel("DAC0"),
      ]),
    );
    await renderWrite({ client, taskKey: draft.key });
    const fio4 = await findChannelListItem("FIO4");
    fireEvent.click(within(fio4).getByRole("checkbox", { name: "Select" }));
    fireEvent.click(screen.getByRole("button", { name: "Add channel" }));
    await waitFor(async () =>
      expect(await savedPorts(draft.key)).toEqual([
        ["digital", "DIO4"],
        ["analog", "DAC0"],
        ["digital", "DIO5"],
      ]),
    );
  });

  it("should update the port from the grid the port button opens", async () => {
    const dev = await createLabJackDevice(client);
    const draft = await createDraft(
      client,
      createConfig(dev.key, [
        createDigitalWriteChannel("DIO4"),
        createDigitalWriteChannel("DIO8"),
      ]),
    );
    await renderWrite({ client, taskKey: draft.key });
    fireEvent.click(await findDialogTriggerByText("FIO4"));
    const dialog = await screen.findByRole("dialog");
    const bank = within(dialog).getByRole("group", { name: "EIO0 to EIO7" });
    const eio0 = within(bank).getByRole("button", { name: "EIO0" });
    expect(eio0.className).toContain("console-port-grid__cell--used");
    fireEvent.click(within(bank).getByRole("button", { name: "EIO2" }));
    await findDialogTriggerByText("EIO2");
    expect(screen.queryByRole("dialog")).toBeNull();
    await waitFor(async () =>
      expect(await savedPorts(draft.key)).toEqual([
        ["digital", "DIO10"],
        ["digital", "DIO8"],
      ]),
    );
  });

  it("should name the port a click in the port picker moves the channel to", async () => {
    const dev = await createLabJackDevice(client);
    const draft = await createDraft(
      client,
      createConfig(dev.key, [createDigitalWriteChannel("DIO4")]),
    );
    await renderWrite({ client, taskKey: draft.key });
    fireEvent.click(await findDialogTriggerByText("FIO4"));
    const dialog = await screen.findByRole("dialog");
    const bank = within(dialog).getByRole("group", { name: "EIO0 to EIO7" });
    const eio2 = within(bank).getByRole("button", { name: "EIO2" });
    fireEvent.pointerOver(eio2, { pointerType: "mouse" });
    const tooltip = await screen.findByRole("tooltip");
    expect(tooltip.textContent).toBe("Move to EIO2");
    expect(eio2.className).not.toContain("console-port-grid__cell--addable");
  });

  it("should offer no channel actions on a free port of the port picker", async () => {
    const dev = await createLabJackDevice(client);
    const draft = await createDraft(
      client,
      createConfig(dev.key, [createDigitalWriteChannel("DIO4")]),
    );
    await renderWrite({ client, taskKey: draft.key });
    fireEvent.click(await findDialogTriggerByText("FIO4"));
    const dialog = await screen.findByRole("dialog");
    const bank = within(dialog).getByRole("group", { name: "EIO0 to EIO7" });
    fireEvent.contextMenu(within(bank).getByRole("button", { name: "EIO2" }));
    await waitFor(() => expect(screen.getByText("Reload Console")).toBeTruthy());
    expect(screen.queryByText("Remove")).toBeNull();
  });

  it("should add a channel on the lowest free port of the device", async () => {
    const dev = await createLabJackDevice(client, { model: LabJack.Device.T7_MODEL });
    const draft = await createDraft(client, createConfig(dev.key, []));
    await renderWrite({ client, taskKey: draft.key });
    fireEvent.click(await screen.findByText("Add channel"));
    await waitFor(async () =>
      expect(await savedPorts(draft.key)).toEqual([["digital", "DIO0"]]),
    );
  });

  it("should move a channel to the lowest free port of its new type", async () => {
    const dev = await createLabJackDevice(client);
    const draft = await createDraft(
      client,
      createConfig(dev.key, [
        createDigitalWriteChannel("DIO4"),
        createAnalogWriteChannel("DAC0"),
      ]),
    );
    await renderWrite({ client, taskKey: draft.key });
    await findDialogTriggerByText("FIO4");
    await chooseType(0, "Analog");
    await waitFor(async () =>
      expect(await savedPorts(draft.key)).toEqual([
        ["analog", "DAC1"],
        ["analog", "DAC0"],
      ]),
    );
  });

  it("should add a channel on a free port, of the type the port takes", async () => {
    const dev = await createLabJackDevice(client);
    const draft = await createDraft(
      client,
      createConfig(dev.key, [createDigitalWriteChannel("DIO4")]),
    );
    await renderWrite({ client, taskKey: draft.key });
    const section = await screen.findByRole("group", { name: "Analog outputs" });
    fireEvent.click(within(section).getByRole("button", { name: "DAC1" }));
    await waitFor(async () =>
      expect(await savedPorts(draft.key)).toEqual([
        ["digital", "DIO4"],
        ["analog", "DAC1"],
      ]),
    );
    expect(screen.getByText("1 of 2 ports used")).toBeTruthy();
  });

  it("should name the port a click on a free port adds a channel on", async () => {
    const dev = await createLabJackDevice(client);
    const draft = await createDraft(
      client,
      createConfig(dev.key, [createDigitalWriteChannel("DIO4")]),
    );
    await renderWrite({ client, taskKey: draft.key });
    const section = await screen.findByRole("group", { name: "Analog outputs" });
    const dac1 = within(section).getByRole("button", { name: "DAC1" });
    expect(dac1.className).toContain("console-port-grid__cell--addable");
    fireEvent.pointerOver(dac1, { pointerType: "mouse" });
    const tooltip = await screen.findByRole("tooltip");
    expect(tooltip.textContent).toBe("Add a channel on DAC1");
  });

  it("should mark a channel's port while the pointer is on its item", async () => {
    const dev = await createLabJackDevice(client);
    const draft = await createDraft(
      client,
      createConfig(dev.key, [createAnalogWriteChannel("DAC1")]),
    );
    await renderWrite({ client, taskKey: draft.key });
    const section = await screen.findByRole("group", { name: "Analog outputs" });
    const dac1 = within(section).getByRole("button", { name: "DAC1" });
    fireEvent.pointerEnter(screen.getByRole("option"));
    expect(dac1.className).toContain("console-port-grid__cell--hovered");
    fireEvent.pointerLeave(screen.getByRole("option"));
    expect(dac1.className).not.toContain("console-port-grid__cell--hovered");
  });

  it("should mark a channel's item while the pointer is on its port", async () => {
    const dev = await createLabJackDevice(client);
    const draft = await createDraft(
      client,
      createConfig(dev.key, [createAnalogWriteChannel("DAC1")]),
    );
    await renderWrite({ client, taskKey: draft.key });
    const section = await screen.findByRole("group", { name: "Analog outputs" });
    const dac1 = within(section).getByRole("button", { name: "DAC1" });
    fireEvent.pointerEnter(dac1);
    expect(screen.getByRole("option").className).toContain("lyra--hovered");
    fireEvent.pointerLeave(dac1);
    expect(screen.getByRole("option").className).not.toContain("lyra--hovered");
  });

  it("should mark each channel on a port another channel uses", async () => {
    const dev = await createLabJackDevice(client);
    const draft = await createDraft(
      client,
      createConfig(dev.key, [
        createAnalogWriteChannel("DAC0"),
        createAnalogWriteChannel("DAC1"),
        createAnalogWriteChannel("DAC0"),
      ]),
    );
    await renderWrite({ client, taskKey: draft.key });
    const trigger = (text: HTMLElement) => text.closest(".lyra-dialog__trigger");
    await waitFor(() => {
      for (const text of screen.getAllByText("DAC0"))
        expect(trigger(text)?.getAttribute("aria-invalid")).toBe("true");
    });
    expect(trigger(screen.getByText("DAC1"))?.getAttribute("aria-invalid")).toBeNull();
    fireEvent.pointerOver(screen.getAllByText("DAC0")[0], { pointerType: "mouse" });
    const tooltip = await screen.findByRole("tooltip");
    expect(tooltip.textContent).toBe("Another channel uses DAC0");
  });

  it("should create command and state channels, update the device, and save the task", async () => {
    const dev = await createLabJackDevice(client);
    const draft = await createDraft(
      client,
      createConfig(dev.key, [
        createDigitalWriteChannel("DIO4"),
        createAnalogWriteChannel("DAC0"),
      ]),
    );
    const { container } = await renderWrite({ client, taskKey: draft.key });
    const created = await deployAndAwaitTask(
      client,
      container,
      draft.key,
      LabJack.Task.WRITE_SCHEMAS,
    );
    expect(created.type).toBe(LabJack.Task.WRITE_TYPE);
    expect(created.rack).toBe(dev.rack);
    const [doCh, aoCh] = created.config.channels;
    expect(doCh.cmdChannel).not.toBe(0);
    expect(doCh.stateChannel).not.toBe(0);
    expect(aoCh.cmdChannel).not.toBe(0);
    expect(aoCh.stateChannel).not.toBe(0);

    const identifier = dev.properties.identifier;
    const doCmd = await client.channels.retrieve(doCh.cmdChannel);
    expect(doCmd.name).toBe(`${identifier}_DIO4_cmd`);
    expect(doCmd.dataType.toString()).toBe("uint8");
    const doCmdIndex = await client.channels.retrieve(doCmd.index);
    expect(doCmdIndex.name).toBe(`${identifier}_DIO4_cmd_time`);
    const doState = await client.channels.retrieve(doCh.stateChannel);
    expect(doState.name).toBe(`${identifier}_DIO4_state`);
    const aoCmd = await client.channels.retrieve(aoCh.cmdChannel);
    expect(aoCmd.dataType.toString()).toBe("float32");

    const updated = await client.devices.retrieve({
      key: dev.key,
      schemas: LabJack.Device.SCHEMAS,
    });
    expect(updated.properties.writeStateIndex).not.toBe(0);
    expect(updated.properties.DO.channels.DIO4).toEqual({
      command: doCh.cmdChannel,
      state: doCh.stateChannel,
    });
    expect(updated.properties.AO.channels.DAC0).toEqual({
      command: aoCh.cmdChannel,
      state: aoCh.stateChannel,
    });
    const stateIndex = await client.channels.retrieve(
      updated.properties.writeStateIndex,
    );
    expect(stateIndex.name).toBe(`${identifier}_write_state_time`);
    expect(stateIndex.isIndex).toBe(true);
  });

  it("should honor custom command and state channel names", async () => {
    const dev = await createLabJackDevice(client);
    const cmdName = uniqueName("lj_cmd");
    const stateName = uniqueName("lj_state");
    const draft = await createDraft(
      client,
      createConfig(dev.key, [
        createDigitalWriteChannel("DIO4", {
          cmdChannelName: cmdName,
          stateChannelName: stateName,
        }),
      ]),
    );
    const { container } = await renderWrite({ client, taskKey: draft.key });
    const created = await deployAndAwaitTask(
      client,
      container,
      draft.key,
      LabJack.Task.WRITE_SCHEMAS,
    );
    const [ch] = created.config.channels;
    const cmd = await client.channels.retrieve(ch.cmdChannel);
    expect(cmd.name).toBe(cmdName);
    const cmdIndex = await client.channels.retrieve(cmd.index);
    expect(cmdIndex.name).toBe(`${cmdName}_time`);
    const state = await client.channels.retrieve(ch.stateChannel);
    expect(state.name).toBe(stateName);
  });

  it("should bind a new entry to the channels the device already maps", async () => {
    const dev = await createLabJackDevice(client);
    const config = createConfig(dev.key, [createDigitalWriteChannel("DIO4")]);
    const firstDraft = await createDraft(client, config);
    const first = await renderWrite({ client, taskKey: firstDraft.key });
    const firstTask = await deployAndAwaitTask(
      client,
      first.container,
      firstDraft.key,
      LabJack.Task.WRITE_SCHEMAS,
    );
    first.unmount();
    const cmd = await client.channels.retrieve(firstTask.config.channels[0].cmdChannel);
    const secondDraft = await createDraft(client, config);
    await renderWrite({ client, taskKey: secondDraft.key });
    await screen.findByText(cmd.name);
    await waitFor(async () => {
      const saved = await client.tasks.retrieve({
        key: secondDraft.key,
        schemas: LabJack.Task.WRITE_SCHEMAS,
      });
      expect(saved.config.channels[0].cmdChannel).toBe(cmd.key);
      expect(saved.config.channels[0].stateChannel).toBe(
        firstTask.config.channels[0].stateChannel,
      );
    });
  });

  it("should reuse existing command and state channels when redeployed", async () => {
    const dev = await createLabJackDevice(client);
    const config = createConfig(dev.key, [createDigitalWriteChannel("DIO4")]);
    const firstDraft = await createDraft(client, config);
    const first = await renderWrite({ client, taskKey: firstDraft.key });
    const firstTask = await deployAndAwaitTask(
      client,
      first.container,
      firstDraft.key,
      LabJack.Task.WRITE_SCHEMAS,
    );
    first.unmount();

    const secondDraft = await createDraft(client, config);
    const second = await renderWrite({ client, taskKey: secondDraft.key });
    const secondTask = await deployAndAwaitTask(
      client,
      second.container,
      secondDraft.key,
      LabJack.Task.WRITE_SCHEMAS,
    );
    expect(secondTask.config.channels[0].cmdChannel).toBe(
      firstTask.config.channels[0].cmdChannel,
    );
    expect(secondTask.config.channels[0].stateChannel).toBe(
      firstTask.config.channels[0].stateChannel,
    );
  });

  it("should reuse the channels a device maps with no state index", async () => {
    const stateIndex = await client.channels.create({
      name: uniqueName("lj_state_time"),
      dataType: "timestamp",
      isIndex: true,
    });
    const state = await client.channels.create({
      name: uniqueName("lj_state"),
      dataType: "uint8",
      index: stateIndex.key,
    });
    const cmdIndex = await client.channels.create({
      name: uniqueName("lj_cmd_time"),
      dataType: "timestamp",
      isIndex: true,
    });
    const cmd = await client.channels.create({
      name: uniqueName("lj_cmd"),
      dataType: "uint8",
      index: cmdIndex.key,
    });
    const dev = await createLabJackDevice(client, {
      properties: {
        DO: { channels: { DIO4: { command: cmd.key, state: state.key } } },
      },
    });
    const draft = await createDraft(
      client,
      createConfig(dev.key, [createDigitalWriteChannel("DIO4")]),
    );
    const { container } = await renderWrite({ client, taskKey: draft.key });
    const created = await deployAndAwaitTask(
      client,
      container,
      draft.key,
      LabJack.Task.WRITE_SCHEMAS,
    );
    expect(created.config.channels[0].cmdChannel).toBe(cmd.key);
    expect(created.config.channels[0].stateChannel).toBe(state.key);
    const updated = await client.devices.retrieve({
      key: dev.key,
      schemas: LabJack.Device.SCHEMAS,
    });
    expect(updated.properties.writeStateIndex).toBe(stateIndex.key);
  });

  it("should show the names deploying gives each channel in the list", async () => {
    const dev = await createLabJackDevice(client);
    const draft = await createDraft(
      client,
      createConfig(dev.key, [createDigitalWriteChannel("DIO4")]),
    );
    await renderWrite({ client, taskKey: draft.key });
    const { identifier } = dev.properties;
    expect(await screen.findByText(`${identifier}_DIO4_cmd`)).toBeTruthy();
    expect(screen.getByText(`${identifier}_DIO4_state`)).toBeTruthy();
    expect(screen.queryByText("No channel")).toBeNull();
  });

  it("should create the task with one channel", async () => {
    const tsk = await createWithHook(
      LabJack.Task.useCreateWrite,
      LabJack.Task.WRITE_SCHEMAS,
    );
    expect(tsk.config.channels).toHaveLength(1);
  });
});

describe("Write disabled channels", () => {
  it("should show no conflict for a port a disabled channel shares", async () => {
    const dev = await createLabJackDevice(client);
    const draft = await createDraft(
      client,
      createConfig(dev.key, [
        createAnalogWriteChannel("DAC0", { disabled: true }),
        createAnalogWriteChannel("DAC0"),
      ]),
    );
    await renderWrite({ client, taskKey: draft.key });
    const section = await screen.findByRole("group", { name: "Analog outputs" });
    const port = within(section).getByRole("button", { name: "DAC0" });
    expect(port.className).toContain("console-port-grid__cell--used");
    expect(port.className).not.toContain("console-port-grid__cell--conflict");
    expect(screen.queryByText("Another channel uses DAC0")).toBeNull();
  });

  it("should deploy and redeploy a disabled channel on a port another channel uses", async () => {
    const dev = await createLabJackDevice(client);
    const firstDraft = await createDraft(
      client,
      createConfig(dev.key, [
        createAnalogWriteChannel("DAC0", { disabled: true }),
        createAnalogWriteChannel("DAC0"),
      ]),
    );
    const first = await renderWrite({ client, taskKey: firstDraft.key });
    const { config } = await deployAndAwaitTask(
      client,
      first.container,
      firstDraft.key,
      LabJack.Task.WRITE_SCHEMAS,
    );
    expect(config.channels[1].cmdChannel).toBe(config.channels[0].cmdChannel);
    expect(config.channels[1].stateChannel).toBe(config.channels[0].stateChannel);
    first.unmount();
    const secondDraft = await createDraft(client, config);
    const second = await renderWrite({ client, taskKey: secondDraft.key });
    await deployAndAwaitTask(client, second.container, secondDraft.key);
  });
});
