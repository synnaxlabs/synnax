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

import { Modbus } from "@/feature/modbus";
import { createModbusDevice } from "@/feature/modbus/testutil";
import { createTestSink } from "@/platform/analytics/testutil";
import { createRunningRack } from "@/platform/device/testutil";
import { pressSaveTrigger, renderModalOpener } from "@/platform/modals/testutil";
import {
  answerNextCommand,
  commitFieldInput,
  selectFromDropdown,
} from "@/platform/task/testutil";
import { uniqueName } from "@/testutil";

const client = createTestClient();

describe("Modbus.Device.useConnectModal", () => {
  it("should populate the form from an existing device", async () => {
    const dev = await createModbusDevice(client, {
      properties: {
        connection: { host: "modbus-existing.local", port: 1502 },
      },
    });
    await renderModalOpener(Modbus.Device.useConnectModal, [{ deviceKey: dev.key }], {
      client,
    });
    await screen.findByDisplayValue(dev.name);
    expect(screen.getByDisplayValue("modbus-existing.local")).toBeTruthy();
    expect(screen.getByDisplayValue("1502")).toBeTruthy();
  });

  it("should show the unit and byte order of an existing device", async () => {
    const dev = await createModbusDevice(client, {
      properties: { connection: { unitId: 7, swapBytes: true, swapWords: false } },
    });
    await renderModalOpener(Modbus.Device.useConnectModal, [{ deviceKey: dev.key }], {
      client,
    });
    await screen.findByDisplayValue(dev.name);
    expect(screen.getByDisplayValue("7")).toBeTruthy();
    expect(screen.getByText("Little-endian (DCBA)")).toBeTruthy();
  });

  it("should save the unit and byte order entered for a new server", async () => {
    const rack = await createRunningRack(client, "modbus");
    const scan = await rack.createTask({
      name: uniqueName("modbus_scan"),
      type: Modbus.Task.SCAN_TYPE,
      config: {},
    });
    const { result } = await renderModalOpener(
      Modbus.Device.useConnectModal,
      [{ rack: rack.key }],
      { client },
    );
    fireEvent.change(screen.getByPlaceholderText("localhost"), {
      target: { value: "gateway.local" },
    });
    commitFieldInput(screen.getByDisplayValue<HTMLInputElement>("255"), "12");
    await selectFromDropdown("Word swap (CDAB)", "Big-endian (ABCD)");
    const { answered } = await answerNextCommand(client, scan);
    pressSaveTrigger();
    await answered;
    const key = await result();
    if (key == null) throw new Error("the modal closed without a device");
    const dev = await client.devices.retrieve({ key, schemas: Modbus.Device.SCHEMAS });
    expect(dev.properties.connection.unitId).toBe(12);
    expect(dev.properties.connection.swapBytes).toBe(false);
    expect(dev.properties.connection.swapWords).toBe(true);
    expect(dev.properties.connection.timeoutMs).toBe(1000);
    expect(dev.properties.requests.writeCodes).toBe("multiple");
  });

  it("should start a new server at the standard Modbus TCP port", async () => {
    await renderModalOpener(Modbus.Device.useConnectModal, [{}], { client });
    await screen.findByText("Modbus TCP server");
    expect(screen.getByDisplayValue("502")).toBeTruthy();
  });

  it("should resolve with the device it connected", async () => {
    const rack = await createRunningRack(client, "modbus");
    const scan = await rack.createTask({
      name: uniqueName("modbus_scan"),
      type: Modbus.Task.SCAN_TYPE,
      config: {},
    });
    const { result } = await renderModalOpener(
      Modbus.Device.useConnectModal,
      [{ rack: rack.key }],
      { client },
    );
    fireEvent.change(screen.getByPlaceholderText("localhost"), {
      target: { value: "localhost" },
    });
    const { answered } = await answerNextCommand(client, scan);
    pressSaveTrigger();
    await answered;
    const key = await result();
    if (key == null) throw new Error("the modal closed without a device");
    expect((await client.devices.retrieve({ key })).rack).toBe(rack.key);
  });

  it("should report a device it connected", async () => {
    const rack = await createRunningRack(client, "modbus");
    const scan = await rack.createTask({
      name: uniqueName("modbus_scan"),
      type: Modbus.Task.SCAN_TYPE,
      config: {},
    });
    const analytics = createTestSink();
    await renderModalOpener(Modbus.Device.useConnectModal, [{ rack: rack.key }], {
      client,
      analytics,
    });
    fireEvent.change(screen.getByPlaceholderText("localhost"), {
      target: { value: "localhost" },
    });
    const { answered } = await answerNextCommand(client, scan);
    pressSaveTrigger();
    await answered;
    await waitFor(() =>
      expect(analytics.capture).toHaveBeenCalledWith("device_connected", {
        integration: "modbus",
      }),
    );
  });

  it("should show a failure from a Driver that sends no description", async () => {
    const rack = await createRunningRack(client, "modbus");
    const scan = await rack.createTask({
      name: uniqueName("modbus_scan"),
      type: Modbus.Task.SCAN_TYPE,
      config: {},
    });
    await renderModalOpener(Modbus.Device.useConnectModal, [{ rack: rack.key }], {
      client,
    });
    fireEvent.change(screen.getByPlaceholderText("localhost"), {
      target: { value: "localhost" },
    });
    const { answered } = await answerNextCommand(client, scan, {
      variant: "error",
      message: "Connection refused",
    });
    pressSaveTrigger();
    await answered;
    expect(await screen.findByText("Connection refused")).toBeTruthy();
  });

  // A running rack keeps Connect enabled while none is chosen, so the save fails
  // validation. That error proves the keys reached the Connect button's save path.
  it("should submit on the shortcut its footer advertises", async () => {
    await createRunningRack(client, "modbus");
    await renderModalOpener(Modbus.Device.useConnectModal, [{}], { client });
    await screen.findByRole("dialog");
    pressSaveTrigger();
    expect(await screen.findByText(/rack is required/i)).toBeTruthy();
  });
});
