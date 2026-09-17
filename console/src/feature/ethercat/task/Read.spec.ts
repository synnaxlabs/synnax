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
import { type Status } from "@synnaxlabs/pluto";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeAll, describe, expect, it } from "vitest";

import { EtherCAT } from "@/feature/ethercat";
import {
  createAutoReadChannel,
  createIdentifier,
  createManualReadChannel,
  createPDOs,
  createSlaveDevice,
} from "@/feature/ethercat/testutil";
import {
  clickDeploy,
  createTestChannel,
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
  config: EtherCAT.Task.ReadPayload["config"],
) => await client.tasks.create({ ...ZERO_DRAFT, config }, EtherCAT.Task.READ_SCHEMAS);

const renderRead = async (config: EtherCAT.Task.ReadPayload["config"]) => {
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

describe("EtherCAT Read", () => {
  it("should render channels from the task row's config with their port labels", async () => {
    const slave = await createSlaveDevice(client, testRack.key, {
      identifier: createIdentifier(),
      network: "eth0",
      pdos: createPDOs(),
    });
    await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [
        createAutoReadChannel(slave.key, "Status"),
        createManualReadChannel(slave.key, 0x6000, 5),
      ],
    });
    await waitFor(() =>
      expect(screen.getAllByText("Status").length).toBeGreaterThan(0),
    );
    expect(screen.getByText("0x6000:5")).toBeTruthy();
  });

  it("should show the automatic channel detail fields when it is selected", async () => {
    const slave = await createSlaveDevice(client, testRack.key, {
      identifier: createIdentifier(),
      network: "eth0",
      pdos: createPDOs(),
    });
    await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [createAutoReadChannel(slave.key, "Status")],
    });
    fireEvent.click((await screen.findAllByText("Status"))[0]);
    await waitFor(() => expect(screen.getByText("Slave device")).toBeTruthy());
    expect(screen.getByText("Mode")).toBeTruthy();
    expect(screen.getByText("PDO")).toBeTruthy();
    expect(screen.queryByText("Index (hex)")).toBeNull();
  });

  it("should show the manual address fields, including subindex, when a manual channel is selected", async () => {
    const slave = await createSlaveDevice(client, testRack.key, {
      identifier: createIdentifier(),
      network: "eth0",
    });
    await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [createManualReadChannel(slave.key, 0x6000, 7)],
    });
    fireEvent.click(await screen.findByText("0x6000:7"));
    await waitFor(() => expect(screen.getByText("Index (hex)")).toBeTruthy());
    expect(screen.getByText("Subindex")).toBeTruthy();
    expect(screen.getByDisplayValue("7")).toBeTruthy();
    expect(screen.getByText("Bit length")).toBeTruthy();
    expect(screen.getByText("Data type")).toBeTruthy();
    expect(screen.queryByText("PDO")).toBeNull();
  });

  it("should swap a channel between automatic and manual modes preserving its slave", async () => {
    const slave = await createSlaveDevice(client, testRack.key, {
      identifier: createIdentifier(),
      network: "eth0",
      pdos: createPDOs(),
    });
    await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [createAutoReadChannel(slave.key, "Status")],
    });
    fireEvent.click((await screen.findAllByText("Status"))[0]);
    fireEvent.click(await screen.findByText("Automatic (PDO)"));
    fireEvent.click(await screen.findByText("Manual (Address)"));
    await waitFor(() => expect(screen.getByText("Index (hex)")).toBeTruthy());
    expect(screen.getByText("Subindex")).toBeTruthy();
    await waitFor(() => expect(screen.getByText(slave.name)).toBeTruthy());
  });

  describe("deploying against a live Core", () => {
    it("should create the index and data channels, update the slave, and save the task", async () => {
      const identifier = createIdentifier();
      const namedChannel = uniqueName("ecat_named");
      const slave = await createSlaveDevice(client, testRack.key, {
        identifier,
        network: "eth0",
        pdos: createPDOs(),
      });
      const { container, draft } = await renderRead({
        ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
        channels: [
          createAutoReadChannel(slave.key, "Status"),
          createManualReadChannel(slave.key, 0x6001, 2, { name: namedChannel }),
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
      const [auto, manual] = created.config.channels;
      expect(auto.channel).not.toBe(0);
      expect(manual.channel).not.toBe(0);

      const autoChannel = await client.channels.retrieve(auto.channel);
      expect(autoChannel.name).toBe(`${identifier}_Status`);
      expect(autoChannel.dataType.toString()).toBe("uint16");
      const named = await client.channels.retrieve(manual.channel);
      expect(named.name).toBe(namedChannel);

      const updated = await client.devices.retrieve({
        key: slave.key,
        schemas: EtherCAT.Device.SLAVE_SCHEMAS,
      });
      expect(updated.properties.readIndex).not.toBe(0);
      expect(updated.properties.read.channels.auto_Status).toBe(auto.channel);
      expect(updated.properties.read.channels[`manual_${0x6001}_2`]).toBe(
        manual.channel,
      );
      const index = await client.channels.retrieve(updated.properties.readIndex);
      expect(index.name).toBe(`${identifier}_time`);
      expect(index.isIndex).toBe(true);
    });

    it("should reuse the existing index and channels when redeployed", async () => {
      const slave = await createSlaveDevice(client, testRack.key, {
        identifier: createIdentifier(),
        network: "eth0",
        pdos: createPDOs(),
      });
      const config = {
        ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
        channels: [createAutoReadChannel(slave.key, "Status")],
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

    it("should surface an error when the task has no channels", async () => {
      const { container, statuses } = await renderRead({
        ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
        channels: [],
      });
      await clickDeploy(container);
      await awaitStatus(statuses, /Failed to/);
      await awaitStatus(statuses, /No channels configured/);
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
        channels: [createAutoReadChannel(slave.key, "Status")],
      });
      await clickDeploy(container);
      await awaitStatus(statuses, /Failed to/);
      await awaitStatus(statuses, /is not configured/);
    });
  });
});

describe("EtherCAT Read device map binding", () => {
  const createSlave = async (channels: Record<string, number> = {}) =>
    await createSlaveDevice(client, testRack.key, {
      identifier: createIdentifier(),
      network: "eth0",
      pdos: createPDOs(),
      read: { channels },
    });

  it("should show the channel the slave map binds to a row that holds none", async () => {
    const bound = await createTestChannel(client, "ecat");
    const slave = await createSlave({ auto_Status: bound.key });
    await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [createAutoReadChannel(slave.key, "Status")],
    });
    await screen.findByText(bound.name);
  });

  it("should save the bound channel into the task", async () => {
    const bound = await createTestChannel(client, "ecat");
    const slave = await createSlave({ auto_Status: bound.key });
    const { draft } = await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [createAutoReadChannel(slave.key, "Status")],
    });
    await screen.findByText(bound.name);
    await waitFor(async () => {
      const saved = await client.tasks.retrieve({
        key: draft.key,
        schemas: EtherCAT.Task.READ_SCHEMAS,
      });
      expect(saved.config.channels[0].channel).toBe(bound.key);
    });
  });

  it("should drop a row's stale channel when the slave map has no entry for its PDO", async () => {
    const stale = await createTestChannel(client, "ecat");
    const slave = await createSlave();
    await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [createAutoReadChannel(slave.key, "Status", { channel: stale.key })],
    });
    await screen.findByText("No channel");
    expect(screen.queryByText(stale.name)).toBeNull();
  });

  it("should keep a row's channel while it names no slave", async () => {
    const own = await createTestChannel(client, "ecat");
    await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [createAutoReadChannel("", "Status", { channel: own.key })],
    });
    await screen.findByText(own.name);
  });

  it("should bind each row from its own slave", async () => {
    const boundA = await createTestChannel(client, "ecat_a");
    const boundB = await createTestChannel(client, "ecat_b");
    const slaveA = await createSlave({ auto_Status: boundA.key });
    const slaveB = await createSlave({ auto_Status: boundB.key });
    await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [
        createAutoReadChannel(slaveA.key, "Status"),
        createAutoReadChannel(slaveB.key, "Status"),
      ],
    });
    await screen.findByText(boundA.name);
    await screen.findByText(boundB.name);
  });
});
