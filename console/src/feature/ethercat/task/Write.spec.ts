// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  channel,
  type ethercat,
  type rack,
  type Synnax,
  type task,
} from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { type Status } from "@synnaxlabs/lyra/status";
import { id } from "@synnaxlabs/x";
import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { beforeAll, describe, expect, it } from "vitest";
import { type z } from "zod";

import { EtherCAT } from "@/feature/ethercat";
import {
  CONTROL_ENTRY,
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
  findChannelListItem,
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
        EtherCAT.Task.createEntryWriteChannel(slave.key, CONTROL_ENTRY),
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
      channels: [EtherCAT.Task.createEntryWriteChannel(slave.key, outputs[3])],
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
      channels: outputs.map((e) => EtherCAT.Task.createEntryWriteChannel(slave.key, e)),
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
    outputs.forEach((e, i) =>
      expect(updated.properties.write.channels).toContainEqual({
        index: e.index,
        subIndex: 1,
        command: cmdKeys[i],
        state: stateKeys[i],
      }),
    );
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

  it("should mark in the list each channel that shares an entry", async () => {
    const slave = await createRepeatedNameSlave(client, testRack.key);
    const { outputs } = createRepeatedNamePDOs();
    await renderWrite({
      ...EtherCAT.Task.WRITE_SCHEMAS.config.parse({}),
      channels: [
        EtherCAT.Task.createEntryWriteChannel(slave.key, outputs[3]),
        EtherCAT.Task.createEntryWriteChannel(slave.key, outputs[3]),
        EtherCAT.Task.createEntryWriteChannel(slave.key, outputs[0]),
      ],
    });
    const shared = await screen.findAllByRole("option", { name: /0x7030:01/ });
    await waitFor(() =>
      expect(shared.map((o) => o.classList.contains("lyra--status-error"))).toEqual([
        true,
        true,
      ]),
    );
    const alone = screen.getByRole("option", { name: /0x7000:01/ });
    expect(alone.classList.contains("lyra--status-error")).toBe(false);
  });

  it("should block configuration when two channels use the same entry", async () => {
    const slave = await createRepeatedNameSlave(client, testRack.key);
    const { outputs } = createRepeatedNamePDOs();
    const { container, statuses } = await renderWrite({
      ...EtherCAT.Task.WRITE_SCHEMAS.config.parse({}),
      channels: [
        EtherCAT.Task.createEntryWriteChannel(slave.key, outputs[3]),
        EtherCAT.Task.createEntryWriteChannel(slave.key, outputs[3]),
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

  it("should add a channel for every unused output of a slave from the context menu", async () => {
    const setpoint: ethercat.PDOEntry = {
      name: "Setpoint",
      index: 0x7000,
      subIndex: 2,
      bitLength: 16,
      dataType: "int16",
    };
    const slave = await createSlaveDevice(client, testRack.key, {
      identifier: createIdentifier(),
      network: "eth0",
      pdos: { inputs: [], outputs: [CONTROL_ENTRY, setpoint] },
    });
    await renderWrite({
      ...EtherCAT.Task.WRITE_SCHEMAS.config.parse({}),
      channels: [EtherCAT.Task.createEntryWriteChannel(slave.key, CONTROL_ENTRY)],
    });
    fireEvent.contextMenu(await screen.findByRole("option", { name: /0x7000:01/ }));
    fireEvent.click(await screen.findByText(`Add every output of ${slave.name}`));
    await waitFor(() =>
      expect(screen.getByRole("option", { name: /0x7000:02/ })).toBeTruthy(),
    );
    expect(screen.getAllByRole("option", { name: /0x7000:01/ })).toHaveLength(1);
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
      channels: [EtherCAT.Task.createEntryWriteChannel(slave.key, CONTROL_ENTRY)],
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
    expect(updated.properties.write.channels).toEqual([
      { index: 0x7000, subIndex: 1, command: ch.cmdChannel, state: ch.stateChannel },
    ]);
    const stateIndex = await client.channels.retrieve(
      updated.properties.writeStateIndex,
    );
    expect(stateIndex.name).toBe(`${identifier}_state_time`);
  });

  it("should save the stop value typed for a selected channel", async () => {
    const slave = await createSlaveDevice(client, testRack.key, {
      identifier: createIdentifier(),
      network: "eth0",
      pdos: createPDOs(),
    });
    const { draft } = await renderWrite({
      ...EtherCAT.Task.WRITE_SCHEMAS.config.parse({}),
      channels: [EtherCAT.Task.createEntryWriteChannel(slave.key, CONTROL_ENTRY)],
    });
    fireEvent.click(await screen.findByRole("option", { name: /0x7000:01/ }));
    const stopValue = await screen.findByLabelText("Stop value");
    fireEvent.change(stopValue, { target: { value: "5" } });
    fireEvent.blur(stopValue);
    await waitFor(async () => {
      const saved = await client.tasks.retrieve({
        key: draft.key,
        schemas: EtherCAT.Task.WRITE_SCHEMAS,
      });
      expect(saved.config.channels[0]).toMatchObject({ stopValue: 5 });
    });
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
        {
          ...EtherCAT.Task.createEntryWriteChannel(slave.key, CONTROL_ENTRY),
          cmdChannelName: cmdName,
          stateChannelName: stateName,
        },
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
      channels: [EtherCAT.Task.createEntryWriteChannel(slave.key, CONTROL_ENTRY)],
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
      channels: [EtherCAT.Task.createEntryWriteChannel(slave.key, CONTROL_ENTRY)],
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
    await client.tasks.delete(firstTask.key);
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

  const deploySlave = async (identifier: string) => {
    const slave = await createSlaveDevice(client, testRack.key, {
      identifier,
      network: "eth0",
      pdos: createPDOs(),
    });
    const { container, draft, unmount } = await renderWrite({
      ...EtherCAT.Task.WRITE_SCHEMAS.config.parse({}),
      channels: [EtherCAT.Task.createEntryWriteChannel(slave.key, CONTROL_ENTRY)],
    });
    const deployed = await deployAndAwaitTask(
      client,
      container,
      draft.key,
      EtherCAT.Task.WRITE_SCHEMAS,
    );
    unmount();
    const updated = await client.devices.retrieve({
      key: slave.key,
      schemas: EtherCAT.Device.SLAVE_SCHEMAS,
    });
    return { slave: updated, channel: deployed.config.channels[0], task: deployed.key };
  };

  // The Core refuses to delete a slave that a task uses.
  const deleteSlave = async (slave: string, task: string) => {
    await client.tasks.delete(task);
    await client.devices.delete(slave);
  };

  it("should reuse the channels of a deleted slave with the same identifier", async () => {
    const identifier = createIdentifier();
    const first = await deploySlave(identifier);
    await deleteSlave(first.slave.key, first.task);
    const second = await deploySlave(identifier);
    expect(second.channel.cmdChannel).toBe(first.channel.cmdChannel);
    expect(second.channel.stateChannel).toBe(first.channel.stateChannel);
    expect(second.slave.properties.writeStateIndex).toBe(
      first.slave.properties.writeStateIndex,
    );
  });

  it("should reuse a deleted slave's channels when a non-slave EtherCAT device exists", async () => {
    const coupler = await client.devices.create({
      key: id.create(),
      name: uniqueName("ecat_coupler"),
      rack: testRack.key,
      location: "test-location",
      make: EtherCAT.Device.MAKE,
      model: "EK1100",
      properties: {},
    });
    try {
      const identifier = createIdentifier();
      const first = await deploySlave(identifier);
      await deleteSlave(first.slave.key, first.task);
      const second = await deploySlave(identifier);
      expect(second.channel.cmdChannel).toBe(first.channel.cmdChannel);
    } finally {
      await client.devices.delete(coupler.key);
    }
  });

  it("should reject a custom command name that a live slave's channel holds", async () => {
    const cmdChannelName = uniqueName("ecat_cmd");
    const renderSlave = async () => {
      const slave = await createSlaveDevice(client, testRack.key, {
        identifier: createIdentifier(),
        network: "eth0",
        pdos: createPDOs(),
      });
      return await renderWrite({
        ...EtherCAT.Task.WRITE_SCHEMAS.config.parse({}),
        channels: [
          {
            ...EtherCAT.Task.createEntryWriteChannel(slave.key, CONTROL_ENTRY),
            cmdChannelName,
          },
        ],
      });
    };
    const owner = await renderSlave();
    await deployAndAwaitTask(
      client,
      owner.container,
      owner.draft.key,
      EtherCAT.Task.WRITE_SCHEMAS,
    );
    owner.unmount();
    const { container, statuses } = await renderSlave();
    await clickDeploy(container);
    await awaitStatus(
      statuses,
      new RegExp(
        `A channel named ${cmdChannelName} already exists\\. Give the channel ` +
          "another name\\.",
      ),
    );
  });

  it("should reject a default command name that a live slave's channel holds", async () => {
    const base = createIdentifier();
    const deploy = async (
      identifier: string,
      entry: ethercat.PDOEntry,
      overrides: Partial<ethercat.EntryWriteChannel> = {},
    ) => {
      const slave = await createSlaveDevice(client, testRack.key, {
        identifier,
        network: "eth0",
        pdos: { inputs: [], outputs: [entry] },
      });
      return await renderWrite({
        ...EtherCAT.Task.WRITE_SCHEMAS.config.parse({}),
        channels: [
          { ...EtherCAT.Task.createEntryWriteChannel(slave.key, entry), ...overrides },
        ],
      });
    };
    const owner = await deploy(base, { ...CONTROL_ENTRY, name: "x Control" });
    await deployAndAwaitTask(
      client,
      owner.container,
      owner.draft.key,
      EtherCAT.Task.WRITE_SCHEMAS,
    );
    owner.unmount();
    const { container, statuses } = await deploy(`${base}_x`, CONTROL_ENTRY, {
      stateChannelName: uniqueName("ecat_state"),
    });
    await clickDeploy(container);
    await awaitStatus(
      statuses,
      new RegExp(
        `A channel named ${base}_x_Control_cmd already exists\\. Give the channel ` +
          "another name\\.",
      ),
    );
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
        EtherCAT.Task.createEntryWriteChannel(slaveA.key, CONTROL_ENTRY),
        EtherCAT.Task.createEntryWriteChannel(slaveB.key, CONTROL_ENTRY),
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
      channels: [EtherCAT.Task.createEntryWriteChannel(slave.key, CONTROL_ENTRY)],
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

  it("should create the task from a slave with a channel for each output", async () => {
    const slave = await createSlaveDevice(client, testRack.key, {
      identifier: createIdentifier(),
      network: "eth0",
      pdos: createPDOs(),
    });
    const tsk = await createWithHook(
      EtherCAT.Task.useCreateWrite,
      EtherCAT.Task.WRITE_SCHEMAS,
      { deviceKey: slave.key },
    );
    const addresses = tsk.config.channels.map((ch) =>
      ch.type === "entry"
        ? { device: ch.device, index: ch.index, subIndex: ch.subIndex }
        : ch.type,
    );
    expect(addresses).toEqual(
      slave.properties.pdos.outputs.map(({ index, subIndex }) => ({
        device: slave.key,
        index,
        subIndex,
      })),
    );
  });
});

describe("Write channel names", () => {
  it("should show the names deploying gives the command and state channels", async () => {
    const identifier = createIdentifier();
    const slave = await createSlaveDevice(client, testRack.key, {
      identifier,
      network: "eth0",
      pdos: createPDOs(),
    });
    await renderWrite({
      ...EtherCAT.Task.WRITE_SCHEMAS.config.parse({}),
      channels: [EtherCAT.Task.createEntryWriteChannel(slave.key, CONTROL_ENTRY)],
    });
    const base = channel.escapeInvalidName(`${identifier}_Control`);
    const item = await findChannelListItem(`${base}_cmd`);
    expect(within(item).getByText(`${base}_state`)).toBeTruthy();
  });
});
