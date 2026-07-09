// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type rack, task } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeAll, describe, expect, it } from "vitest";

import { EtherCAT } from "@/feature/ethercat";
import {
  createAutoOutputChannel,
  createIdentifier,
  createManualOutputChannel,
  createPDOs,
  createSlaveDevice,
} from "@/feature/ethercat/testutil";
import {
  awaitTaskKey,
  clickConfigure,
  renderTaskFormView,
} from "@/platform/task/testutil";
import { stubGeometry, uniqueName } from "@/testutil";

const client = createTestClient();

stubGeometry();

let testRack: rack.Rack;

beforeAll(async () => {
  testRack = await client.racks.create({ name: uniqueName("ecat_rack") });
});

const renderWrite = async (config?: unknown) =>
  await renderTaskFormView(EtherCAT.Task.Write, EtherCAT.Task.WRITE_TYPE, {
    client,
    args: config == null ? {} : { config },
  });

describe("EtherCAT Write", () => {
  it("should render output channels with their port labels", async () => {
    const slave = await createSlaveDevice(client, testRack.key, {
      identifier: createIdentifier(),
      network: "eth0",
      pdos: createPDOs(),
    });
    await renderWrite({
      ...EtherCAT.Task.ZERO_WRITE_PAYLOAD.config,
      channels: [
        createAutoOutputChannel(slave.key, "Control"),
        createManualOutputChannel(slave.key, 0x7000, 3),
      ],
    });
    await waitFor(() => expect(screen.getByText("Control")).toBeTruthy());
    expect(screen.getByText("0x7000:3")).toBeTruthy();
  });

  it("should show the manual address fields when a manual output channel is selected", async () => {
    const slave = await createSlaveDevice(client, testRack.key, {
      identifier: createIdentifier(),
      network: "eth0",
    });
    await renderWrite({
      ...EtherCAT.Task.ZERO_WRITE_PAYLOAD.config,
      channels: [createManualOutputChannel(slave.key, 0x7000, 4)],
    });
    fireEvent.click(await screen.findByText("0x7000:4"));
    await waitFor(() => expect(screen.getByText("Index (hex)")).toBeTruthy());
    expect(screen.getByText("Subindex")).toBeTruthy();
    expect(screen.getByDisplayValue("4")).toBeTruthy();
  });

  describe("configure against a live cluster", () => {
    it("should create command and state channels, update the slave, and save the task", async () => {
      const identifier = createIdentifier();
      const slave = await createSlaveDevice(client, testRack.key, {
        identifier,
        network: "eth0",
        pdos: createPDOs(),
      });
      const rendered = await renderWrite({
        ...EtherCAT.Task.ZERO_WRITE_PAYLOAD.config,
        channels: [createAutoOutputChannel(slave.key, "Control")],
      });
      await clickConfigure();
      const taskKey = await awaitTaskKey(rendered);
      const created = await client.tasks.retrieve({
        key: taskKey,
        schemas: EtherCAT.Task.WRITE_SCHEMAS,
      });
      expect(created.type).toBe(EtherCAT.Task.WRITE_TYPE);
      expect(task.rackKey(created.key)).toBe(testRack.key);
      const [ch] = created.config.channels;
      expect(ch.cmdChannel).not.toBe(0);
      expect(ch.stateChannel).not.toBe(0);

      const cmd = await client.channels.retrieve(ch.cmdChannel);
      expect(cmd.name).toBe(`${identifier}_Control_cmd`);
      expect(cmd.dataType.toString()).toBe("uint16");
      const cmdIndex = await client.channels.retrieve(cmd.index);
      expect(cmdIndex.name).toBe(`${identifier}_Control_cmd_time`);
      expect(cmdIndex.isIndex).toBe(true);
      const state = await client.channels.retrieve(ch.stateChannel);
      expect(state.name).toBe(`${identifier}_Control_state`);

      const updated = await client.devices.retrieve({
        key: slave.key,
        schemas: EtherCAT.Device.SLAVE_SCHEMAS,
      });
      expect(updated.properties.writeStateIndex).not.toBe(0);
      const { channels } = updated.properties.write;
      expect(EtherCAT.Task.getChannelByMapKey(channels, "auto_Control")).toBe(
        ch.cmdChannel,
      );
      expect(EtherCAT.Task.getChannelByMapKey(channels, "auto_Control_state")).toBe(
        ch.stateChannel,
      );
      const stateIndex = await client.channels.retrieve(
        updated.properties.writeStateIndex,
      );
      expect(stateIndex.name).toBe(`${identifier}_state_time`);
    });

    it("should honor custom command and state channel names", async () => {
      const slave = await createSlaveDevice(client, testRack.key, {
        identifier: createIdentifier(),
        network: "eth0",
        pdos: createPDOs(),
      });
      const cmdName = uniqueName("ecat_cmd");
      const stateName = uniqueName("ecat_state");
      const rendered = await renderWrite({
        ...EtherCAT.Task.ZERO_WRITE_PAYLOAD.config,
        channels: [
          createAutoOutputChannel(slave.key, "Control", {
            cmdChannelName: cmdName,
            stateChannelName: stateName,
          }),
        ],
      });
      await clickConfigure();
      const taskKey = await awaitTaskKey(rendered);
      const created = await client.tasks.retrieve({
        key: taskKey,
        schemas: EtherCAT.Task.WRITE_SCHEMAS,
      });
      const [ch] = created.config.channels;
      const cmd = await client.channels.retrieve(ch.cmdChannel);
      expect(cmd.name).toBe(cmdName);
      const state = await client.channels.retrieve(ch.stateChannel);
      expect(state.name).toBe(stateName);
    });

    it("should surface an error when slaves are on different networks", async () => {
      const slaveA = await createSlaveDevice(client, testRack.key, {
        identifier: createIdentifier(),
        network: "eth0",
        pdos: createPDOs(),
      });
      const slaveB = await createSlaveDevice(client, testRack.key, {
        identifier: createIdentifier(),
        network: "eth1",
        pdos: createPDOs(),
      });
      await renderWrite({
        ...EtherCAT.Task.ZERO_WRITE_PAYLOAD.config,
        channels: [
          createAutoOutputChannel(slaveA.key, "Control"),
          createAutoOutputChannel(slaveB.key, "Control"),
        ],
      });
      await clickConfigure();
      fireEvent.click(await screen.findByText(/Failed to/));
      await waitFor(() =>
        expect(screen.getByText(/All slaves must be on the same network/)).toBeTruthy(),
      );
    });

    it("should surface an error when no slave has a valid network", async () => {
      const slave = await createSlaveDevice(client, testRack.key, {
        identifier: createIdentifier(),
        network: "",
        pdos: createPDOs(),
      });
      await renderWrite({
        ...EtherCAT.Task.ZERO_WRITE_PAYLOAD.config,
        channels: [createAutoOutputChannel(slave.key, "Control")],
      });
      await clickConfigure();
      fireEvent.click(await screen.findByText(/Failed to/));
      await waitFor(() =>
        expect(screen.getByText(/No valid network found/)).toBeTruthy(),
      );
    });
  });
});
