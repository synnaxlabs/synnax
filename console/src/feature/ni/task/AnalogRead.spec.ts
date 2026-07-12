// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { task } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { id } from "@synnaxlabs/x";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { NI } from "@/feature/ni";
import {
  awaitStatusDescription,
  createNIDevice,
  renderNITaskForm,
} from "@/feature/ni/task/testutil";
import {
  awaitTaskKey,
  clickConfigure,
  findDialogTriggerByText,
} from "@/platform/task/testutil";
import { stubGeometry, uniqueName } from "@/testutil";

stubGeometry();

const client = createTestClient();

type CreateChannelOverrides = Partial<NI.Task.AIChannel>;

const createChannel = (
  type: NI.Task.AIChannelType,
  port: number,
  overrides: CreateChannelOverrides = {},
): NI.Task.AIChannel =>
  ({
    ...NI.Task.ZERO_AI_CHANNELS[type],
    key: id.create(),
    port,
    device: "placeholder_device",
    ...overrides,
  }) as NI.Task.AIChannel;

const renderAnalogRead = async (args = {}) =>
  await renderNITaskForm(NI.Task.AnalogRead, NI.Task.ANALOG_READ_TYPE, {
    client,
    args,
  });

describe("AnalogRead", () => {
  it("should seed a voltage channel bound to the device passed through view args", async () => {
    const dev = await createNIDevice(client);
    await renderAnalogRead({ deviceKey: dev.key });
    await waitFor(() =>
      expect(screen.getByText("Terminal Configuration")).toBeTruthy(),
    );
    await findDialogTriggerByText(dev.name);
  });

  it("should render the detail form for every channel type as it is selected", async () => {
    const cases: [NI.Task.AIChannelType, string][] = [
      ["ai_accel", "Current Excitation Source"],
      ["ai_bridge", "Nominal Bridge Resistance"],
      ["ai_current", "Shunt Resistor Location"],
      ["ai_force_bridge_table", "Force Units"],
      ["ai_force_bridge_two_point_lin", "Physical Value One"],
      ["ai_force_iepe", "Current Excitation Source"],
      ["ai_microphone", "Microphone Sensitivity"],
      ["ai_pressure_bridge_table", "Pressure Units"],
      ["ai_pressure_bridge_two_point_lin", "Electrical Value Two"],
      ["ai_resistance", "Resistance Configuration"],
      ["ai_rtd", "RTD Type"],
      ["ai_strain_gauge", "Gage Factor"],
      ["ai_temp_builtin", "Temperature Units"],
      ["ai_thermocouple", "Thermocouple Type"],
      ["ai_torque_bridge_table", "Torque Units"],
      ["ai_torque_bridge_two_point_lin", "Electrical Value One"],
      ["ai_velocity_iepe", "Velocity Units"],
      ["ai_voltage", "Terminal Configuration"],
    ];
    await renderAnalogRead({
      config: {
        ...NI.Task.ZERO_ANALOG_READ_PAYLOAD.config,
        channels: cases.map(([type], i) =>
          createChannel(type, i, { name: `chan_${type}` }),
        ),
      },
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
      ["table", "Table CSV"],
    ] as const;
    await renderAnalogRead({
      config: {
        ...NI.Task.ZERO_ANALOG_READ_PAYLOAD.config,
        channels: cases.map(([scaleType], i) => ({
          ...createChannel("ai_voltage", i, { name: `chan_${scaleType}` }),
          customScale: NI.Task.ZERO_SCALES[scaleType],
        })),
      },
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

  it("should swap the channel to the newly selected type and keep its port", async () => {
    await renderAnalogRead({
      config: {
        ...NI.Task.ZERO_ANALOG_READ_PAYLOAD.config,
        channels: [createChannel("ai_voltage", 3)],
      },
    });
    fireEvent.click(await findDialogTriggerByText("Voltage"));
    fireEvent.click(await screen.findByText("Thermocouple"));
    await waitFor(() => expect(screen.getByText("Thermocouple Type")).toBeTruthy());
    expect(screen.queryByText("Terminal Configuration")).toBeNull();
    const port = screen.getByDisplayValue("3");
    expect(port).toBeTruthy();
  });

  describe("configure against a live cluster", () => {
    it("should create index and data channels, update the device, and save the task", async () => {
      const dev = await createNIDevice(client);
      const namedChannel = uniqueName("ai_named");
      const rendered = await renderAnalogRead({
        config: {
          ...NI.Task.ZERO_ANALOG_READ_PAYLOAD.config,
          channels: [
            createChannel("ai_voltage", 0, { device: dev.key }),
            createChannel("ai_current", 1, { device: dev.key, name: namedChannel }),
          ],
        },
      });
      await clickConfigure();
      const taskKey = await awaitTaskKey(rendered);
      const created = await client.tasks.retrieve({
        key: taskKey,
        schemas: NI.Task.ANALOG_READ_SCHEMAS,
      });
      expect(created.type).toBe(NI.Task.ANALOG_READ_TYPE);
      expect(task.rackKey(created.key)).toBe(dev.rack);
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

    it("should reuse existing channels when reconfigured", async () => {
      const dev = await createNIDevice(client);
      const rendered = await renderAnalogRead({
        config: {
          ...NI.Task.ZERO_ANALOG_READ_PAYLOAD.config,
          channels: [createChannel("ai_voltage", 0, { device: dev.key })],
        },
      });
      await clickConfigure();
      const taskKey = await awaitTaskKey(rendered);
      const first = await client.tasks.retrieve({
        key: taskKey,
        schemas: NI.Task.ANALOG_READ_SCHEMAS,
      });
      await clickConfigure();
      await waitFor(async () => {
        const again = await client.tasks.retrieve({
          key: taskKey,
          schemas: NI.Task.ANALOG_READ_SCHEMAS,
        });
        expect(again.config.channels[0].channel).toBe(first.config.channels[0].channel);
      });
      const dvc = await client.devices.retrieve({
        key: dev.key,
        schemas: NI.Device.SCHEMAS,
      });
      expect(dvc.properties.analogInput.channels["0"]).toBe(
        first.config.channels[0].channel,
      );
    });

    it("should surface an error when channels span devices on different racks", async () => {
      const devA = await createNIDevice(client);
      const devB = await createNIDevice(client);
      const { statuses } = await renderAnalogRead({
        config: {
          ...NI.Task.ZERO_ANALOG_READ_PAYLOAD.config,
          channels: [
            createChannel("ai_voltage", 0, { device: devA.key }),
            createChannel("ai_voltage", 1, { device: devB.key }),
          ],
        },
      });
      await clickConfigure();
      await awaitStatusDescription(statuses, /are on different racks/);
    });

    it("should surface an error when the task has no channels", async () => {
      const { statuses } = await renderAnalogRead({
        config: { ...NI.Task.ZERO_ANALOG_READ_PAYLOAD.config, channels: [] },
      });
      await clickConfigure();
      await awaitStatusDescription(
        statuses,
        /No devices selected in task configuration/,
      );
    });
  });
});
