// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type rack, type Synnax, task } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { type Status } from "@synnaxlabs/pluto";
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
import { awaitCommand, clickDeploy, renderTaskFormTab } from "@/platform/task/testutil";
import { stubGeometry, uniqueName } from "@/testutil";

const client = createTestClient();

stubGeometry();

let testRack: rack.Rack;

beforeAll(async () => {
  testRack = await client.racks.create({ name: uniqueName("ecat_rack") });
});

// Draft creates mint their own key; the zero payload's empty key must not be sent.
const { key: _key, ...ZERO_DRAFT } = EtherCAT.Task.ZERO_WRITE_PAYLOAD;

const createDraft = async (
  client: Synnax,
  config: EtherCAT.Task.WritePayload["config"],
) => await client.tasks.create({ ...ZERO_DRAFT, config }, EtherCAT.Task.WRITE_SCHEMAS);

const deployAndAwaitTask = async (
  client: Synnax,
  container: ParentNode,
  key: task.Key,
) => {
  const streamer = await client.openStreamer(task.COMMAND_CHANNEL_NAME);
  try {
    await clickDeploy(container);
    await awaitCommand(streamer, key);
  } finally {
    streamer.close();
  }
  return await client.tasks.retrieve({
    key,
    schemas: EtherCAT.Task.WRITE_SCHEMAS,
  });
};

const renderWrite = async (config: EtherCAT.Task.WritePayload["config"]) => {
  const draft = await createDraft(client, config);
  const statuses: Status.NotificationSpec[] = [];
  const rendered = await renderTaskFormTab(EtherCAT.Task.Write, {
    client,
    taskKey: draft.key,
    onStatuses: (next) => {
      statuses.length = 0;
      statuses.push(...next);
    },
  });
  return { ...rendered, draft, statuses };
};

const awaitStatus = async (
  statuses: Status.NotificationSpec[],
  pattern: RegExp,
): Promise<void> => {
  await waitFor(() => {
    const matched = statuses.some(
      (s) => pattern.test(s.description ?? "") || pattern.test(s.message),
    );
    expect(matched).toBe(true);
  });
};

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

  describe("deploying against a live cluster", () => {
    it("should create command and state channels, update the slave, and save the task", async () => {
      const identifier = createIdentifier();
      const slave = await createSlaveDevice(client, testRack.key, {
        identifier,
        network: "eth0",
        pdos: createPDOs(),
      });
      const { container, draft } = await renderWrite({
        ...EtherCAT.Task.ZERO_WRITE_PAYLOAD.config,
        channels: [createAutoOutputChannel(slave.key, "Control")],
      });
      const created = await deployAndAwaitTask(client, container, draft.key);
      expect(created.type).toBe(EtherCAT.Task.WRITE_TYPE);
      expect(created.rack).toBe(testRack.key);
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
      const { container, draft } = await renderWrite({
        ...EtherCAT.Task.ZERO_WRITE_PAYLOAD.config,
        channels: [
          createAutoOutputChannel(slave.key, "Control", {
            cmdChannelName: cmdName,
            stateChannelName: stateName,
          }),
        ],
      });
      const created = await deployAndAwaitTask(client, container, draft.key);
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
      const { container, statuses } = await renderWrite({
        ...EtherCAT.Task.ZERO_WRITE_PAYLOAD.config,
        channels: [
          createAutoOutputChannel(slaveA.key, "Control"),
          createAutoOutputChannel(slaveB.key, "Control"),
        ],
      });
      await clickDeploy(container);
      await awaitStatus(statuses, /Failed to/);
      await awaitStatus(statuses, /All slaves must be on the same network/);
    });

    it("should surface an error when no slave has a valid network", async () => {
      const slave = await createSlaveDevice(client, testRack.key, {
        identifier: createIdentifier(),
        network: "",
        pdos: createPDOs(),
      });
      const { container, statuses } = await renderWrite({
        ...EtherCAT.Task.ZERO_WRITE_PAYLOAD.config,
        channels: [createAutoOutputChannel(slave.key, "Control")],
      });
      await clickDeploy(container);
      await awaitStatus(statuses, /Failed to/);
      await awaitStatus(statuses, /No valid network found/);
    });
  });
});
