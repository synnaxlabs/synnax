// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { LabJack } from "@/feature/labjack";
import {
  createAOChannel,
  createDOChannel,
  createLabJackDevice,
} from "@/feature/labjack/testutil";
import {
  awaitTaskKey,
  clickConfigure,
  findDialogTriggerByText,
  renderTaskFormTab,
} from "@/platform/task/testutil";
import { getIconButton, stubGeometry, uniqueName } from "@/testutil";

const client = createTestClient();

stubGeometry();

const renderWrite = async (params = {}) =>
  await renderTaskFormTab(LabJack.Task.Write, LabJack.Task.WRITE_TYPE, {
    client,
    params,
  });

const createConfig = (
  device: string,
  channels: LabJack.Task.OutputChannel[],
): unknown => ({ ...LabJack.Task.ZERO_WRITE_PAYLOAD.config, device, channels });

describe("LabJack Write", () => {
  it("should render output channels with port selectors and type buttons", async () => {
    const dev = await createLabJackDevice(client);
    await renderWrite({
      config: createConfig(dev.key, [createDOChannel("DIO4"), createAOChannel("DAC0")]),
    });
    await waitFor(() => expect(screen.getByText("FIO4")).toBeTruthy());
    expect(screen.getByText("DAC0")).toBeTruthy();
    expect(screen.getAllByText("Analog")).toHaveLength(2);
    expect(screen.getAllByText("Digital")).toHaveLength(2);
  });

  it("should swap the channel to the analog port space when Analog is clicked", async () => {
    const dev = await createLabJackDevice(client);
    await renderWrite({
      config: createConfig(dev.key, [createDOChannel("DIO4")]),
    });
    await waitFor(() => expect(screen.getByText("FIO4")).toBeTruthy());
    fireEvent.click(screen.getByText("Analog"));
    await waitFor(() => expect(screen.getByText("DAC0")).toBeTruthy());
    expect(screen.queryByText("FIO4")).toBeNull();
  });

  it("should add a zero channel when the list is empty and Add is pressed", async () => {
    const dev = await createLabJackDevice(client);
    const { container } = await renderWrite({
      config: createConfig(dev.key, []),
    });
    await waitFor(() => expect(screen.getByText("No channels in task.")).toBeTruthy());
    fireEvent.click(getIconButton(container, "add"));
    await waitFor(() => expect(screen.getByText("FIO4")).toBeTruthy());
  });

  it("should add the next open port of the same type when Add is pressed", async () => {
    const dev = await createLabJackDevice(client);
    const { container } = await renderWrite({
      config: createConfig(dev.key, [createDOChannel("DIO4")]),
    });
    await waitFor(() => expect(screen.getByText("FIO4")).toBeTruthy());
    fireEvent.click(getIconButton(container, "add"));
    await waitFor(() => expect(screen.getByText("FIO5")).toBeTruthy());
  });

  it("should update the port through the port select dialog", async () => {
    const dev = await createLabJackDevice(client);
    await renderWrite({
      config: createConfig(dev.key, [createDOChannel("DIO4")]),
    });
    fireEvent.click(await findDialogTriggerByText("FIO4"));
    fireEvent.click(await screen.findByText("EIO2"));
    await waitFor(() => expect(screen.queryByText("EIO2")).toBeTruthy());
  });

  describe("configure against a live cluster", () => {
    it("should create command and state channels, update the device, and save the task", async () => {
      const dev = await createLabJackDevice(client);
      const rendered = await renderWrite({
        config: createConfig(dev.key, [
          createDOChannel("DIO4"),
          createAOChannel("DAC0"),
        ]),
      });
      await clickConfigure();
      const taskKey = await awaitTaskKey(rendered);
      const created = await client.tasks.retrieve({
        key: taskKey,
        schemas: LabJack.Task.WRITE_SCHEMAS,
      });
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
      const rendered = await renderWrite({
        config: createConfig(dev.key, [
          createDOChannel("DIO4", {
            cmdChannelName: cmdName,
            stateChannelName: stateName,
          }),
        ]),
      });
      await clickConfigure();
      const taskKey = await awaitTaskKey(rendered);
      const created = await client.tasks.retrieve({
        key: taskKey,
        schemas: LabJack.Task.WRITE_SCHEMAS,
      });
      const [ch] = created.config.channels;
      const cmd = await client.channels.retrieve(ch.cmdChannel);
      expect(cmd.name).toBe(cmdName);
      const cmdIndex = await client.channels.retrieve(cmd.index);
      expect(cmdIndex.name).toBe(`${cmdName}_time`);
      const state = await client.channels.retrieve(ch.stateChannel);
      expect(state.name).toBe(stateName);
    });

    it("should reuse existing command and state channels when reconfigured", async () => {
      const dev = await createLabJackDevice(client);
      const config = createConfig(dev.key, [createDOChannel("DIO4")]);
      const first = await renderWrite({ config });
      await clickConfigure();
      const firstKey = await awaitTaskKey(first);
      const firstTask = await client.tasks.retrieve({
        key: firstKey,
        schemas: LabJack.Task.WRITE_SCHEMAS,
      });
      first.unmount();

      const second = await renderWrite({ config });
      await clickConfigure();
      const secondKey = await awaitTaskKey(second);
      const secondTask = await client.tasks.retrieve({
        key: secondKey,
        schemas: LabJack.Task.WRITE_SCHEMAS,
      });
      expect(secondTask.config.channels[0].cmdChannel).toBe(
        firstTask.config.channels[0].cmdChannel,
      );
      expect(secondTask.config.channels[0].stateChannel).toBe(
        firstTask.config.channels[0].stateChannel,
      );
    });
  });
});
