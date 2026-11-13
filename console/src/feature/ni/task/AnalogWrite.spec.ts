// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type task } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { id } from "@synnaxlabs/x";
import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { NI } from "@/feature/ni";
import {
  createHardware,
  createNIDevice,
  renderNITaskForm,
} from "@/feature/ni/task/testutil";
import {
  createWithHook,
  deployAndAwaitTask,
  findChannelListItem,
  selectFromDropdown,
} from "@/platform/task/testutil";
import {
  findDialogTriggerByText,
  isSelectButtonSelected,
  uniqueName,
} from "@/testutil";

const client = createTestClient();

type CreateChannelOverrides = Partial<NI.Task.AOChannel>;

const createChannel = (
  type: NI.Task.AOChannelType,
  port: number,
  overrides: CreateChannelOverrides = {},
): NI.Task.AOChannel =>
  ({
    ...NI.Task.createAOChannel(type),
    key: id.create(),
    port,
    ...(type === "ao_func_gen" ? null : { cmdChannelName: `cmd_${type}_${port}` }),
    ...overrides,
  }) as NI.Task.AOChannel;

// Drafts carry no key; the created row mints its own.
const ZERO_DRAFT: task.New<NI.Task.AnalogWriteSchemas> = {
  name: "NI analog write task",
  type: NI.Task.ANALOG_WRITE_TYPE,
  config: NI.Task.ANALOG_WRITE_SCHEMAS.config.parse({}),
};

const createDraft = async (
  config: task.Payload<NI.Task.AnalogWriteSchemas>["config"],
) => await client.tasks.create({ ...ZERO_DRAFT, config }, NI.Task.ANALOG_WRITE_SCHEMAS);

const renderAnalogWrite = async (
  config: task.Payload<NI.Task.AnalogWriteSchemas>["config"],
) => {
  const draft = await createDraft(config);
  const rendered = await renderNITaskForm(NI.Task.AnalogWrite, {
    client,
    taskKey: draft.key,
  });
  return { ...rendered, draft };
};

const commanded = (channels: NI.Task.AOChannel[]) =>
  channels.filter((c) => !NI.Task.isGenerated(c));

const createConfig = (channels: NI.Task.AOChannel[], device = "") => ({
  ...NI.Task.ANALOG_WRITE_SCHEMAS.config.parse({}),
  device,
  channels,
});

describe("AnalogWrite", () => {
  it("should render the detail form for every channel type as it is selected", async () => {
    const cases: [NI.Task.AOChannelType, string][] = [
      ["ao_current", "Range"],
      ["ao_func_gen", "Frequency"],
      ["ao_voltage", "Custom scaling"],
    ];
    await renderAnalogWrite(
      createConfig(cases.map(([type], i) => createChannel(type, i))),
    );
    for (const [i, [type, distinguishingLabel]] of cases.entries()) {
      fireEvent.click(await findChannelListItem(String(i)));
      await waitFor(
        () =>
          expect(
            screen.getAllByText(distinguishingLabel).length,
            `detail form for ${type} did not render "${distinguishingLabel}"`,
          ).toBeGreaterThan(0),
        { onTimeout: (e) => new Error(`${type}: ${e.message}`) },
      );
    }
  });

  it("should name each end of a range", async () => {
    await renderAnalogWrite(createConfig([createChannel("ao_current", 0)]));
    fireEvent.click(await findChannelListItem("cmd_ao_current_0"));
    expect(await screen.findByRole("textbox", { name: "Minimum value" })).toBeTruthy();
    expect(screen.getByRole("textbox", { name: "Maximum value" })).toBeTruthy();
  });

  it("should switch the function generator wave type when a wave button is clicked", async () => {
    await renderAnalogWrite(createConfig([createChannel("ao_func_gen", 0)]));
    fireEvent.click(await findChannelListItem("0"));
    await screen.findByText("Triangle");
    await waitFor(() => expect(isSelectButtonSelected("Sine")).toBe(true));
    expect(isSelectButtonSelected("Triangle")).toBe(false);
    fireEvent.click(screen.getByText("Triangle"));
    await waitFor(() => expect(isSelectButtonSelected("Triangle")).toBe(true));
    expect(isSelectButtonSelected("Sine")).toBe(false);
  });

  it("should swap the channel to the newly selected type", async () => {
    await renderAnalogWrite(createConfig([createChannel("ao_voltage", 2)]));
    fireEvent.click(await findChannelListItem("cmd_ao_voltage_2"));
    await screen.findByText("Custom scaling");
    await selectFromDropdown("Voltage", "Function generator");
    await waitFor(() => expect(screen.getByText("Frequency")).toBeTruthy());
    expect(screen.queryByText("Custom scaling")).toBeNull();
    await selectFromDropdown("Function generator", "Voltage");
    await screen.findByText("Custom scaling");
  });

  it("should create command and state channels and update the device", async () => {
    const dev = await createNIDevice(client);
    const rendered = await renderAnalogWrite(
      createConfig(
        [
          createChannel("ao_voltage", 0, {
            cmdChannelName: "",
            stateChannelName: "",
          }),
        ],
        dev.key,
      ),
    );
    await deployAndAwaitTask(client, rendered.container, rendered.draft.key);
    const created = await client.tasks.retrieve({
      key: rendered.draft.key,
      schemas: NI.Task.ANALOG_WRITE_SCHEMAS,
    });
    expect(created.type).toBe(NI.Task.ANALOG_WRITE_TYPE);
    expect(created.rack).toBe(dev.rack);
    const [c0] = commanded(created.config.channels);
    expect(c0.cmdChannel).not.toBe(0);
    expect(c0.stateChannel).not.toBe(0);

    const identifier = dev.properties.identifier;
    const cmd = await client.channels.retrieve(c0.cmdChannel);
    expect(cmd.name).toBe(`${identifier}_ao_0_cmd`);
    const state = await client.channels.retrieve(c0.stateChannel);
    expect(state.name).toBe(`${identifier}_ao_0_state`);
    const cmdIndex = await client.channels.retrieve(cmd.index);
    expect(cmdIndex.name).toBe(`${identifier}_ao_0_cmd_time`);
    expect(cmdIndex.isIndex).toBe(true);

    const updated = await client.devices.retrieve({
      key: dev.key,
      schemas: NI.Device.SCHEMAS,
    });
    expect(updated.properties.analogOutput.stateIndex).not.toBe(0);
    expect(updated.properties.analogOutput.channels["0"]).toEqual({
      command: c0.cmdChannel,
      state: c0.stateChannel,
    });
    const stateIndex = await client.channels.retrieve(
      updated.properties.analogOutput.stateIndex,
    );
    expect(stateIndex.name).toBe(`${identifier}_ao_state_time`);
  });

  it("should create no channels for a task of function generators", async () => {
    const dev = await createNIDevice(client);
    const rendered = await renderAnalogWrite(
      createConfig([createChannel("ao_func_gen", 0)], dev.key),
    );
    await deployAndAwaitTask(client, rendered.container, rendered.draft.key);
    const created = await client.tasks.retrieve({
      key: rendered.draft.key,
      schemas: NI.Task.ANALOG_WRITE_SCHEMAS,
    });
    expect(created.config.channels[0]).not.toHaveProperty("cmdChannel");
    const updated = await client.devices.retrieve({
      key: dev.key,
      schemas: NI.Device.SCHEMAS,
    });
    expect(updated.properties.analogOutput.stateIndex).toBe(0);
    expect(updated.properties.analogOutput.channels).toEqual({});
  });

  it("should create channels only for the commanded channels of a mixed task", async () => {
    const dev = await createNIDevice(client);
    const rendered = await renderAnalogWrite(
      createConfig(
        [
          createChannel("ao_func_gen", 0),
          createChannel("ao_voltage", 1, { cmdChannelName: "" }),
        ],
        dev.key,
      ),
    );
    await deployAndAwaitTask(client, rendered.container, rendered.draft.key);
    const created = await client.tasks.retrieve({
      key: rendered.draft.key,
      schemas: NI.Task.ANALOG_WRITE_SCHEMAS,
    });
    const [generator, voltage] = created.config.channels;
    expect(generator).not.toHaveProperty("cmdChannel");
    expect(voltage).not.toHaveProperty("cmdChannel", 0);
    expect(voltage).not.toHaveProperty("stateChannel", 0);
    const updated = await client.devices.retrieve({
      key: dev.key,
      schemas: NI.Device.SCHEMAS,
    });
    expect(Object.keys(updated.properties.analogOutput.channels)).toEqual(["1"]);
  });

  it("should use custom command and state channel names when provided", async () => {
    const dev = await createNIDevice(client);
    const cmdName = uniqueName("ao_cmd");
    const stateName = uniqueName("ao_state");
    const rendered = await renderAnalogWrite(
      createConfig(
        [
          createChannel("ao_voltage", 0, {
            cmdChannelName: cmdName,
            stateChannelName: stateName,
          }),
        ],
        dev.key,
      ),
    );
    await deployAndAwaitTask(client, rendered.container, rendered.draft.key);
    const created = await client.tasks.retrieve({
      key: rendered.draft.key,
      schemas: NI.Task.ANALOG_WRITE_SCHEMAS,
    });
    const [c0] = commanded(created.config.channels);
    const cmd = await client.channels.retrieve(c0.cmdChannel);
    expect(cmd.name).toBe(cmdName);
    const state = await client.channels.retrieve(c0.stateChannel);
    expect(state.name).toBe(stateName);
    const cmdIndex = await client.channels.retrieve(cmd.index);
    expect(cmdIndex.name).toBe(`${cmdName}_time`);
  });

  it("should bind a new entry to the channel the device already maps", async () => {
    const dev = await createNIDevice(client);
    const config = createConfig(
      [createChannel("ao_voltage", 0, { cmdChannelName: "", stateChannelName: "" })],
      dev.key,
    );
    const first = await renderAnalogWrite(config);
    await deployAndAwaitTask(client, first.container, first.draft.key);
    const firstTask = await client.tasks.retrieve({
      key: first.draft.key,
      schemas: NI.Task.ANALOG_WRITE_SCHEMAS,
    });
    first.unmount();
    const [bound] = commanded(firstTask.config.channels);
    const cmd = await client.channels.retrieve(bound.cmdChannel);
    const second = await renderAnalogWrite(config);
    await findChannelListItem(cmd.name);
    await waitFor(async () => {
      const saved = await client.tasks.retrieve({
        key: second.draft.key,
        schemas: NI.Task.ANALOG_WRITE_SCHEMAS,
      });
      const [rebound] = commanded(saved.config.channels);
      expect(rebound.cmdChannel).toBe(bound.cmdChannel);
      expect(rebound.stateChannel).toBe(bound.stateChannel);
    });
  });

  it("should reuse existing channels when redeployed", async () => {
    const dev = await createNIDevice(client);
    const config = createConfig(
      [
        createChannel("ao_voltage", 0, {
          cmdChannelName: "",
          stateChannelName: "",
        }),
      ],
      dev.key,
    );
    const first = await renderAnalogWrite(config);
    await deployAndAwaitTask(client, first.container, first.draft.key);
    const firstTask = await client.tasks.retrieve({
      key: first.draft.key,
      schemas: NI.Task.ANALOG_WRITE_SCHEMAS,
    });
    first.unmount();
    const second = await renderAnalogWrite(config);
    await deployAndAwaitTask(client, second.container, second.draft.key);
    await waitFor(async () => {
      const again = await client.tasks.retrieve({
        key: second.draft.key,
        schemas: NI.Task.ANALOG_WRITE_SCHEMAS,
      });
      const [reused] = commanded(again.config.channels);
      const [original] = commanded(firstTask.config.channels);
      expect(reused.cmdChannel).toBe(original.cmdChannel);
      expect(reused.stateChannel).toBe(original.stateChannel);
    });
  });

  it("should create the task with one channel", async () => {
    const tsk = await createWithHook(
      NI.Task.useCreateAnalogWrite,
      NI.Task.ANALOG_WRITE_SCHEMAS,
    );
    expect(tsk.config.channels).toHaveLength(1);
  });
});

describe("AnalogWrite channel names", () => {
  it("should show the names deploying gives each channel in the list", async () => {
    const dev = await createNIDevice(client);
    const { identifier } = dev.properties;
    await renderAnalogWrite(
      createConfig([createChannel("ao_voltage", 1, { cmdChannelName: "" })], dev.key),
    );
    const item = await findChannelListItem(`${identifier}_ao_1_cmd`);
    expect(within(item).getByText(`${identifier}_ao_1_state`)).toBeTruthy();
  });
});

describe("AnalogWrite function generators", () => {
  it("should show no channel names in the list or the details", async () => {
    const dev = await createNIDevice(client);
    const { identifier } = dev.properties;
    await renderAnalogWrite(
      createConfig(
        [
          createChannel("ao_func_gen", 0),
          createChannel("ao_voltage", 1, { cmdChannelName: "" }),
        ],
        dev.key,
      ),
    );
    await findChannelListItem(`${identifier}_ao_1_cmd`);
    expect(screen.queryByText(`${identifier}_ao_0_cmd`)).toBeNull();
    expect(screen.queryByText(`${identifier}_ao_0_state`)).toBeNull();
  });

  it("should offer no channel renames in the context menu", async () => {
    await renderAnalogWrite(
      createConfig([
        createChannel("ao_func_gen", 0),
        createChannel("ao_voltage", 1, { cmdChannelName: "named_cmd" }),
      ]),
    );
    fireEvent.contextMenu(await findChannelListItem("named_cmd"));
    await screen.findByText("Rename command channel");
    fireEvent.contextMenu(await findChannelListItem("0"));
    await waitFor(() =>
      expect(screen.queryByText("Rename command channel")).toBeNull(),
    );
  });

  it("should not skip a disabled channel on the port of a function generator", async () => {
    await renderAnalogWrite(
      createConfig([
        createChannel("ao_func_gen", 0),
        createChannel("ao_voltage", 0, { cmdChannelName: "named_cmd", disabled: true }),
      ]),
    );
    await findChannelListItem("named_cmd");
    expect(screen.getAllByRole("img", { name: "New channel" })).not.toHaveLength(0);
  });

  it("should name a function generator by its type in the ports", async () => {
    const dev = await createNIDevice(client, {
      properties: { hardware: createHardware() },
    });
    const { identifier } = dev.properties;
    await renderAnalogWrite(
      createConfig(
        [
          createChannel("ao_func_gen", 0),
          createChannel("ao_voltage", 0, { cmdChannelName: "" }),
        ],
        dev.key,
      ),
    );
    fireEvent.click(await screen.findByText("Ports"));
    const grid = await screen.findByRole("group", { name: `${dev.name} ports` });
    const cell = within(grid).getByRole("button", { name: "ao0" });
    fireEvent.pointerOver(cell, { pointerType: "mouse" });
    expect(
      await screen.findByText(
        `ao0 is used by Function generator and ${identifier}_ao_0_cmd`,
      ),
    ).toBeTruthy();
  });

  it("should clear the channel names when the type becomes function generator", async () => {
    await renderAnalogWrite(
      createConfig([createChannel("ao_voltage", 2, { cmdChannelName: "named_cmd" })]),
    );
    fireEvent.click(await findChannelListItem("named_cmd"));
    await screen.findByText("Custom scaling");
    await selectFromDropdown("Voltage", "Function generator");
    await waitFor(() => expect(screen.queryByText("named_cmd")).toBeNull());
    await selectFromDropdown("Function generator", "Voltage");
    await screen.findByText("Custom scaling");
    expect(screen.queryByText("named_cmd")).toBeNull();
  });
});

describe("AnalogWrite ports", () => {
  it.each([
    ["ao_voltage", 20],
    ["ao_current", 0.5],
  ] as const)("should mark a %s channel outside every range", async (type, maxVal) => {
    const dev = await createNIDevice(client, {
      properties: { hardware: createHardware() },
    });
    await renderAnalogWrite(
      createConfig([createChannel(type, 0, { minVal: 0, maxVal })], dev.key),
    );
    await waitFor(() =>
      expect(screen.getByLabelText("Maximum value").getAttribute("aria-invalid")).toBe(
        "true",
      ),
    );
  });

  it("should give a channel switched to current the device's current range", async () => {
    const hardware = createHardware();
    hardware.analogOutputs[0].types.push("ao_current");
    const dev = await createNIDevice(client, { properties: { hardware } });
    await renderAnalogWrite(createConfig([createChannel("ao_voltage", 0)], dev.key));
    fireEvent.click(await findChannelListItem("cmd_ao_voltage_0"));
    await selectFromDropdown("Voltage", "Current");
    const max = await screen.findByLabelText<HTMLInputElement>("Maximum value");
    await waitFor(() => expect(max.value).toBe("0.02"));
    expect(screen.getByLabelText<HTMLInputElement>("Minimum value").value).toBe("0");
    expect(max.getAttribute("aria-invalid")).not.toBe("true");
  });

  it("should keep the range of a scaled channel switched to current", async () => {
    const hardware = createHardware();
    hardware.analogOutputs[0].types.push("ao_current");
    const dev = await createNIDevice(client, { properties: { hardware } });
    const scaled = createChannel("ao_voltage", 0, {
      minVal: 0,
      maxVal: 100,
      customScale: {
        type: "linear",
        slope: 10,
        yIntercept: 0,
        preScaledUnits: "Volts",
        scaledUnits: "Volts",
      },
    });
    await renderAnalogWrite(createConfig([scaled], dev.key));
    fireEvent.click(await findChannelListItem("cmd_ao_voltage_0"));
    await selectFromDropdown("Voltage", "Current");
    await findDialogTriggerByText("Current");
    expect(screen.getByLabelText<HTMLInputElement>("Minimum value").value).toBe("0");
    expect(screen.getByLabelText<HTMLInputElement>("Maximum value").value).toBe("100");
  });

  it("should deploy a channel on a port that does not report its types", async () => {
    const hardware = createHardware();
    hardware.analogOutputs[0] = { ...hardware.analogOutputs[0], types: [] };
    const dev = await createNIDevice(client, { properties: { hardware } });
    const rendered = await renderAnalogWrite(
      createConfig(
        [createChannel("ao_voltage", 0, { cmdChannelName: "", stateChannelName: "" })],
        dev.key,
      ),
    );
    await deployAndAwaitTask(client, rendered.container, rendered.draft.key);
  });
});

describe("AnalogWrite disabled channels", () => {
  it("should deploy and redeploy a disabled channel on a port another channel uses", async () => {
    const dev = await createNIDevice(client);
    const first = await renderAnalogWrite(
      createConfig(
        [
          createChannel("ao_voltage", 0, { cmdChannelName: "", disabled: true }),
          createChannel("ao_voltage", 0, { cmdChannelName: "" }),
        ],
        dev.key,
      ),
    );
    const { config } = await deployAndAwaitTask(
      client,
      first.container,
      first.draft.key,
      NI.Task.ANALOG_WRITE_SCHEMAS,
    );
    const [disabled, enabled] = commanded(config.channels);
    expect(enabled.cmdChannel).toBe(disabled.cmdChannel);
    expect(enabled.stateChannel).toBe(disabled.stateChannel);
    first.unmount();
    const second = await renderAnalogWrite(config);
    await deployAndAwaitTask(client, second.container, second.draft.key);
  });
});
