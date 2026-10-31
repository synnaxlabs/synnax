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
  createEntryReadChannel,
  createIdentifier,
  createPDOs,
  createRepeatedNamePDOs,
  createRepeatedNameSlave,
  createSlaveDevice,
  createUnresolvedReadChannel,
  STATUS_ENTRY,
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
const ZERO_DRAFT: task.New<EtherCAT.Task.ReadSchemas> = {
  name: "EtherCAT read task",
  type: EtherCAT.Task.READ_TYPE,
  config: EtherCAT.Task.READ_SCHEMAS.config.parse({}),
};

const createDraft = async (
  client: Synnax,
  config: z.infer<EtherCAT.Task.ReadSchemas["config"]>,
) => await client.tasks.create({ ...ZERO_DRAFT, config }, EtherCAT.Task.READ_SCHEMAS);

// The form renders read-only until the update grant lands, and a preview field renders
// no input, so wait for it to become editable before querying fields.
const renderRead = async (config: z.infer<EtherCAT.Task.ReadSchemas["config"]>) => {
  const draft = await createDraft(client, config);
  const statuses: Status.NotificationSpec[] = [];
  const rendered = await renderTaskFormTab(EtherCAT.Task.Read, {
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

describe("Read", () => {
  it("should render channels from the task row's config with their port labels", async () => {
    const slave = await createSlaveDevice(client, testRack.key, {
      identifier: createIdentifier(),
      network: "eth0",
      pdos: createPDOs(),
    });
    await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [
        createEntryReadChannel(slave.key, STATUS_ENTRY),
        createUnresolvedReadChannel(slave.key, "Input"),
      ],
    });
    expect(await screen.findByRole("option", { name: /0x6000:01/ })).toBeTruthy();
    expect(screen.getByRole("option", { name: /^Input/ })).toBeTruthy();
  });

  it("should show the entry picker and data type when a channel is selected", async () => {
    const slave = await createRepeatedNameSlave(client, testRack.key);
    const { inputs } = createRepeatedNamePDOs();
    await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [createEntryReadChannel(slave.key, inputs[1])],
    });
    fireEvent.click(await screen.findByRole("option", { name: /0x6010:01/ }));
    await waitFor(() => expect(screen.getByText("Slave device")).toBeTruthy());
    expect(await screen.findByText("Channel 2 › Input · 0x6010:01")).toBeTruthy();
    expect(screen.getByRole("button", { name: /^Entry/ })).toBeTruthy();
    expect(screen.getByText("Data type")).toBeTruthy();
    expect(screen.queryByText("Mode")).toBeNull();
  });

  it("should list each of eight repeated entry names apart in the picker", async () => {
    const slave = await createRepeatedNameSlave(client, testRack.key);
    const { inputs } = createRepeatedNamePDOs();
    await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [createEntryReadChannel(slave.key, inputs[0])],
    });
    fireEvent.click(await screen.findByRole("option", { name: /0x6000:01/ }));
    fireEvent.click(await screen.findByRole("button", { name: /^Entry/ }));
    for (let i = 0; i < 8; i++)
      expect(
        await screen.findByRole("option", {
          name: new RegExp(`Channel ${i + 1} › Input`),
        }),
      ).toBeTruthy();
    expect(screen.getAllByText("0x6070:01")).toHaveLength(1);
  });

  it("should list an address that two PDOs map only once", async () => {
    const value = { ...STATUS_ENTRY, name: "Value", pdoName: "Channel 1" };
    const slave = await createSlaveDevice(client, testRack.key, {
      identifier: createIdentifier(),
      network: "eth0",
      pdos: { inputs: [value, { ...value, pdoName: "Compact" }], outputs: [] },
    });
    await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [createEntryReadChannel(slave.key, value)],
    });
    fireEvent.click(await screen.findByRole("option", { name: /0x6000:01/ }));
    fireEvent.click(await screen.findByRole("button", { name: /^Entry/ }));
    await waitFor(() =>
      expect(screen.getAllByRole("option", { name: /Value/ })).toHaveLength(1),
    );
    expect(screen.queryByText("Compact › Value")).toBeNull();
  });

  it("should set the address and data type of the entry picked", async () => {
    const slave = await createRepeatedNameSlave(client, testRack.key);
    const { inputs } = createRepeatedNamePDOs();
    const { draft } = await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [createEntryReadChannel(slave.key, inputs[0], { dataType: "uint16" })],
    });
    fireEvent.click(await screen.findByRole("option", { name: /0x6000:01/ }));
    fireEvent.click(await screen.findByRole("button", { name: /^Entry/ }));
    fireEvent.click(await screen.findByRole("option", { name: /Channel 3 › Input/ }));
    await waitFor(async () => {
      const saved = await client.tasks.retrieve({
        key: draft.key,
        schemas: EtherCAT.Task.READ_SCHEMAS,
      });
      expect(saved.config.channels[0]).toMatchObject({
        type: "entry",
        index: 0x6020,
        subIndex: 1,
        dataType: "uint8",
      });
    });
  });

  it("should show why an unresolved channel cannot run", async () => {
    const slave = await createRepeatedNameSlave(client, testRack.key);
    await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [createUnresolvedReadChannel(slave.key, "Input")],
    });
    fireEvent.click(await screen.findByRole("option", { name: /Input/ }));
    const addresses = Array.from(
      { length: 8 },
      (_, i) => `0x60${i}0:01 (Channel ${i + 1})`,
    ).join(", ");
    expect(
      await screen.findByText(
        `'Input' matches more than one input entry on slave ${slave.name}: ` +
          `${addresses}. Pick an entry.`,
      ),
    ).toBeTruthy();
    expect(screen.queryByText("Data type")).toBeNull();
  });

  it("should turn an unresolved channel into an entry when one is picked", async () => {
    const slave = await createRepeatedNameSlave(client, testRack.key);
    const { draft } = await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [createUnresolvedReadChannel(slave.key, "Input")],
    });
    fireEvent.click(await screen.findByRole("option", { name: /Input/ }));
    fireEvent.click(await screen.findByText("Select entry"));
    fireEvent.click(await screen.findByRole("option", { name: /Channel 3 › Input/ }));
    await waitFor(async () => {
      const saved = await client.tasks.retrieve({
        key: draft.key,
        schemas: EtherCAT.Task.READ_SCHEMAS,
      });
      const [ch] = saved.config.channels;
      expect(ch).toMatchObject({ type: "entry", index: 0x6020, subIndex: 1 });
      expect("pdo" in ch).toBe(false);
    });
    expect(await screen.findByText("Data type")).toBeTruthy();
  });

  it("should title a channel by its address when its slave lacks the entry", async () => {
    const slave = await createSlaveDevice(client, testRack.key, {
      identifier: createIdentifier(),
      network: "eth0",
      pdos: createPDOs(),
    });
    await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [createEntryReadChannel(slave.key, { ...STATUS_ENTRY, subIndex: 2 })],
    });
    fireEvent.click(await screen.findByRole("option", { name: /0x6000:02/ }));
    await waitFor(() => expect(screen.getAllByText("0x6000:02")).toHaveLength(2));
    expect(screen.queryByText(/· 0x6000:02/)).toBeNull();
  });

  it("should ask for a slave when the picker opens with none selected", async () => {
    await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [EtherCAT.Task.createReadChannel([])],
    });
    fireEvent.click(await screen.findByRole("option", { name: /No entry/ }));
    fireEvent.click(await screen.findByText("Select entry"));
    expect(await screen.findByText("Select a slave device first.")).toBeTruthy();
  });

  it("should ask for a scan when the slave has no input entries", async () => {
    const slave = await createSlaveDevice(client, testRack.key, {
      identifier: createIdentifier(),
      network: "eth0",
      pdos: { inputs: [], outputs: createPDOs().outputs },
    });
    await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [{ ...EtherCAT.Task.createReadChannel([]), device: slave.key }],
    });
    fireEvent.click(await screen.findByRole("option", { name: /No entry/ }));
    fireEvent.click(await screen.findByText("Select entry"));
    expect(
      await screen.findByText(
        "This slave has no input entries. Scan the network again.",
      ),
    ).toBeTruthy();
  });

  it("should create the index and data channels, update the slave, and save the task", async () => {
    const identifier = createIdentifier();
    const namedChannel = uniqueName("ecat_named");
    const second = { ...STATUS_ENTRY, name: "Counter", subIndex: 2 };
    const slave = await createSlaveDevice(client, testRack.key, {
      identifier,
      network: "eth0",
      pdos: { inputs: [STATUS_ENTRY, second], outputs: [] },
    });
    const { container, draft } = await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [
        createEntryReadChannel(slave.key, STATUS_ENTRY),
        createEntryReadChannel(slave.key, second, { name: namedChannel }),
      ],
    });
    const created = await deployAndAwaitTask(
      client,
      container,
      draft.key,
      EtherCAT.Task.READ_SCHEMAS,
    );
    expect(created.type).toBe(EtherCAT.Task.READ_TYPE);
    expect(created.rack).toBe(testRack.key);
    const [status, named] = created.config.channels;
    expect(status.channel).not.toBe(0);
    expect(named.channel).not.toBe(0);

    const statusChannel = await client.channels.retrieve(status.channel);
    expect(statusChannel.name).toBe(`${identifier}_Status`);
    expect(statusChannel.dataType.toString()).toBe("uint16");
    const namedRetrieved = await client.channels.retrieve(named.channel);
    expect(namedRetrieved.name).toBe(namedChannel);

    const updated = await client.devices.retrieve({
      key: slave.key,
      schemas: EtherCAT.Device.SLAVE_SCHEMAS,
    });
    expect(updated.properties.readIndex).not.toBe(0);
    expect(updated.properties.read.channels[`${0x6000}_1`]).toBe(status.channel);
    expect(updated.properties.read.channels[`${0x6000}_2`]).toBe(named.channel);
    const index = await client.channels.retrieve(updated.properties.readIndex);
    expect(index.name).toBe(`${identifier}_time`);
    expect(index.isIndex).toBe(true);
  });

  it("should create one channel for each of eight entries that share a name", async () => {
    const slave = await createRepeatedNameSlave(client, testRack.key);
    const { identifier } = slave.properties;
    const { inputs } = createRepeatedNamePDOs();
    const { container, draft } = await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: inputs.map((e) => createEntryReadChannel(slave.key, e)),
    });
    const created = await deployAndAwaitTask(
      client,
      container,
      draft.key,
      EtherCAT.Task.READ_SCHEMAS,
    );
    const keys = created.config.channels.map(({ channel }) => channel);
    expect(new Set(keys).size).toBe(8);
    const channels = await client.channels.retrieve(keys);
    expect(channels.map(({ name }) => name).sort()).toEqual(
      inputs.map((_, i) => `${identifier}_Channel_${i + 1}_Input`).sort(),
    );
    const updated = await client.devices.retrieve({
      key: slave.key,
      schemas: EtherCAT.Device.SLAVE_SCHEMAS,
    });
    inputs.forEach((e, i) =>
      expect(updated.properties.read.channels[`${e.index}_1`]).toBe(keys[i]),
    );
  });

  it("should block configuration when two channels use the same entry", async () => {
    const slave = await createRepeatedNameSlave(client, testRack.key);
    const { inputs } = createRepeatedNamePDOs();
    const { container, statuses } = await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [
        createEntryReadChannel(slave.key, inputs[3]),
        createEntryReadChannel(slave.key, inputs[3]),
      ],
    });
    await clickDeploy(container);
    await awaitStatus(
      statuses,
      new RegExp(
        `More than one channel uses entry 0x6030:01 on slave ${slave.name}\\. ` +
          "Pick a different entry for one of them\\.$",
      ),
    );
  });

  it("should block configuration of an entry the slave does not have", async () => {
    const slave = await createSlaveDevice(client, testRack.key, {
      identifier: createIdentifier(),
      network: "eth0",
      pdos: createPDOs(),
    });
    // The address of an entry picked on another slave before the slave was switched.
    const { inputs } = createRepeatedNamePDOs();
    const { container, statuses } = await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [createEntryReadChannel(slave.key, inputs[3])],
    });
    await clickDeploy(container);
    await awaitStatus(
      statuses,
      new RegExp(
        `Slave ${slave.name} has no input entry at 0x6030:01\\. Pick an entry\\.$`,
      ),
    );
    const updated = await client.devices.retrieve({
      key: slave.key,
      schemas: EtherCAT.Device.SLAVE_SCHEMAS,
    });
    expect(updated.properties.readIndex).toBe(0);
  });

  it("should configure a task whose unresolved channel is disabled", async () => {
    const slave = await createRepeatedNameSlave(client, testRack.key);
    const { inputs } = createRepeatedNamePDOs();
    const { container, draft } = await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [
        createEntryReadChannel(slave.key, inputs[0]),
        createUnresolvedReadChannel(slave.key, "Input", { disabled: true }),
      ],
    });
    const created = await deployAndAwaitTask(
      client,
      container,
      draft.key,
      EtherCAT.Task.READ_SCHEMAS,
    );
    const [bound, disabled] = created.config.channels;
    expect(bound.channel).not.toBe(0);
    expect(disabled).toMatchObject({ type: "unresolved", pdo: "Input", channel: 0 });
  });

  it("should block configuration of an unresolved channel", async () => {
    const slave = await createRepeatedNameSlave(client, testRack.key);
    const { container, statuses } = await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [createUnresolvedReadChannel(slave.key, "Input")],
    });
    await clickDeploy(container);
    await awaitStatus(
      statuses,
      /'Input' matches more than one input entry on slave .+\. Pick an entry\.$/,
    );
  });

  it("should block configuration of a channel with no entry", async () => {
    const slave = await createRepeatedNameSlave(client, testRack.key);
    const { container, statuses } = await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [{ ...EtherCAT.Task.createReadChannel([]), device: slave.key }],
    });
    await clickDeploy(container);
    await awaitStatus(
      statuses,
      new RegExp(`A channel on slave ${slave.name} has no entry\\. Pick an entry\\.$`),
    );
  });

  it("should bind a new entry to the channel the device already maps", async () => {
    const slave = await createSlaveDevice(client, testRack.key, {
      identifier: createIdentifier(),
      network: "eth0",
      pdos: createPDOs(),
    });
    const config = {
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [createEntryReadChannel(slave.key, STATUS_ENTRY)],
    };
    const first = await renderRead(config);
    const firstTask = await deployAndAwaitTask(
      client,
      first.container,
      first.draft.key,
      EtherCAT.Task.READ_SCHEMAS,
    );
    first.unmount();
    const existing = await client.channels.retrieve(
      firstTask.config.channels[0].channel,
    );
    const second = await renderRead(config);
    await screen.findByText(existing.name);
    await waitFor(async () => {
      const saved = await client.tasks.retrieve({
        key: second.draft.key,
        schemas: EtherCAT.Task.READ_SCHEMAS,
      });
      expect(saved.config.channels[0].channel).toBe(existing.key);
    });
  });

  it("should reuse the existing index and channels when redeployed", async () => {
    const slave = await createSlaveDevice(client, testRack.key, {
      identifier: createIdentifier(),
      network: "eth0",
      pdos: createPDOs(),
    });
    const config = {
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [createEntryReadChannel(slave.key, STATUS_ENTRY)],
    };
    const first = await renderRead(config);
    const firstTask = await deployAndAwaitTask(
      client,
      first.container,
      first.draft.key,
      EtherCAT.Task.READ_SCHEMAS,
    );
    first.unmount();

    const second = await renderRead(config);
    const secondTask = await deployAndAwaitTask(
      client,
      second.container,
      second.draft.key,
      EtherCAT.Task.READ_SCHEMAS,
    );
    expect(secondTask.config.channels[0].channel).toBe(
      firstTask.config.channels[0].channel,
    );
  });

  it("should reject a data type that differs from the mapped channel's", async () => {
    const slave = await createSlaveDevice(client, testRack.key, {
      identifier: createIdentifier(),
      network: "eth0",
      pdos: createPDOs(),
    });
    const render = async (dataType: string) =>
      await renderRead({
        ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
        channels: [createEntryReadChannel(slave.key, STATUS_ENTRY, { dataType })],
      });
    const first = await render("uint16");
    await deployAndAwaitTask(
      client,
      first.container,
      first.draft.key,
      EtherCAT.Task.READ_SCHEMAS,
    );
    first.unmount();
    const { container, statuses } = await render("int16");
    await clickDeploy(container);
    await awaitStatus(statuses, /holds uint16 data, not int16/);
  });

  it("should reject a data type that does not fit the entry before creating a channel", async () => {
    const slave = await createSlaveDevice(client, testRack.key, {
      identifier: createIdentifier(),
      network: "eth0",
      pdos: createPDOs(),
    });
    const { container, statuses } = await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [
        createEntryReadChannel(slave.key, STATUS_ENTRY, { dataType: "float64" }),
      ],
    });
    await clickDeploy(container);
    await awaitStatus(
      statuses,
      /Data type float64 does not fit entry 0x6000:01 on slave .+, which holds 16 bits\. Pick a 2 byte number type\./,
    );
    const updated = await client.devices.retrieve({
      key: slave.key,
      schemas: EtherCAT.Device.SLAVE_SCHEMAS,
    });
    expect(updated.properties.read.channels).toEqual({});
  });

  it("should mark the channel list when the task has no channels", async () => {
    const { container } = await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [],
    });
    await clickDeploy(container);
    expect(
      await screen.findByText("At least one enabled channel is required"),
    ).toBeTruthy();
  });

  it("should surface an error when a slave is not configured", async () => {
    const slave = await createSlaveDevice(
      client,
      testRack.key,
      { identifier: createIdentifier(), network: "eth0", pdos: createPDOs() },
      false,
    );
    const { container, statuses } = await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [createEntryReadChannel(slave.key, STATUS_ENTRY)],
    });
    await clickDeploy(container);
    await awaitStatus(statuses, /Failed to/);
    await awaitStatus(statuses, /is not configured/);
  });

  it("should surface an error when a slave no longer exists", async () => {
    const createSlave = async () =>
      await createSlaveDevice(client, testRack.key, {
        identifier: createIdentifier(),
        network: "eth0",
        pdos: createPDOs(),
      });
    const slave = await createSlave();
    const deleted = await createSlave();
    await client.devices.delete(deleted.key);
    const { container, statuses } = await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [
        createEntryReadChannel(slave.key, STATUS_ENTRY),
        createEntryReadChannel(deleted.key, STATUS_ENTRY),
      ],
    });
    await clickDeploy(container);
    await awaitStatus(
      statuses,
      new RegExp(`^Found no device with key ${deleted.key}$`),
    );
  });

  it("should create the task with one channel", async () => {
    const tsk = await createWithHook(
      EtherCAT.Task.useCreateRead,
      EtherCAT.Task.READ_SCHEMAS,
    );
    expect(tsk.config.channels).toHaveLength(1);
  });
});
