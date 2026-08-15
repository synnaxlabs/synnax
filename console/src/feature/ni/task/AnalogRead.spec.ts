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
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { assert, describe, expect, it } from "vitest";

import { NI } from "@/feature/ni";
import {
  awaitStatusDescription,
  createNIDevice,
  getCoefficientInput,
  getCoefficientsField,
  renderNITaskForm,
} from "@/feature/ni/task/testutil";
import {
  clickDeploy,
  commitFieldInput,
  deployAndAwaitTask,
  findDialogTriggerByText,
  getLabeledInput,
  selectFromDropdown,
} from "@/platform/task/testutil";
import { getIconButton, uniqueName } from "@/testutil";

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
  name: "NI Analog Read Task",
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
      expect(screen.getByText("Terminal Configuration")).toBeTruthy(),
    );
    await findDialogTriggerByText(dev.name);
  });

  it("should render the detail form for every channel type as it is selected", async () => {
    const cases: [NI.Task.AIChannelType, string][] = [
      ["ai_accel", "Current Excitation Source"],
      ["ai_accel_4_wire_dc_voltage", "Use Excitation for Scaling"],
      ["ai_accel_charge", "Acceleration Units"],
      ["ai_bridge", "Nominal Bridge Resistance"],
      ["ai_charge", "Charge Units"],
      ["ai_current", "Shunt Resistor Location"],
      ["ai_current_rms", "Shunt Resistor Location"],
      ["ai_force_bridge_polynomial", "Forward Coefficients"],
      ["ai_force_bridge_table", "Force Units"],
      ["ai_force_bridge_two_point_lin", "Physical Value One"],
      ["ai_force_iepe", "Current Excitation Source"],
      ["ai_freq_voltage", "Threshold Level"],
      ["ai_microphone", "Microphone Sensitivity"],
      ["ai_pressure_bridge_polynomial", "Reverse Coefficients"],
      ["ai_pressure_bridge_table", "Pressure Units"],
      ["ai_pressure_bridge_two_point_lin", "Electrical Value Two"],
      ["ai_resistance", "Resistance Configuration"],
      ["ai_rtd", "RTD Type"],
      ["ai_strain_gauge", "Gage Factor"],
      ["ai_temp_builtin", "Temperature Units"],
      ["ai_thermistor_iex", "Steinhart-Hart A"],
      ["ai_thermistor_vex", "Reference Resistor"],
      ["ai_thermocouple", "Thermocouple Type"],
      ["ai_torque_bridge_polynomial", "Forward Coefficients"],
      ["ai_torque_bridge_table", "Torque Units"],
      ["ai_torque_bridge_two_point_lin", "Electrical Value One"],
      ["ai_velocity_iepe", "Velocity Units"],
      ["ai_voltage", "Terminal Configuration"],
      ["ai_voltage_rms", "Terminal Configuration"],
      ["ai_voltage_with_excit", "Use Excitation for Scaling"],
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
      fireEvent.click(await screen.findByText(`chan_${type}`));
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

  it("should render the scale form matching each channel's custom scale", async () => {
    const cases = [
      ["linear", "Slope"],
      ["map", "Pre-Scaled Min"],
      ["polynomial", "Forward Coefficients"],
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
      fireEvent.click(await screen.findByText(`chan_${scaleType}`));
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
    await screen.findByText("CJC Source");
    expect(getLabeledInput("CJC Value").value).toBe("5");
    expect(screen.queryByText("CJC Port")).toBeNull();
    await selectFromDropdown("Constant Value", "Channel");
    await waitFor(() => expect(getLabeledInput("CJC Port").value).toBe("0"));
    expect(screen.queryByText("CJC Value")).toBeNull();
    // Switching back must re-seed the value to 0, not resurface the stale 5.
    await selectFromDropdown("Channel", "Constant Value");
    await waitFor(() => expect(getLabeledInput("CJC Value").value).toBe("0"));
    expect(screen.queryByText("CJC Port")).toBeNull();
    await selectFromDropdown("Constant Value", "Built In");
    await waitFor(() => expect(screen.queryByText("CJC Value")).toBeNull());
    expect(screen.queryByText("CJC Port")).toBeNull();
  });

  it("should swap the channel to the newly selected type and keep its port", async () => {
    await renderAnalogRead({
      ...NI.Task.ANALOG_READ_SCHEMAS.config.parse({}),
      channels: [createChannel("ai_voltage", 3)],
    });
    fireEvent.click(await findDialogTriggerByText("Voltage"));
    fireEvent.click(await screen.findByText("Thermocouple"));
    await waitFor(() => expect(screen.getByText("Thermocouple Type")).toBeTruthy());
    expect(screen.queryByText("Terminal Configuration")).toBeNull();
    const port = screen.getByDisplayValue("3");
    expect(port).toBeTruthy();
  });

  describe("deploying against a live cluster", () => {
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
      const index = await client.channels.retrieve(
        updated.properties.analogInput.index,
      );
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
      await screen.findByText("Forward Coefficients");
      const forward = getCoefficientsField("Forward Coefficients");
      expect(forward.rows).toHaveLength(3);
      fireEvent.click(getIconButton(forward.rows[1], "close"));
      await waitFor(() =>
        expect(getCoefficientsField("Forward Coefficients").rows).toHaveLength(2),
      );
      fireEvent.click(getCoefficientsField("Reverse Coefficients").add);
      await waitFor(() =>
        expect(getCoefficientsField("Reverse Coefficients").rows).toHaveLength(1),
      );
      const [reverseRow] = getCoefficientsField("Reverse Coefficients").rows;
      commitFieldInput(getCoefficientInput(reverseRow), "7.5");

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

    it("should surface an error when the task has no channels", async () => {
      const { statuses, container } = await renderAnalogRead({
        ...NI.Task.ANALOG_READ_SCHEMAS.config.parse({}),
        channels: [],
      });
      await clickDeploy(container);
      await awaitStatusDescription(
        statuses,
        /No devices selected in task configuration/,
      );
    });
  });
});
