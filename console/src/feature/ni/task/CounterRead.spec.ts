// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, task } from "@synnaxlabs/client";
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
  selectFromDropdown,
} from "@/platform/task/testutil";
import { stubGeometry, uniqueName } from "@/testutil";

stubGeometry();

const client = createTestClient();

type CreateChannelOverrides = Partial<NI.Task.CIChannel>;

const createChannel = (
  type: NI.Task.CIChannelType,
  port: number,
  overrides: CreateChannelOverrides = {},
): NI.Task.CIChannel =>
  ({
    ...NI.Task.ZERO_CI_CHANNELS[type],
    key: id.create(),
    port,
    device: "placeholder_device",
    ...overrides,
  }) as NI.Task.CIChannel;

const renderCounterRead = async (args = {}) =>
  await renderNITaskForm(NI.Task.CounterRead, NI.Task.COUNTER_READ_TYPE, {
    client,
    args,
  });

const createConfig = (channels: NI.Task.CIChannel[]) => ({
  ...NI.Task.ZERO_COUNTER_READ_PAYLOAD.config,
  channels,
});

describe("CounterRead", () => {
  it("should render the detail form for every channel type as it is selected", async () => {
    const cases: [NI.Task.CIChannelType, string][] = [
      ["ci_frequency", "Measurement Method"],
      ["ci_edge_count", "Count Direction"],
      ["ci_period", "Measurement Method"],
      ["ci_pulse_width", "Starting Edge"],
      ["ci_semi_period", "Scaled Units"],
      ["ci_two_edge_sep", "Edge 1"],
      ["ci_velocity_linear", "Distance / Pulse"],
      ["ci_velocity_angular", "Pulses / Rev"],
      ["ci_position_linear", "Z Index Enable"],
      ["ci_position_angular", "Initial Angle"],
      ["ci_duty_cycle", "Active Edge"],
    ];
    await renderCounterRead({
      config: createConfig(
        cases.map(([type], i) => createChannel(type, i, { name: `chan_${type}` })),
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

  it("should reveal the measurement time field for two-counter high-frequency measurement", async () => {
    await renderCounterRead({
      config: createConfig([
        createChannel("ci_frequency", 0, { measMethod: "HighFreq2Ctr" }),
      ]),
    });
    await waitFor(() => expect(screen.getByText("Measurement Time (s)")).toBeTruthy());
    expect(screen.queryByText("Divisor")).toBeNull();
  });

  it("should reveal the divisor field for two-counter large-range measurement", async () => {
    await renderCounterRead({
      config: createConfig([
        createChannel("ci_frequency", 0, { measMethod: "LargeRng2Ctr" }),
      ]),
    });
    await waitFor(() => expect(screen.getByText("Divisor")).toBeTruthy());
    expect(screen.queryByText("Measurement Time (s)")).toBeNull();
  });

  it("should swap the channel to the newly selected type", async () => {
    await renderCounterRead({
      config: createConfig([createChannel("ci_frequency", 0)]),
    });
    await screen.findByText("Measurement Method");
    await selectFromDropdown("Frequency", "Edge Count");
    await waitFor(() => expect(screen.getByText("Count Direction")).toBeTruthy());
    expect(screen.queryByText("Measurement Method")).toBeNull();
  });

  describe("configure against a live cluster", () => {
    it("should create counter channels and update the device", async () => {
      const dev = await createNIDevice(client);
      const namedChannel = uniqueName("ctr_named");
      const rendered = await renderCounterRead({
        config: createConfig([
          createChannel("ci_frequency", 0, { device: dev.key }),
          createChannel("ci_edge_count", 1, { device: dev.key, name: namedChannel }),
        ]),
      });
      await clickConfigure();
      const taskKey = await awaitTaskKey(rendered);
      const created = await client.tasks.retrieve({
        key: taskKey,
        schemas: NI.Task.COUNTER_READ_SCHEMAS,
      });
      expect(created.type).toBe(NI.Task.COUNTER_READ_TYPE);
      expect(task.rackKey(created.key)).toBe(dev.rack);
      const [c0, c1] = created.config.channels;
      expect(c0.channel).not.toBe(0);
      expect(c1.channel).not.toBe(0);

      const identifier = dev.properties.identifier;
      const defaultNamed = await client.channels.retrieve(c0.channel);
      expect(defaultNamed.name).toBe(`${identifier}_ctr_0`);
      const named = await client.channels.retrieve(c1.channel);
      expect(named.name).toBe(namedChannel);

      const updated = await client.devices.retrieve({
        key: dev.key,
        schemas: NI.Device.SCHEMAS,
      });
      expect(updated.properties.counterInput.channels["0"]).toBe(c0.channel);
      expect(updated.properties.counterInput.channels["1"]).toBe(c1.channel);
      const index = await client.channels.retrieve(
        updated.properties.counterInput.index,
      );
      expect(index.name).toBe(`${identifier}_ctr_time`);
      expect(index.isIndex).toBe(true);
    });

    it("should surface an error when the task has no channels", async () => {
      const { statuses } = await renderCounterRead({
        config: createConfig([]),
      });
      await clickConfigure();
      await awaitStatusDescription(
        statuses,
        /No device selected in task configuration/,
      );
    });

    it("should surface an error when channels span devices on different racks", async () => {
      const devA = await createNIDevice(client);
      const devB = await createNIDevice(client);
      const { statuses } = await renderCounterRead({
        config: createConfig([
          createChannel("ci_frequency", 0, { device: devA.key }),
          createChannel("ci_frequency", 1, { device: devB.key }),
        ]),
      });
      await clickConfigure();
      await awaitStatusDescription(
        statuses,
        /Cannot create task with channels from multiple racks/,
      );
    });
  });
});
