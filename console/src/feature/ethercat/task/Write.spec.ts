// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type rack, type Synnax, type task } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { type Status } from "@synnaxlabs/lyra/status";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeAll, describe, expect, it } from "vitest";
import { type z } from "zod";

import { EtherCAT } from "@/feature/ethercat";
import {
  CONTROL_ENTRY,
  createEntryWriteChannel,
  createIdentifier,
  createPDOs,
  createRepeatedNamePDOs,
  createRepeatedNameSlave,
  createSlaveDevice,
  createUnresolvedWriteChannel,
} from "@/feature/ethercat/testutil";
import {
  awaitEditableForm,
  clickDeploy,
  createWithHook,
  deployAndAwaitTask,
  renderTaskFormTab,
} from "@/platform/task/testutil";
import { uniqueName } from "@/testutil";

const client = createTestClient();

let testRack: rack.Rack;

beforeAll(async () => {
  testRack = await client.racks.create({ name: uniqueName("ecat_rack") });
});

// Drafts carry no key; the created row mints its own.
const ZERO_DRAFT: task.New<EtherCAT.Task.WriteSchemas> = {
  name: "EtherCAT write task",
  type: EtherCAT.Task.WRITE_TYPE,
  config: EtherCAT.Task.WRITE_SCHEMAS.config.parse({}),
};

const createDraft = async (
  client: Synnax,
  config: z.infer<EtherCAT.Task.WriteSchemas["config"]>,
) => await client.tasks.create({ ...ZERO_DRAFT, config }, EtherCAT.Task.WRITE_SCHEMAS);

// The form renders read-only until the update grant lands, and a preview field renders
// no input, so wait for it to become editable before querying fields.
const renderWrite = async (config: z.infer<EtherCAT.Task.WriteSchemas["config"]>) => {
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
  await awaitEditableForm();
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

describe("Write", () => {
  it("should render output channels with their port labels", async () => {
    const slave = await createSlaveDevice(client, testRack.key, {
      identifier: createIdentifier(),
      network: "eth0",
      pdos: createPDOs(),
    });
    await renderWrite({
      ...EtherCAT.Task.WRITE_SCHEMAS.config.parse({}),
      channels: [
        createEntryWriteChannel(slave.key, CONTROL_ENTRY),
        createUnresolvedWriteChannel(slave.key, "Output"),
      ],
    });
    expect(await screen.findByRole("option", { name: /0x7000:01/ })).toBeTruthy();
    expect(screen.getByRole("option", { name: /^Output/ })).toBeTruthy();
  });

  it("should label a selected output channel by its PDO and entry names", async () => {
    const slave = await createRepeatedNameSlave(client, testRack.key);
    const { outputs } = createRepeatedNamePDOs();
    await renderWrite({
      ...EtherCAT.Task.WRITE_SCHEMAS.config.parse({}),
      channels: [createEntryWriteChannel(slave.key, outputs[3])],
    });
    fireEvent.click(await screen.findByRole("option", { name: /0x7030:01/ }));
    expect(await screen.findByText("Channel 4 › Output · 0x7030:01")).toBeTruthy();
    expect(screen.getByText("Channel 4 › Output")).toBeTruthy();
  });

  it("should create command and state channels for each of eight entries that share a name", async () => {
    const slave = await createRepeatedNameSlave(client, testRack.key);
    const { identifier } = slave.properties;
    const { outputs } = createRepeatedNamePDOs();
    const { container, draft } = await renderWrite({
      ...EtherCAT.Task.WRITE_SCHEMAS.config.parse({}),
      channels: outputs.map((e) => createEntryWriteChannel(slave.key, e)),
    });
    const created = await deployAndAwaitTask(
      client,
      container,
      draft.key,
      EtherCAT.Task.WRITE_SCHEMAS,
    );
    const cmdKeys = created.config.channels.map(({ cmdChannel }) => cmdChannel);
    const stateKeys = created.config.channels.map(({ stateChannel }) => stateChannel);
    expect(new Set([...cmdKeys, ...stateKeys]).size).toBe(16);
    const cmds = await client.channels.retrieve(cmdKeys);
    expect(cmds.map(({ name }) => name).sort()).toEqual(
      outputs.map((_, i) => `${identifier}_Channel_${i + 1}_Output_cmd`).sort(),
    );
    const updated = await client.devices.retrieve({
      key: slave.key,
      schemas: EtherCAT.Device.SLAVE_SCHEMAS,
    });
    const { channels } = updated.properties.write;
    outputs.forEach((e, i) => {
      expect(channels[`${e.index}_1`]).toBe(cmdKeys[i]);
      expect(channels[`state_${e.index}_1`]).toBe(stateKeys[i]);
    });
  });

  it("should block configuration of an unresolved output channel", async () => {
    const slave = await createRepeatedNameSlave(client, testRack.key);
    const { container, statuses } = await renderWrite({
      ...EtherCAT.Task.WRITE_SCHEMAS.config.parse({}),
      channels: [createUnresolvedWriteChannel(slave.key, "Output")],
    });
    await clickDeploy(container);
    await awaitStatus(
      statuses,
      /'Output' matches more than one output entry on slave .+\. Pick an entry\.$/,
    );
  });

  it("should block configuration when two channels use the same entry", async () => {
    const slave = await createRepeatedNameSlave(client, testRack.key);
    const { outputs } = createRepeatedNamePDOs();
    const { container, statuses } = await renderWrite({
      ...EtherCAT.Task.WRITE_SCHEMAS.config.parse({}),
      channels: [
        createEntryWriteChannel(slave.key, outputs[3]),
        createEntryWriteChannel(slave.key, outputs[3]),
      ],
    });
    await clickDeploy(container);
    await awaitStatus(
      statuses,
      new RegExp(
        `More than one channel uses entry 0x7030:01 on slave ${slave.name}\\. ` +
          "Pick a different entry for one of them\\.$",
      ),
    );
  });

  it("should block configuration of a channel with no entry", async () => {
    const slave = await createSlaveDevice(client, testRack.key, {
      identifier: createIdentifier(),
      network: "eth0",
      pdos: createPDOs(),
    });
    const { container, statuses } = await renderWrite({
      ...EtherCAT.Task.WRITE_SCHEMAS.config.parse({}),
      channels: [{ ...EtherCAT.Task.createWriteChannel([]), device: slave.key }],
    });
    await clickDeploy(container);
    await awaitStatus(
      statuses,
      new RegExp(`A channel on slave ${slave.name} has no entry\\. Pick an entry\\.$`),
    );
  });

  it("should create command and state channels, update the slave, and save the task", async () => {
    const identifier = createIdentifier();
    const slave = await createSlaveDevice(client, testRack.key, {
      identifier,
      network: "eth0",
      pdos: createPDOs(),
    });
    const { container, draft } = await renderWrite({
      ...EtherCAT.Task.WRITE_SCHEMAS.config.parse({}),
      channels: [createEntryWriteChannel(slave.key, CONTROL_ENTRY)],
    });
    const created = await deployAndAwaitTask(
      client,
      container,
      draft.key,
      EtherCAT.Task.WRITE_SCHEMAS,
    );
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
    expect(channels[`${0x7000}_1`]).toBe(ch.cmdChannel);
    expect(channels[`state_${0x7000}_1`]).toBe(ch.stateChannel);
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
      ...EtherCAT.Task.WRITE_SCHEMAS.config.parse({}),
      channels: [
        createEntryWriteChannel(slave.key, CONTROL_ENTRY, {
          cmdChannelName: cmdName,
          stateChannelName: stateName,
        }),
      ],
    });
    const created = await deployAndAwaitTask(
      client,
      container,
      draft.key,
      EtherCAT.Task.WRITE_SCHEMAS,
    );
    const [ch] = created.config.channels;
    const cmd = await client.channels.retrieve(ch.cmdChannel);
    expect(cmd.name).toBe(cmdName);
    const state = await client.channels.retrieve(ch.stateChannel);
    expect(state.name).toBe(stateName);
  });

  it("should bind a new entry to the channel the device already maps", async () => {
    const slave = await createSlaveDevice(client, testRack.key, {
      identifier: createIdentifier(),
      network: "eth0",
      pdos: createPDOs(),
    });
    const config = {
      ...EtherCAT.Task.WRITE_SCHEMAS.config.parse({}),
      channels: [createEntryWriteChannel(slave.key, CONTROL_ENTRY)],
    };
    const first = await renderWrite(config);
    const firstTask = await deployAndAwaitTask(
      client,
      first.container,
      first.draft.key,
      EtherCAT.Task.WRITE_SCHEMAS,
    );
    first.unmount();
    const [bound] = firstTask.config.channels;
    const cmd = await client.channels.retrieve(bound.cmdChannel);
    const second = await renderWrite(config);
    await screen.findByText(cmd.name);
    await waitFor(async () => {
      const saved = await client.tasks.retrieve({
        key: second.draft.key,
        schemas: EtherCAT.Task.WRITE_SCHEMAS,
      });
      expect(saved.config.channels[0].cmdChannel).toBe(bound.cmdChannel);
      expect(saved.config.channels[0].stateChannel).toBe(bound.stateChannel);
    });
  });

  it("should recreate only the deleted half of a command and state pair", async () => {
    const slave = await createSlaveDevice(client, testRack.key, {
      identifier: createIdentifier(),
      network: "eth0",
      pdos: createPDOs(),
    });
    const config = {
      ...EtherCAT.Task.WRITE_SCHEMAS.config.parse({}),
      channels: [createEntryWriteChannel(slave.key, CONTROL_ENTRY)],
    };
    const first = await renderWrite(config);
    const firstTask = await deployAndAwaitTask(
      client,
      first.container,
      first.draft.key,
      EtherCAT.Task.WRITE_SCHEMAS,
    );
    first.unmount();
    const [bound] = firstTask.config.channels;
    await client.channels.delete(bound.stateChannel);

    const second = await renderWrite(config);
    const secondTask = await deployAndAwaitTask(
      client,
      second.container,
      second.draft.key,
      EtherCAT.Task.WRITE_SCHEMAS,
    );
    const [rebound] = secondTask.config.channels;
    expect(rebound.cmdChannel).toBe(bound.cmdChannel);
    expect(rebound.stateChannel).not.toBe(bound.stateChannel);
    expect(rebound.stateChannel).not.toBe(0);
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
      ...EtherCAT.Task.WRITE_SCHEMAS.config.parse({}),
      channels: [
        createEntryWriteChannel(slaveA.key, CONTROL_ENTRY),
        createEntryWriteChannel(slaveB.key, CONTROL_ENTRY),
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
      ...EtherCAT.Task.WRITE_SCHEMAS.config.parse({}),
      channels: [createEntryWriteChannel(slave.key, CONTROL_ENTRY)],
    });
    await clickDeploy(container);
    await awaitStatus(statuses, /Failed to/);
    await awaitStatus(statuses, /No network found/);
  });

  it("should create the task with one channel", async () => {
    const tsk = await createWithHook(
      EtherCAT.Task.useCreateWrite,
      EtherCAT.Task.WRITE_SCHEMAS,
    );
    expect(tsk.config.channels).toHaveLength(1);
  });
});
