// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { device as deviceClient, NotFoundError, query } from "@synnaxlabs/client";
import { createTestClient, RoleClients } from "@synnaxlabs/client/testutil";
import { type Status } from "@synnaxlabs/lyra/status";
import { TimeStamp } from "@synnaxlabs/x";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { describe, expect, it } from "vitest";

import { Device } from "@/feature/device";
import { createIdentifier, createSlaveDevice } from "@/feature/ethercat/testutil";
import { createModbusDevice } from "@/feature/modbus/testutil";
import { NI } from "@/feature/ni";
import { OPCUA } from "@/feature/opcua";
import { createTestDevice } from "@/platform/device/testutil";
import { findButton } from "@/platform/modals/testutil";
import { Tree } from "@/platform/tree";
import {
  createHostItems,
  createResource,
  expandTreeRow,
  findTreeRow,
  HOST_ID,
  openTreeRowContextMenu,
  renderLiveTree,
  renderTreeContextMenu,
} from "@/platform/tree/testutil";
import {
  assertDefined,
  awaitTextEditingElement,
  CaptureStatuses,
  commitTextEdit,
  uniqueName,
} from "@/testutil";

const client = createTestClient();
const roles = new RoleClients(client);

// Hosts the devices under the host root, outside any rack or group.
const hostDevices = (...devices: { key: string }[]) =>
  createHostItems((c) => [
    Tree.createSource(
      c.devices,
      query.keysRequest(devices.map((d) => d.key)),
      Tree.toResource("device"),
    ),
  ]);

const renderDeviceTree = async (
  host: ReturnType<typeof hostDevices>,
  extra?: ReactNode,
) =>
  await renderLiveTree({
    client,
    root: HOST_ID,
    items: { ...Device.TREE_ITEMS, ...host },
    extra,
  });

describe("device/tree", () => {
  it("renders the device with its location", async () => {
    const dev = await createTestDevice(client);
    const host = hostDevices(dev);
    await renderDeviceTree(host);
    expect(await screen.findByText(dev.name)).toBeTruthy();
    expect(await screen.findByText("dev1")).toBeTruthy();
  });

  it("lists the devices that a device holds", async () => {
    const chassis = await createTestDevice(client);
    const module = await client.devices.create({
      key: uniqueName("module"),
      name: uniqueName("module"),
      rack: chassis.rack,
      location: "slot1",
      make: "make",
      model: "model",
      properties: {},
      parent: chassis.key,
    });
    await renderDeviceTree(hostDevices(chassis));
    await findTreeRow(chassis.name);
    expandTreeRow(chassis.name);
    expect(await findTreeRow(module.name)).toBeTruthy();
  });

  it("shows a status change of a device that a device holds", async () => {
    const chassis = await createTestDevice(client);
    const module = await client.devices.create({
      key: uniqueName("module"),
      name: uniqueName("module"),
      rack: chassis.rack,
      location: "slot1",
      make: "make",
      model: "model",
      properties: {},
      parent: chassis.key,
    });
    await renderDeviceTree(hostDevices(chassis));
    await findTreeRow(chassis.name);
    expandTreeRow(chassis.name);
    const indicator = async () =>
      (await findTreeRow(module.name)).querySelector(".pluto-device__status-indicator")
        ?.outerHTML;
    const before = await indicator();
    expect(before).toBeDefined();
    await client.statuses.set({
      owner: deviceClient.resourceID(module.key),
      name: "",
      variant: "error",
      message: uniqueName("message"),
      time: TimeStamp.now(),
    });
    await waitFor(async () => expect(await indicator()).not.toBe(before));
  });

  it("renames a device from the context menu", async () => {
    const dev = await createTestDevice(client);
    const host = hostDevices(dev);
    await renderDeviceTree(host);
    await openTreeRowContextMenu(dev.name);
    fireEvent.click(await screen.findByText("Rename"));
    const editor = await awaitTextEditingElement();
    const renamed = uniqueName("renamed");
    commitTextEdit(editor, renamed);
    await waitFor(async () =>
      expect((await client.devices.retrieve(dev.key)).name).toBe(renamed),
    );
  });

  it("deletes a device after confirmation", async () => {
    const dev = await createTestDevice(client);
    const host = hostDevices(dev);
    await renderDeviceTree(host);
    await openTreeRowContextMenu(dev.name);
    fireEvent.click(await screen.findByText("Delete"));
    await screen.findByText(`Are you sure you want to delete ${dev.name}?`);
    fireEvent.click(findButton("Delete"));
    await waitFor(async () => {
      await expect(client.devices.retrieve(dev.key)).rejects.toSatisfy((e) =>
        NotFoundError.matches(e),
      );
    });
  });

  it("reports a delete that a task config refuses", async () => {
    const dev = await createTestDevice(client);
    const rack = await client.racks.retrieve({ key: dev.rack });
    const task = await rack.createTask({
      name: uniqueName("read"),
      type: "modbus_read",
      config: { device: dev.key, channels: [] },
    });
    let statuses: Status.NotificationSpec[] = [];
    await renderDeviceTree(
      hostDevices(dev),
      createElement(CaptureStatuses, { onStatuses: (s) => (statuses = s) }),
    );
    await openTreeRowContextMenu(dev.name);
    fireEvent.click(await screen.findByText("Delete"));
    await screen.findByText(`Are you sure you want to delete ${dev.name}?`);
    fireEvent.click(findButton("Delete"));
    await waitFor(() =>
      expect(statuses).toContainEqual(
        expect.objectContaining({
          message: "Failed to delete device",
          description:
            `cannot delete device ${dev.name} because task ${task.name} uses it: ` +
            "validation error",
        }),
      ),
    );
    expect((await client.devices.retrieve({ key: dev.key })).key).toBe(dev.key);
  });

  it("should configure a found device with an action to create a read task", async () => {
    const dev = await createTestDevice(client, {
      make: NI.Device.MAKE,
      configured: false,
    });
    const host = hostDevices(dev);
    await renderDeviceTree(host);
    await openTreeRowContextMenu(dev.name);
    fireEvent.click(await screen.findByText("Configure"));
    expect(await screen.findByText("Save and create read task")).toBeTruthy();
  });

  it("should open the EtherCAT form from Configure on a configured slave", async () => {
    const rack = await client.racks.create({ name: uniqueName("ecat_rack") });
    const identifier = createIdentifier();
    const slave = await createSlaveDevice(client, rack.key, { identifier });
    const host = hostDevices(slave);
    await renderDeviceTree(host);
    await openTreeRowContextMenu(slave.name);
    fireEvent.click(await screen.findByText("Configure"));
    expect(await screen.findByDisplayValue(identifier)).toBeTruthy();
    expect(screen.queryByText("Save and create read task")).toBeNull();
  });

  it("should open the connect form from Edit connection", async () => {
    const dev = await createModbusDevice(client);
    const host = hostDevices(dev);
    await renderDeviceTree(host);
    await openTreeRowContextMenu(dev.name);
    fireEvent.click(await screen.findByText("Edit connection"));
    expect(await screen.findByText("Modbus server")).toBeTruthy();
    await waitFor(() => expect(screen.getByDisplayValue(dev.name)).toBeTruthy());
  });
});

describe("device location", () => {
  it("should read the location in the format of the device's integration", async () => {
    const dev = await createTestDevice(client, {
      make: OPCUA.Device.MAKE,
      location: "opc.tcp://10.0.0.5:4840",
    });
    await renderDeviceTree(hostDevices(dev));
    expect(await screen.findByText("10.0.0.5")).toBeTruthy();
    expect(screen.queryByText(/opc\.tcp/)).toBeNull();
  });
});

describe("permission to write the device", () => {
  it("should offer Reload Console", async () => {
    const dev = await createTestDevice(client);
    const Item = Device.TREE_ITEMS.device;
    assertDefined(Item.ContextMenu);
    await renderTreeContextMenu(Item.ContextMenu, {
      client,
      resources: [createResource(deviceClient.resourceID(dev.key), dev.name)],
    });
    expect(await screen.findByText("Reload Console")).toBeTruthy();
  });

  it("should withhold rename, grouping, and delete from a viewer", async () => {
    const dev = await createTestDevice(client);
    const Item = Device.TREE_ITEMS.device;
    assertDefined(Item.ContextMenu);
    await renderTreeContextMenu(Item.ContextMenu, {
      client: await roles.get("Viewer"),
      resources: [createResource(deviceClient.resourceID(dev.key), dev.name)],
    });
    expect(await screen.findByText("Copy properties")).toBeTruthy();
    expect(screen.queryByText("Rename")).toBeNull();
    expect(screen.queryByText("Group selection")).toBeNull();
    expect(screen.queryByText("Delete")).toBeNull();
  });
});
