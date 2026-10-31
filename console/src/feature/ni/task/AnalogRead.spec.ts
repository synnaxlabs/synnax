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
import { assert, describe, expect, it } from "vitest";

import { NI } from "@/feature/ni";
import {
  awaitStatusDescription,
  createHardware,
  createNIDevice,
  renderNITaskForm,
} from "@/feature/ni/task/testutil";
import {
  clickDeploy,
  commitFieldInput,
  createWithHook,
  deployAndAwaitTask,
  findChannelListItem,
  selectFromDropdown,
} from "@/platform/task/testutil";
import {
  findDialogTriggerByText,
  getIconButton,
  getInputTable,
  getLabeledInput,
  uniqueName,
} from "@/testutil";

const client = createTestClient();

type CreateChannelOverrides = Partial<NI.Task.AIChannel>;

const createChannel = (
  type: NI.Task.AIChannelType,
  port: number,
  overrides: CreateChannelOverrides = {},
): NI.Task.AIChannel =>
  ({
    ...NI.Task.createAIChannel(type),
    key: id.create(),
    port,
    device: "placeholder_device",
    ...overrides,
  }) as NI.Task.AIChannel;

// Drafts carry no key; the created row mints its own.
const ZERO_DRAFT: task.New<NI.Task.AnalogReadSchemas> = {
  name: "NI analog read task",
  type: NI.Task.ANALOG_READ_TYPE,
  config: NI.Task.ANALOG_READ_SCHEMAS.config.parse({}),
};

const createDraft = async (config: task.Payload<NI.Task.AnalogReadSchemas>["config"]) =>
  await client.tasks.create({ ...ZERO_DRAFT, config }, NI.Task.ANALOG_READ_SCHEMAS);

const renderAnalogRead = async (
  config: task.Payload<NI.Task.AnalogReadSchemas>["config"],
) => {
  const draft = await createDraft(config);
  const rendered = await renderNITaskForm(NI.Task.AnalogRead, {
    client,
    taskKey: draft.key,
  });
  return { ...rendered, draft };
};

describe("AnalogRead", () => {
  it("should seed a voltage channel bound to the device on the task row", async () => {
    const dev = await createNIDevice(client);
    await renderAnalogRead({
      ...NI.Task.ANALOG_READ_SCHEMAS.config.parse({}),
      channels: [createChannel("ai_voltage", 0, { device: dev.key })],
    });
    await waitFor(() =>
      expect(screen.getByText("Terminal configuration")).toBeTruthy(),
    );
    await findDialogTriggerByText(dev.name);
  });

  it("should show NA in the port cell of the portless built-in temperature channel", async () => {
    const dev = await createNIDevice(client);
    await renderAnalogRead({
      ...NI.Task.ANALOG_READ_SCHEMAS.config.parse({}),
      channels: [
        createChannel("ai_voltage", 3, { device: dev.key }),
        {
          ...NI.Task.createAIChannel("ai_temp_builtin"),
          key: id.create(),
          device: dev.key,
        },
      ],
    });
    await findChannelListItem("3");
    await findChannelListItem("NA");
  });

  it("should render the detail form for every channel type as it is selected", async () => {
    const cases: [NI.Task.AIChannelType, string][] = [
      ["ai_accel", "Current excitation source"],
      ["ai_accel_4_wire_dc_voltage", "Use excitation for scaling"],
      ["ai_accel_charge", "Acceleration units"],
      ["ai_bridge", "Nominal bridge resistance"],
      ["ai_charge", "Charge units"],
      ["ai_current", "Shunt resistor location"],
      ["ai_current_rms", "Shunt resistor location"],
      ["ai_force_bridge_polynomial", "Forward coefficients"],
      ["ai_force_bridge_table", "Force units"],
      ["ai_force_bridge_two_point_lin", "Physical value one"],
      ["ai_force_iepe", "Current excitation source"],
      ["ai_freq_voltage", "Threshold level"],
      ["ai_microphone", "Microphone sensitivity"],
      ["ai_pressure_bridge_polynomial", "Reverse coefficients"],
      ["ai_pressure_bridge_table", "Pressure units"],
      ["ai_pressure_bridge_two_point_lin", "Electrical value two"],
      ["ai_resistance", "Resistance configuration"],
      ["ai_rtd", "RTD type"],
      ["ai_strain_gauge", "Gage factor"],
      ["ai_temp_builtin", "Temperature units"],
      ["ai_thermistor_iex", "Steinhart-Hart A"],
      ["ai_thermistor_vex", "Reference resistor"],
      ["ai_thermocouple", "Thermocouple type"],
      ["ai_torque_bridge_polynomial", "Forward coefficients"],
      ["ai_torque_bridge_table", "Torque units"],
      ["ai_torque_bridge_two_point_lin", "Electrical value one"],
      ["ai_velocity_iepe", "Velocity units"],
      ["ai_voltage", "Terminal configuration"],
      ["ai_voltage_rms", "Terminal configuration"],
      ["ai_voltage_with_excit", "Use excitation for scaling"],
    ];
    // A generated type without a case here is a type the console cannot edit.
    expect(new Set(cases.map(([type]) => type))).toEqual(
      new Set(Object.keys(NI.Task.AI_CHANNEL_TYPE_NAMES)),
    );
    await renderAnalogRead({
      ...NI.Task.ANALOG_READ_SCHEMAS.config.parse({}),
      channels: cases.map(([type], i) =>
        createChannel(type, i, { name: `chan_${type}` }),
      ),
    });
    for (const [type, distinguishingLabel] of cases) {
      fireEvent.click(await findChannelListItem(`chan_${type}`));
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

  it("should hide the scale section for unscaled channel types", async () => {
    await renderAnalogRead({
      ...NI.Task.ANALOG_READ_SCHEMAS.config.parse({}),
      channels: [
        createChannel("ai_thermocouple", 0, { name: "chan_unscaled" }),
        createChannel("ai_voltage", 1, { name: "chan_scaled" }),
      ],
    });
    fireEvent.click(await screen.findByText("chan_scaled"));
    await waitFor(() => expect(screen.getByText("Custom scaling")).toBeTruthy());
    fireEvent.click(screen.getByText("chan_unscaled"));
    await waitFor(() => expect(screen.getByText("Thermocouple type")).toBeTruthy());
    expect(screen.queryByText("Custom scaling")).toBeNull();
  });

  it("should render the scale form matching each channel's custom scale", async () => {
    const cases = [
      ["linear", "Slope"],
      ["map", "Pre-scaled min"],
      ["polynomial", "Forward coefficients"],
      ["table", "Table CSV"],
    ] as const;
    await renderAnalogRead({
      ...NI.Task.ANALOG_READ_SCHEMAS.config.parse({}),
      channels: cases.map(([scaleType], i) => ({
        ...createChannel("ai_voltage", i, { name: `chan_${scaleType}` }),
        customScale: NI.Task.createScale(scaleType),
      })),
    });
    for (const [scaleType, distinguishingLabel] of cases) {
      fireEvent.click(await findChannelListItem(`chan_${scaleType}`));
      await waitFor(
        () =>
          expect(
            screen.getAllByText(distinguishingLabel).length,
            `scale form for ${scaleType} did not render "${distinguishingLabel}"`,
          ).toBeGreaterThan(0),
        { onTimeout: (e) => new Error(`${scaleType}: ${e.message}`) },
      );
    }
  });

  it("should show and seed the matching CJC field as the CJC source is switched", async () => {
    await renderAnalogRead({
      ...NI.Task.ANALOG_READ_SCHEMAS.config.parse({}),
      channels: [
        createChannel("ai_thermocouple", 0, {
          cjc: { source: "const_val", val: 5 },
        }),
      ],
    });
    await screen.findByText("CJC source");
    expect(getLabeledInput("CJC value").value).toBe("5");
    expect(screen.queryByText("CJC port")).toBeNull();
    await selectFromDropdown("Constant value", "Channel");
    await waitFor(() => expect(getLabeledInput("CJC port").value).toBe("0"));
    expect(screen.queryByText("CJC value")).toBeNull();
    // Switching back must re-seed the value to 0, not resurface the stale 5.
    await selectFromDropdown("Channel", "Constant value");
    await waitFor(() => expect(getLabeledInput("CJC value").value).toBe("0"));
    expect(screen.queryByText("CJC port")).toBeNull();
    await selectFromDropdown("Constant value", "Built in");
    await waitFor(() => expect(screen.queryByText("CJC value")).toBeNull());
    expect(screen.queryByText("CJC port")).toBeNull();
  });

  it("should swap the channel to the newly selected type and keep its port", async () => {
    await renderAnalogRead({
      ...NI.Task.ANALOG_READ_SCHEMAS.config.parse({}),
      channels: [createChannel("ai_voltage", 3)],
    });
    fireEvent.click(await findDialogTriggerByText("Voltage"));
    fireEvent.click(await screen.findByText("Thermocouple"));
    await waitFor(() => expect(screen.getByText("Thermocouple type")).toBeTruthy());
    expect(screen.queryByText("Terminal configuration")).toBeNull();
    const port = screen.getByDisplayValue("3");
    expect(port).toBeTruthy();
  });

  it("should drop the scale section on a swap to an unscaled type and restore it", async () => {
    await renderAnalogRead({
      ...NI.Task.ANALOG_READ_SCHEMAS.config.parse({}),
      channels: [createChannel("ai_voltage", 0)],
    });
    await screen.findByText("Custom scaling");
    fireEvent.click(await findDialogTriggerByText("Voltage"));
    fireEvent.click(await screen.findByText("Thermocouple"));
    await waitFor(() => expect(screen.queryByText("Custom scaling")).toBeNull());
    fireEvent.click(await findDialogTriggerByText("Thermocouple"));
    fireEvent.click(await screen.findByText("Voltage"));
    await screen.findByText("Custom scaling");
  });

  it("should create index and data channels, update the device, and save the task", async () => {
    const dev = await createNIDevice(client);
    const namedChannel = uniqueName("ai_named");
    const { container, draft } = await renderAnalogRead({
      ...NI.Task.ANALOG_READ_SCHEMAS.config.parse({}),
      channels: [
        createChannel("ai_voltage", 0, { device: dev.key }),
        createChannel("ai_current", 1, { device: dev.key, name: namedChannel }),
      ],
    });
    await deployAndAwaitTask(client, container, draft.key);
    const created = await client.tasks.retrieve({
      key: draft.key,
      schemas: NI.Task.ANALOG_READ_SCHEMAS,
    });
    expect(created.type).toBe(NI.Task.ANALOG_READ_TYPE);
    expect(created.rack).toBe(dev.rack);
    const [c0, c1] = created.config.channels;
    expect(c0.channel).not.toBe(0);
    expect(c1.channel).not.toBe(0);

    const identifier = dev.properties.identifier;
    const dataChannel = await client.channels.retrieve(c0.channel);
    expect(dataChannel.name).toBe(`${identifier}_ai_0`);
    const named = await client.channels.retrieve(c1.channel);
    expect(named.name).toBe(namedChannel);

    const updated = await client.devices.retrieve({
      key: dev.key,
      schemas: NI.Device.SCHEMAS,
    });
    expect(updated.properties.analogInput.index).not.toBe(0);
    expect(updated.properties.analogInput.channels["0"]).toBe(c0.channel);
    expect(updated.properties.analogInput.channels["1"]).toBe(c1.channel);
    const index = await client.channels.retrieve(updated.properties.analogInput.index);
    expect(index.name).toBe(`${identifier}_ai_time`);
    expect(index.isIndex).toBe(true);
  });

  it("should persist coefficients added and removed through the polynomial editor", async () => {
    const dev = await createNIDevice(client);
    const { container, draft } = await renderAnalogRead({
      ...NI.Task.ANALOG_READ_SCHEMAS.config.parse({}),
      channels: [
        createChannel("ai_force_bridge_polynomial", 0, {
          device: dev.key,
          forwardCoeffs: [1, 2, 3],
        }),
      ],
    });
    await screen.findByText("Forward coefficients");
    const forward = getInputTable("Forward coefficients");
    expect(forward.rows).toHaveLength(3);
    fireEvent.click(getIconButton(forward.rows[1], "close"));
    await waitFor(() =>
      expect(getInputTable("Forward coefficients").rows).toHaveLength(2),
    );
    fireEvent.click(getInputTable("Reverse coefficients").add);
    await waitFor(() =>
      expect(getInputTable("Reverse coefficients").rows).toHaveLength(1),
    );
    commitFieldInput(getInputTable("Reverse coefficients").cell(0), "7.5");

    await deployAndAwaitTask(client, container, draft.key);
    const created = await client.tasks.retrieve({
      key: draft.key,
      schemas: NI.Task.ANALOG_READ_SCHEMAS,
    });
    const [channel] = created.config.channels;
    assert(channel.type === "ai_force_bridge_polynomial");
    // The middle coefficient goes, not the last one.
    expect(channel.forwardCoeffs).toEqual([1, 3]);
    expect(channel.reverseCoeffs).toEqual([7.5]);
  });

  it("should bind a new entry to the channel the device already maps", async () => {
    const dev = await createNIDevice(client);
    const config = {
      ...NI.Task.ANALOG_READ_SCHEMAS.config.parse({}),
      channels: [createChannel("ai_voltage", 0, { device: dev.key })],
    };
    const first = await renderAnalogRead(config);
    await deployAndAwaitTask(client, first.container, first.draft.key);
    const firstTask = await client.tasks.retrieve({
      key: first.draft.key,
      schemas: NI.Task.ANALOG_READ_SCHEMAS,
    });
    first.unmount();
    const existing = await client.channels.retrieve(
      firstTask.config.channels[0].channel,
    );
    const second = await renderAnalogRead(config);
    await findChannelListItem(existing.name);
    await waitFor(async () => {
      const saved = await client.tasks.retrieve({
        key: second.draft.key,
        schemas: NI.Task.ANALOG_READ_SCHEMAS,
      });
      expect(saved.config.channels[0].channel).toBe(existing.key);
    });
  });

  it("should reuse existing channels when redeployed", async () => {
    const dev = await createNIDevice(client);
    const config = {
      ...NI.Task.ANALOG_READ_SCHEMAS.config.parse({}),
      channels: [createChannel("ai_voltage", 0, { device: dev.key })],
    };
    const first = await renderAnalogRead(config);
    await deployAndAwaitTask(client, first.container, first.draft.key);
    const firstTask = await client.tasks.retrieve({
      key: first.draft.key,
      schemas: NI.Task.ANALOG_READ_SCHEMAS,
    });
    first.unmount();
    const second = await renderAnalogRead(config);
    await deployAndAwaitTask(client, second.container, second.draft.key);
    await waitFor(async () => {
      const again = await client.tasks.retrieve({
        key: second.draft.key,
        schemas: NI.Task.ANALOG_READ_SCHEMAS,
      });
      expect(again.config.channels[0].channel).toBe(
        firstTask.config.channels[0].channel,
      );
    });
    const dvc = await client.devices.retrieve({
      key: dev.key,
      schemas: NI.Device.SCHEMAS,
    });
    expect(dvc.properties.analogInput.channels["0"]).toBe(
      firstTask.config.channels[0].channel,
    );
  });

  it("should surface an error when channels span devices on different racks", async () => {
    const devA = await createNIDevice(client);
    const devB = await createNIDevice(client);
    const { statuses, container } = await renderAnalogRead({
      ...NI.Task.ANALOG_READ_SCHEMAS.config.parse({}),
      channels: [
        createChannel("ai_voltage", 0, { device: devA.key }),
        createChannel("ai_voltage", 1, { device: devB.key }),
      ],
    });
    await clickDeploy(container);
    await awaitStatusDescription(statuses, /are on different racks/);
  });

  it("should mark the channel list when the task has no channels", async () => {
    const { container } = await renderAnalogRead({
      ...NI.Task.ANALOG_READ_SCHEMAS.config.parse({}),
      channels: [],
    });
    await clickDeploy(container);
    expect(
      await screen.findByText("At least one enabled channel is required"),
    ).toBeTruthy();
  });

  it("should mark the device field when a channel has no device", async () => {
    const { container } = await renderAnalogRead({
      ...NI.Task.ANALOG_READ_SCHEMAS.config.parse({}),
      channels: [createChannel("ai_voltage", 0, { device: "" })],
    });
    await clickDeploy(container);
    expect(await screen.findByText("Device is required")).toBeTruthy();
  });

  it("should create the task with one channel", async () => {
    const tsk = await createWithHook(
      NI.Task.useCreateAnalogRead,
      NI.Task.ANALOG_READ_SCHEMAS,
    );
    expect(tsk.config.channels).toHaveLength(1);
  });

  it("should create the task with one channel on the given device", async () => {
    const dev = await createNIDevice(client);
    const tsk = await createWithHook(
      NI.Task.useCreateAnalogRead,
      NI.Task.ANALOG_READ_SCHEMAS,
      { deviceKey: dev.key },
    );
    expect(tsk.config.channels.map(({ device }) => device)).toEqual([dev.key]);
  });
});

describe("AnalogRead ports", () => {
  const createDevice = async () =>
    await createNIDevice(client, { properties: { hardware: createHardware() } });

  const renderWith = async (channels: NI.Task.AIChannel[]) =>
    await renderAnalogRead({
      ...NI.Task.ANALOG_READ_SCHEMAS.config.parse({}),
      channels,
    });

  const savedPorts = async (key: string) => {
    const saved = await client.tasks.retrieve({
      key,
      schemas: NI.Task.ANALOG_READ_SCHEMAS,
    });
    return saved.config.channels.map((c) => ("port" in c ? c.port : null));
  };

  const cell = (grid: HTMLElement, name: string) =>
    within(grid).getByRole("button", { name });

  const openPorts = async (dev: NI.Device.Device) => {
    fireEvent.click(await screen.findByText("Ports"));
    return await screen.findByRole("group", { name: `${dev.name} ports` });
  };

  it.each<[string, (device: string) => NI.Task.AIChannel[], string]>([
    [
      "a port the device does not have",
      (device) => [createChannel("ai_voltage", 20, { device })],
      "The device has no port ai20",
    ],
    [
      "a type the port does not take",
      (device) => [createChannel("ai_thermocouple", 8, { device })],
      "ai8 does not take thermocouple channels",
    ],
    [
      "a terminal configuration the port does not take",
      (device) => [createChannel("ai_voltage", 8, { device, terminalConfig: "Diff" })],
      "ai8 does not take differential",
    ],
  ])("should mark %s", async (_, channels, message) => {
    const dev = await createDevice();
    await renderWith(channels(dev.key));
    expect(await screen.findByText(message)).toBeTruthy();
  });

  it("should mark a channel on the negative input of a differential channel", async () => {
    const dev = await createDevice();
    await renderWith([
      createChannel("ai_voltage", 0, { device: dev.key }),
      createChannel("ai_voltage", 8, {
        device: dev.key,
        terminalConfig: "RSE",
        name: "chan_rse",
      }),
    ]);
    expect(
      await screen.findByText("Another channel uses ai8, its negative input"),
    ).toBeTruthy();
    fireEvent.click(await findChannelListItem("chan_rse"));
    expect(
      await screen.findByText("ai8 is the negative input of another channel"),
    ).toBeTruthy();
  });

  it.each<[string, (device: string) => NI.Task.AIChannel[], number, string, string]>([
    [
      "an error above the single channel maximum",
      (device) => [createChannel("ai_voltage", 0, { device })],
      3_000_000,
      "samples at most 2,000,000 Hz",
      "error",
    ],
    [
      "a warning above the multiplexed maximum",
      (device) => [
        createChannel("ai_voltage", 0, { device }),
        createChannel("ai_voltage", 1, { device }),
      ],
      600_000,
      "may not sample 2 channels faster than 500,000 Hz",
      "warning",
    ],
    [
      "the extra scan channels of a thermocouple device",
      (device) => [createChannel("ai_thermocouple", 0, { device })],
      400_000,
      "may not sample 3 channels faster than 333,333 Hz",
      "warning",
    ],
  ])(
    "should mark the sample rate with %s",
    async (_, channels, rate, message, variant) => {
      const dev = await createDevice();
      await renderAnalogRead({
        ...NI.Task.ANALOG_READ_SCHEMAS.config.parse({}),
        sampleRate: rate,
        channels: channels(dev.key),
      });
      const text = await screen.findByText(`${dev.model} ${message}`);
      expect(text.className).toContain(variant);
    },
  );

  it("should deploy channels on free ports of the device", async () => {
    const dev = await createDevice();
    const { container, draft } = await renderWith([
      createChannel("ai_voltage", 0, { device: dev.key }),
      createChannel("ai_voltage", 1, { device: dev.key }),
    ]);
    await screen.findByText("Ports");
    await deployAndAwaitTask(client, container, draft.key);
  });

  it("should deploy a channel on any port of a device with no port list", async () => {
    const dev = await createNIDevice(client);
    const { container, draft } = await renderWith([
      createChannel("ai_voltage", 99, { device: dev.key }),
    ]);
    await deployAndAwaitTask(client, container, draft.key);
  });

  it("should deploy a channel on a port that does not report its capabilities", async () => {
    const hardware = createHardware();
    for (const port of [8, 9])
      hardware.analogInputs[port] = {
        ...hardware.analogInputs[port],
        terminalConfigs: [],
        types: [],
      };
    const dev = await createNIDevice(client, { properties: { hardware } });
    const { container, draft } = await renderWith([
      createChannel("ai_thermocouple", 8, { device: dev.key }),
      createChannel("ai_voltage", 9, { device: dev.key, terminalConfig: "Diff" }),
    ]);
    await deployAndAwaitTask(client, container, draft.key);
  });

  it("should put each negative input under its port, eight pairs to a group", async () => {
    const dev = await createDevice();
    await renderWith([
      createChannel("ai_voltage", 0, { device: dev.key, terminalConfig: "RSE" }),
    ]);
    const grid = await openPorts(dev);
    expect(grid.children).toHaveLength(1);
    expect(
      within(grid)
        .getAllByRole("button")
        .map((b) => [
          b.getAttribute("aria-label"),
          b.style.gridRow,
          b.style.gridColumn,
        ]),
    ).toEqual(
      Array.from({ length: 8 }, (_, i) => [
        [`ai${i}`, "1", String(i + 1)],
        [`ai${i + 8}`, "2", String(i + 1)],
      ]).flat(),
    );
  });

  it.each<
    [string, (device: string) => NI.Task.AIChannel[], string, string, boolean, string]
  >([
    [
      "span a differential channel across both ports of its pair",
      (device) => [createChannel("ai_voltage", 2, { device, name: "chan_a" })],
      "ai2",
      "used",
      true,
      "ai2 and ai10: chan_a",
    ],
    [
      "keep the pair apart for a channel that is not differential",
      (device) => [
        createChannel("ai_voltage", 2, {
          device,
          terminalConfig: "RSE",
          name: "chan_a",
        }),
      ],
      "ai10",
      "free",
      false,
      "ai10 is free",
    ],
    [
      "mark a differential pair as a conflict when another channel holds a port",
      (device) => [
        createChannel("ai_voltage", 8, { device, name: "chan_a" }),
        createChannel("ai_voltage", 0, { device, name: "chan_b" }),
      ],
      "ai0",
      "conflict",
      true,
      "ai0 and ai8 are used by chan_b and chan_a",
    ],
    [
      "mark a port two channels hold as a conflict",
      (device) => [
        createChannel("ai_voltage", 9, {
          device,
          terminalConfig: "RSE",
          name: "chan_a",
        }),
        createChannel("ai_voltage", 9, {
          device,
          terminalConfig: "RSE",
          name: "chan_b",
        }),
      ],
      "ai9",
      "conflict",
      false,
      "ai9 is used by chan_a and chan_b",
    ],
  ])("should %s", async (_, channels, name, state, paired, tooltip) => {
    const dev = await createDevice();
    await renderWith(channels(dev.key));
    const target = cell(await openPorts(dev), name);
    expect(target.className).toContain(`console-port-grid__cell--${state}`);
    expect(target.className.includes("console-port-grid__cell--paired")).toBe(paired);
    fireEvent.pointerOver(target, { pointerType: "mouse" });
    expect(await screen.findByText(tooltip)).toBeTruthy();
  });

  it("should add a channel on the next port whose negative input is free", async () => {
    const dev = await createDevice();
    const { container, draft } = await renderWith([
      createChannel("ai_voltage", 0, { device: dev.key }),
      createChannel("ai_voltage", 9, { device: dev.key }),
    ]);
    await screen.findByText("Ports");
    fireEvent.click(getIconButton(container, "add"));
    await waitFor(async () => expect(await savedPorts(draft.key)).toEqual([0, 9, 2]));
  });

  it("should add a channel past the device's ports once each pair holds one", async () => {
    const dev = await createDevice();
    const { container, draft } = await renderWith(
      Array.from({ length: 8 }, (_, port) =>
        createChannel("ai_voltage", port, { device: dev.key }),
      ),
    );
    await screen.findByText("Ports");
    fireEvent.click(getIconButton(container, "add"));
    await waitFor(async () =>
      expect(await savedPorts(draft.key)).toEqual([0, 1, 2, 3, 4, 5, 6, 7, 16]),
    );
  });

  it("should add a channel on the lowest free port of the selected channel's device", async () => {
    const devA = await createNIDevice(client);
    const devB = await createNIDevice(client);
    const { container, draft } = await renderWith([
      createChannel("ai_voltage", 0, { device: devA.key }),
      createChannel("ai_voltage", 1, { device: devA.key }),
      createChannel("ai_voltage", 0, { device: devB.key, name: "chan_b" }),
    ]);
    fireEvent.click(await findChannelListItem("chan_b"));
    fireEvent.click(getIconButton(container, "add"));
    await waitFor(async () => {
      const saved = await client.tasks.retrieve({
        key: draft.key,
        schemas: NI.Task.ANALOG_READ_SCHEMAS,
      });
      expect(
        saved.config.channels.map((c) => [c.device, "port" in c ? c.port : null]),
      ).toEqual([
        [devA.key, 0],
        [devA.key, 1],
        [devB.key, 0],
        [devB.key, 1],
      ]);
    });
  });

  it("should set the channel's port from a click on the grid", async () => {
    const dev = await createDevice();
    const { draft } = await renderWith([
      createChannel("ai_voltage", 0, { device: dev.key }),
    ]);
    const grid = await screen.findByRole("group", { name: "Port" });
    expect(cell(grid, "ai0").getAttribute("aria-pressed")).toBe("true");
    fireEvent.click(cell(grid, "ai3"));
    await waitFor(() =>
      expect(cell(grid, "ai3").getAttribute("aria-pressed")).toBe("true"),
    );
    await waitFor(async () => expect(await savedPorts(draft.key)).toEqual([3]));
  });

  it("should take a port number for a device with no port list", async () => {
    const dev = await createNIDevice(client);
    await renderWith([createChannel("ai_voltage", 7, { device: dev.key })]);
    await screen.findByText("Terminal configuration");
    expect(screen.queryByRole("group", { name: "Port" })).toBeNull();
    expect(screen.getByDisplayValue("7")).toBeTruthy();
  });

  it("should mark a channel that shares its port while the user edits", async () => {
    const dev = await createDevice();
    await renderWith([
      createChannel("ai_voltage", 0, { device: dev.key }),
      createChannel("ai_voltage", 0, { device: dev.key }),
    ]);
    expect(await screen.findByText("Another channel uses ai0")).toBeTruthy();
  });

  it("should list only the channel types the port takes", async () => {
    const dev = await createDevice();
    await renderWith([createChannel("ai_voltage", 8, { device: dev.key })]);
    fireEvent.click(await findDialogTriggerByText("Voltage"));
    await screen.findByText("Temperature built-in sensor");
    expect(screen.queryByText("Thermocouple")).toBeNull();
  });

  it("should list every type and terminal configuration on a port that does not report them", async () => {
    const hardware = createHardware();
    hardware.analogInputs[8] = {
      ...hardware.analogInputs[8],
      terminalConfigs: [],
      types: [],
    };
    const dev = await createNIDevice(client, { properties: { hardware } });
    await renderWith([createChannel("ai_voltage", 8, { device: dev.key })]);
    fireEvent.click(await findDialogTriggerByText("Voltage"));
    expect(await screen.findByText("Thermocouple")).toBeTruthy();
    fireEvent.click(await findDialogTriggerByText("Default (Referenced single ended)"));
    expect(await screen.findByText("Differential")).toBeTruthy();
  });

  it("should show the terminal configuration Default gives on the port", async () => {
    const dev = await createDevice();
    await renderWith([createChannel("ai_voltage", 8, { device: dev.key })]);
    expect(
      await findDialogTriggerByText("Default (Referenced single ended)"),
    ).toBeTruthy();
  });

  it("should resolve Default for a current channel on a port that takes only current", async () => {
    const hardware = createHardware();
    hardware.analogInputs[0] = {
      ...hardware.analogInputs[0],
      terminalConfigs: ["RSE", "Diff"],
      defaultTerminalConfig: "Diff",
      types: ["ai_current"],
    };
    const dev = await createNIDevice(client, { properties: { hardware } });
    await renderWith([
      createChannel("ai_current", 0, { device: dev.key }),
      createChannel("ai_voltage", 8, { device: dev.key }),
    ]);
    expect(await findDialogTriggerByText("Default (Differential)")).toBeTruthy();
    expect(
      await screen.findByText("Another channel uses ai8, its negative input"),
    ).toBeTruthy();
    expect(document.querySelector(".console-port-badge")?.className).toContain(
      "console-port-badge--conflict",
    );
  });

  it("should add a channel on a free port from the ports view", async () => {
    const dev = await createDevice();
    const { draft } = await renderWith([
      createChannel("ai_voltage", 0, { device: dev.key }),
    ]);
    fireEvent.click(await screen.findByText("Ports"));
    const grid = await screen.findByRole("group", { name: `${dev.name} ports` });
    fireEvent.click(cell(grid, "ai3"));
    await waitFor(async () => expect(await savedPorts(draft.key)).toEqual([0, 3]));
    expect(cell(grid, "ai3").getAttribute("aria-pressed")).toBe("true");
  });

  it("should add a channel on the clicked port when the first channel has no port", async () => {
    const dev = await createDevice();
    const { draft } = await renderWith([
      {
        ...NI.Task.createAIChannel("ai_temp_builtin"),
        key: id.create(),
        device: dev.key,
      },
    ]);
    fireEvent.click(await screen.findByText("Ports"));
    const grid = await screen.findByRole("group", { name: `${dev.name} ports` });
    fireEvent.click(cell(grid, "ai3"));
    await waitFor(async () => expect(await savedPorts(draft.key)).toEqual([null, 3]));
    expect(cell(grid, "ai3").getAttribute("aria-pressed")).toBe("true");
  });

  it("should select the channel on a used port from the ports view", async () => {
    const dev = await createDevice();
    await renderWith([
      createChannel("ai_voltage", 0, { device: dev.key, name: "chan_first" }),
      createChannel("ai_voltage", 1, { device: dev.key, name: "chan_second" }),
    ]);
    fireEvent.click(await screen.findByText("Ports"));
    const grid = await screen.findByRole("group", { name: `${dev.name} ports` });
    expect(cell(grid, "ai0").getAttribute("aria-pressed")).toBe("true");
    fireEvent.click(cell(grid, "ai1"));
    await waitFor(() =>
      expect(cell(grid, "ai1").getAttribute("aria-pressed")).toBe("true"),
    );
    expect(cell(grid, "ai0").getAttribute("aria-pressed")).toBe("false");
    expect(screen.getByText("chan_second")).toBeTruthy();
  });

  it("should offer no ports view when no device has a port list", async () => {
    const dev = await createNIDevice(client);
    await renderWith([createChannel("ai_voltage", 0, { device: dev.key })]);
    await screen.findByText("Terminal configuration");
    expect(screen.queryByText("Ports")).toBeNull();
  });
});

describe("AnalogRead disabled channels", () => {
  const createDevice = async () =>
    await createNIDevice(client, { properties: { hardware: createHardware() } });

  const renderWith = async (channels: NI.Task.AIChannel[]) =>
    await renderAnalogRead({
      ...NI.Task.ANALOG_READ_SCHEMAS.config.parse({}),
      channels,
    });

  const openPorts = async (dev: NI.Device.Device) => {
    fireEvent.click(await screen.findByText("Ports"));
    return await screen.findByRole("group", { name: `${dev.name} ports` });
  };

  it("should show a port a disabled channel holds as used and disabled", async () => {
    const dev = await createDevice();
    await renderWith([
      createChannel("ai_voltage", 3, {
        device: dev.key,
        disabled: true,
        name: "chan_off",
      }),
    ]);
    const badge = await waitFor(() => {
      const b = document.querySelector(".console-port-badge");
      assert(b != null);
      return b;
    });
    expect(badge.className).toContain("console-port-badge--disabled");
    const port = within(await openPorts(dev)).getByRole("button", { name: "ai3" });
    expect(port.className).toContain("console-port-grid__cell--used");
    expect(port.className).toContain("console-port-grid__cell--disabled");
    fireEvent.contextMenu(port);
    expect(await screen.findByText("Enable")).toBeTruthy();
  });

  it("should show no conflict for a port a disabled channel shares until it is enabled", async () => {
    const dev = await createDevice();
    await renderWith([
      createChannel("ai_voltage", 0, {
        device: dev.key,
        disabled: true,
        name: "chan_off",
      }),
      createChannel("ai_voltage", 0, { device: dev.key, name: "chan_on" }),
    ]);
    const grid = await openPorts(dev);
    const port = within(grid).getByRole("button", { name: "ai0" });
    expect(port.className).toContain("console-port-grid__cell--used");
    expect(port.className).not.toContain("console-port-grid__cell--conflict");
    expect(port.className).not.toContain("console-port-grid__cell--disabled");
    fireEvent.click(screen.getByText("List"));
    expect(screen.queryByText("Another channel uses ai0")).toBeNull();
    fireEvent.click(
      within(await findChannelListItem("chan_off")).getByLabelText("Enabled"),
    );
    expect(await screen.findByText("Another channel uses ai0")).toBeTruthy();
  });

  it("should deploy and redeploy a disabled channel on a port another channel uses", async () => {
    const dev = await createNIDevice(client);
    const first = await renderWith([
      createChannel("ai_voltage", 0, { device: dev.key, disabled: true }),
      createChannel("ai_voltage", 0, { device: dev.key }),
    ]);
    const { config } = await deployAndAwaitTask(
      client,
      first.container,
      first.draft.key,
      NI.Task.ANALOG_READ_SCHEMAS,
    );
    expect(config.channels[1].channel).toBe(config.channels[0].channel);
    first.unmount();
    const second = await renderAnalogRead(config);
    await deployAndAwaitTask(client, second.container, second.draft.key);
  });

  it("should name a new shared data channel after its enabled channel", async () => {
    const dev = await createNIDevice(client);
    const enabled = uniqueName("chan_on");
    const { container, draft } = await renderWith([
      createChannel("ai_voltage", 0, {
        device: dev.key,
        disabled: true,
        name: uniqueName("chan_off"),
      }),
      createChannel("ai_voltage", 0, { device: dev.key, name: enabled }),
    ]);
    const { config } = await deployAndAwaitTask(
      client,
      container,
      draft.key,
      NI.Task.ANALOG_READ_SCHEMAS,
    );
    const created = await client.channels.retrieve(config.channels[0].channel);
    expect(created.name).toBe(enabled);
  });

  it("should deploy a disabled channel on a port the device does not have", async () => {
    const dev = await createDevice();
    const { container, draft } = await renderWith([
      createChannel("ai_voltage", 0, { device: dev.key }),
      createChannel("ai_voltage", 20, { device: dev.key, disabled: true }),
    ]);
    await screen.findByText("Ports");
    await deployAndAwaitTask(client, container, draft.key);
  });

  it("should add a channel on the next port past one a disabled channel holds", async () => {
    const dev = await createDevice();
    const { container, draft } = await renderWith([
      createChannel("ai_voltage", 0, { device: dev.key, disabled: true }),
    ]);
    await screen.findByText("Ports");
    fireEvent.click(getIconButton(container, "add"));
    await waitFor(async () => {
      const saved = await client.tasks.retrieve({
        key: draft.key,
        schemas: NI.Task.ANALOG_READ_SCHEMAS,
      });
      expect(saved.config.channels.map((c) => ("port" in c ? c.port : null))).toEqual([
        0, 1,
      ]);
    });
  });
});

describe("AnalogRead channel names", () => {
  it("should show the name configuring gives each channel in the list", async () => {
    const dev = await createNIDevice(client);
    await renderAnalogRead({
      ...NI.Task.ANALOG_READ_SCHEMAS.config.parse({}),
      channels: [createChannel("ai_voltage", 2, { device: dev.key })],
    });
    const item = await findChannelListItem(`${dev.properties.identifier}_ai_2`);
    expect(within(item).queryByText("No channel")).toBeNull();
  });
});
