// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type rack } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeAll, describe, expect, it } from "vitest";

import { EtherCAT } from "@/feature/ethercat";
import {
  createAutoInputChannel,
  createIdentifier,
  createManualInputChannel,
  createPDOs,
  createSlaveDevice,
} from "@/feature/ethercat/testutil";
import {
  awaitTaskKey,
  clickConfigure,
  renderTaskFormTab,
} from "@/platform/task/testutil";
import { stubGeometry, uniqueName } from "@/testutil";

const client = createTestClient();

stubGeometry();

let testRack: rack.Rack;

beforeAll(async () => {
  testRack = await client.racks.create({ name: uniqueName("ecat_rack") });
});

const renderRead = async (config?: unknown) =>
  await renderTaskFormTab(EtherCAT.Task.Read, EtherCAT.Task.READ_TYPE, {
    client,
    args: config == null ? {} : { config },
  });

describe("EtherCAT Read", () => {
  it("should render channels parsed from an args config with their port labels", async () => {
    const slave = await createSlaveDevice(client, testRack.key, {
      identifier: createIdentifier(),
      network: "eth0",
      pdos: createPDOs(),
    });
    await renderRead({
      ...EtherCAT.Task.ZERO_READ_PAYLOAD.config,
      channels: [
        createAutoInputChannel(slave.key, "Status"),
        createManualInputChannel(slave.key, 0x6000, 5),
      ],
    });
    await waitFor(() => expect(screen.getByText("Status")).toBeTruthy());
    expect(screen.getByText("0x6000:5")).toBeTruthy();
  });

  it("should show the automatic channel detail fields when it is selected", async () => {
    const slave = await createSlaveDevice(client, testRack.key, {
      identifier: createIdentifier(),
      network: "eth0",
      pdos: createPDOs(),
    });
    await renderRead({
      ...EtherCAT.Task.ZERO_READ_PAYLOAD.config,
      channels: [createAutoInputChannel(slave.key, "Status")],
    });
    fireEvent.click(await screen.findByText("Status"));
    await waitFor(() => expect(screen.getByText("Slave Device")).toBeTruthy());
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
      ...EtherCAT.Task.ZERO_READ_PAYLOAD.config,
      channels: [createManualInputChannel(slave.key, 0x6000, 7)],
    });
    fireEvent.click(await screen.findByText("0x6000:7"));
    await waitFor(() => expect(screen.getByText("Index (hex)")).toBeTruthy());
    expect(screen.getByText("Subindex")).toBeTruthy();
    expect(screen.getByDisplayValue("7")).toBeTruthy();
    expect(screen.getByText("Bit Length")).toBeTruthy();
    expect(screen.getByText("Data Type")).toBeTruthy();
    expect(screen.queryByText("PDO")).toBeNull();
  });

  it("should swap a channel between automatic and manual modes preserving its slave", async () => {
    const slave = await createSlaveDevice(client, testRack.key, {
      identifier: createIdentifier(),
      network: "eth0",
      pdos: createPDOs(),
    });
    await renderRead({
      ...EtherCAT.Task.ZERO_READ_PAYLOAD.config,
      channels: [createAutoInputChannel(slave.key, "Status")],
    });
    fireEvent.click(await screen.findByText("Status"));
    fireEvent.click(await screen.findByText("Automatic (PDO)"));
    fireEvent.click(await screen.findByText("Manual (Address)"));
    await waitFor(() => expect(screen.getByText("Index (hex)")).toBeTruthy());
    expect(screen.getByText("Subindex")).toBeTruthy();
    await waitFor(() => expect(screen.getByText(slave.name)).toBeTruthy());
  });

  describe("configure against a live cluster", () => {
    it("should create the index and data channels, update the slave, and save the task", async () => {
      const identifier = createIdentifier();
      const namedChannel = uniqueName("ecat_named");
      const slave = await createSlaveDevice(client, testRack.key, {
        identifier,
        network: "eth0",
        pdos: createPDOs(),
      });
      const rendered = await renderRead({
        ...EtherCAT.Task.ZERO_READ_PAYLOAD.config,
        channels: [
          createAutoInputChannel(slave.key, "Status"),
          createManualInputChannel(slave.key, 0x6001, 2, { name: namedChannel }),
        ],
      });
      await clickConfigure();
      const taskKey = await awaitTaskKey(rendered);
      const created = await client.tasks.retrieve({
        key: taskKey,
        schemas: EtherCAT.Task.READ_SCHEMAS,
      });
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

    it("should reuse the existing index and channels when reconfigured", async () => {
      const slave = await createSlaveDevice(client, testRack.key, {
        identifier: createIdentifier(),
        network: "eth0",
        pdos: createPDOs(),
      });
      const config = {
        ...EtherCAT.Task.ZERO_READ_PAYLOAD.config,
        channels: [createAutoInputChannel(slave.key, "Status")],
      };
      const first = await renderRead(config);
      await clickConfigure();
      const firstKey = await awaitTaskKey(first);
      const firstTask = await client.tasks.retrieve({
        key: firstKey,
        schemas: EtherCAT.Task.READ_SCHEMAS,
      });
      first.unmount();

      const second = await renderRead(config);
      await clickConfigure();
      const secondKey = await awaitTaskKey(second);
      const secondTask = await client.tasks.retrieve({
        key: secondKey,
        schemas: EtherCAT.Task.READ_SCHEMAS,
      });
      expect(secondTask.config.channels[0].channel).toBe(
        firstTask.config.channels[0].channel,
      );
    });

    it("should surface an error when the task has no channels", async () => {
      await renderRead();
      await clickConfigure();
      fireEvent.click(await screen.findByText(/Failed to/));
      await waitFor(() =>
        expect(screen.getByText(/No channels configured/)).toBeTruthy(),
      );
    });

    it("should surface an error when a slave is not configured", async () => {
      const slave = await createSlaveDevice(
        client,
        testRack.key,
        { identifier: createIdentifier(), network: "eth0", pdos: createPDOs() },
        false,
      );
      await renderRead({
        ...EtherCAT.Task.ZERO_READ_PAYLOAD.config,
        channels: [createAutoInputChannel(slave.key, "Status")],
      });
      await clickConfigure();
      fireEvent.click(await screen.findByText(/Failed to/));
      await waitFor(() => expect(screen.getByText(/is not configured/)).toBeTruthy());
    });
  });
});
