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
import { createNIDevice, renderNITaskForm } from "@/feature/ni/task/testutil";
import {
  awaitTaskKey,
  clickConfigure,
  selectFromDropdown,
} from "@/platform/task/testutil";
import { isSelectButtonSelected, stubGeometry, uniqueName } from "@/testutil";

stubGeometry();

const client = createTestClient();

type CreateChannelOverrides = Partial<NI.Task.AOChannel>;

const createChannel = (
  type: NI.Task.AOChannelType,
  port: number,
  overrides: CreateChannelOverrides = {},
): NI.Task.AOChannel =>
  ({
    ...NI.Task.ZERO_AO_CHANNELS[type],
    key: id.create(),
    port,
    cmdChannelName: `cmd_${type}_${port}`,
    ...overrides,
  }) as NI.Task.AOChannel;

const renderAnalogWrite = async (args = {}) =>
  await renderNITaskForm(NI.Task.AnalogWrite, NI.Task.ANALOG_WRITE_TYPE, {
    client,
    args,
  });

const createConfig = (
  channels: NI.Task.AOChannel[],
  device = "placeholder_device",
) => ({ ...NI.Task.ZERO_ANALOG_WRITE_PAYLOAD.config, device, channels });

describe("AnalogWrite", () => {
  it("should render the detail form for every channel type as it is selected", async () => {
    const cases: [NI.Task.AOChannelType, string][] = [
      ["ao_current", "Minimum Value"],
      ["ao_func_gen", "Frequency"],
      ["ao_voltage", "Custom Scaling"],
    ];
    await renderAnalogWrite({
      config: createConfig(cases.map(([type], i) => createChannel(type, i))),
    });
    for (const [i, [type, distinguishingLabel]] of cases.entries()) {
      fireEvent.click(await screen.findByText(`cmd_${type}_${i}`));
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

  it("should switch the function generator wave type when a wave button is clicked", async () => {
    await renderAnalogWrite({
      config: createConfig([createChannel("ao_func_gen", 0)]),
    });
    fireEvent.click(await screen.findByText("cmd_ao_func_gen_0"));
    await screen.findByText("Triangle");
    await waitFor(() => expect(isSelectButtonSelected("Sine")).toBe(true));
    expect(isSelectButtonSelected("Triangle")).toBe(false);
    fireEvent.click(screen.getByText("Triangle"));
    await waitFor(() => expect(isSelectButtonSelected("Triangle")).toBe(true));
    expect(isSelectButtonSelected("Sine")).toBe(false);
  });

  it("should swap the channel to the newly selected type", async () => {
    await renderAnalogWrite({
      config: createConfig([createChannel("ao_voltage", 2)]),
    });
    fireEvent.click(await screen.findByText("cmd_ao_voltage_2"));
    await screen.findByText("Custom Scaling");
    await selectFromDropdown("Voltage", "Function Generator");
    await waitFor(() => expect(screen.getByText("Frequency")).toBeTruthy());
    expect(screen.queryByText("Custom Scaling")).toBeNull();
  });

  describe("configure against a live cluster", () => {
    it("should create command and state channels and update the device", async () => {
      const dev = await createNIDevice(client);
      const rendered = await renderAnalogWrite({
        config: createConfig(
          [
            createChannel("ao_voltage", 0, {
              cmdChannelName: "",
              stateChannelName: "",
            }),
          ],
          dev.key,
        ),
      });
      await clickConfigure();
      const taskKey = await awaitTaskKey(rendered);
      const created = await client.tasks.retrieve({
        key: taskKey,
        schemas: NI.Task.ANALOG_WRITE_SCHEMAS,
      });
      expect(created.type).toBe(NI.Task.ANALOG_WRITE_TYPE);
      expect(task.rackKey(created.key)).toBe(dev.rack);
      const [c0] = created.config.channels;
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

    it("should use custom command and state channel names when provided", async () => {
      const dev = await createNIDevice(client);
      const cmdName = uniqueName("ao_cmd");
      const stateName = uniqueName("ao_state");
      const rendered = await renderAnalogWrite({
        config: createConfig(
          [
            createChannel("ao_voltage", 0, {
              cmdChannelName: cmdName,
              stateChannelName: stateName,
            }),
          ],
          dev.key,
        ),
      });
      await clickConfigure();
      const taskKey = await awaitTaskKey(rendered);
      const created = await client.tasks.retrieve({
        key: taskKey,
        schemas: NI.Task.ANALOG_WRITE_SCHEMAS,
      });
      const [c0] = created.config.channels;
      const cmd = await client.channels.retrieve(c0.cmdChannel);
      expect(cmd.name).toBe(cmdName);
      const state = await client.channels.retrieve(c0.stateChannel);
      expect(state.name).toBe(stateName);
      const cmdIndex = await client.channels.retrieve(cmd.index);
      expect(cmdIndex.name).toBe(`${cmdName}_time`);
    });

    it("should reuse existing channels when reconfigured", async () => {
      const dev = await createNIDevice(client);
      const rendered = await renderAnalogWrite({
        config: createConfig(
          [
            createChannel("ao_voltage", 0, {
              cmdChannelName: "",
              stateChannelName: "",
            }),
          ],
          dev.key,
        ),
      });
      await clickConfigure();
      const taskKey = await awaitTaskKey(rendered);
      const first = await client.tasks.retrieve({
        key: taskKey,
        schemas: NI.Task.ANALOG_WRITE_SCHEMAS,
      });
      await clickConfigure();
      await waitFor(async () => {
        const again = await client.tasks.retrieve({
          key: taskKey,
          schemas: NI.Task.ANALOG_WRITE_SCHEMAS,
        });
        expect(again.config.channels[0].cmdChannel).toBe(
          first.config.channels[0].cmdChannel,
        );
        expect(again.config.channels[0].stateChannel).toBe(
          first.config.channels[0].stateChannel,
        );
      });
    });
  });
});
