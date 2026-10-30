// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { device, status } from "@synnaxlabs/client";
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

  it("should save the unit, byte order, and mask writes entered for a new server", async () => {
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
    fireEvent.click(screen.getByText("Advanced"));
    fireEvent.click(await screen.findByRole("checkbox"));
    const { answered } = await answerNextCommand(client, scan);
    pressSaveTrigger();
    const cmd = await answered;
    const key = await result();
    if (key == null) throw new Error("the modal closed without a device");
    const dev = await client.devices.retrieve({ key, schemas: Modbus.Device.SCHEMAS });
    expect(cmd.args).toEqual({
      connection: dev.properties.connection,
      requests: dev.properties.requests,
    });
    expect(dev.properties.connection.unitId).toBe(12);
    expect(dev.properties.connection.swapBytes).toBe(false);
    expect(dev.properties.connection.swapWords).toBe(true);
    expect(dev.properties.connection.timeoutMs).toBe(1000);
    expect(dev.properties.requests.writeCodes).toBe("multiple");
    expect(dev.properties.requests.maskWriteSupported).toBe(true);
  });

  it("should show the request counters the Driver reports for a server", async () => {
    const dev = await createModbusDevice(client);
    await client.statuses.set(
      status.create<typeof device.statusDetailsZ>({
        key: device.statusKey(dev.key),
        variant: "success",
        message: "Device connected",
        details: {
          rack: dev.rack,
          device: dev.key,
          data: {
            stats: {
              requests: 41,
              retries: 3,
              timeouts: 2,
              exceptions: 5,
              lastResponse: 4_500_000,
              maxResponse: 12_000_000,
            },
          },
        },
      }),
    );
    await renderModalOpener(Modbus.Device.useConnectModal, [{ deviceKey: dev.key }], {
      client,
    });
    fireEvent.click(await screen.findByText("Diagnostics"));
    expect(await screen.findByText("41")).toBeTruthy();
    expect(screen.getByText("3")).toBeTruthy();
    expect(screen.getByText("2")).toBeTruthy();
    expect(screen.getByText("5")).toBeTruthy();
    expect(screen.getByText("4.5 ms")).toBeTruthy();
    expect(screen.getByText("12.0 ms")).toBeTruthy();
  });

  it("should show that a server has no request counters yet", async () => {
    const dev = await createModbusDevice(client);
    await renderModalOpener(Modbus.Device.useConnectModal, [{ deviceKey: dev.key }], {
      client,
    });
    fireEvent.click(await screen.findByText("Diagnostics"));
    expect(await screen.findByText("No requests counted yet")).toBeTruthy();
  });

  it("should not show diagnostics for a new server", async () => {
    await renderModalOpener(Modbus.Device.useConnectModal, [{}], { client });
    await screen.findByText("Advanced");
    expect(screen.queryByText("Diagnostics")).toBeNull();
  });

  it("should start a new server at the standard Modbus TCP port", async () => {
    await renderModalOpener(Modbus.Device.useConnectModal, [{}], { client });
    await screen.findByText("Modbus server");
    expect(screen.getByDisplayValue("502")).toBeTruthy();
  });

  it("should show the serial port and line settings of a serial device", async () => {
    const dev = await createModbusDevice(client, {
      properties: {
        connection: { transport: "rtu", serialPort: "/dev/ttyUSB3", baudRate: 9600 },
      },
    });
    await renderModalOpener(Modbus.Device.useConnectModal, [{ deviceKey: dev.key }], {
      client,
    });
    await screen.findByDisplayValue(dev.name);
    expect(screen.getByDisplayValue("/dev/ttyUSB3")).toBeTruthy();
    expect(screen.getByText("9600")).toBeTruthy();
    expect(screen.getByText("Even")).toBeTruthy();
    expect(screen.getByDisplayValue("1")).toBeTruthy();
    expect(screen.queryByPlaceholderText("localhost")).toBeNull();
  });

  it("should reset the unit but keep the byte order on a new transport", async () => {
    const dev = await createModbusDevice(client, {
      properties: { connection: { unitId: 7, swapBytes: true, swapWords: false } },
    });
    await renderModalOpener(Modbus.Device.useConnectModal, [{ deviceKey: dev.key }], {
      client,
    });
    await screen.findByDisplayValue(dev.name);
    fireEvent.click(screen.getByText("Serial"));
    expect(await screen.findByDisplayValue("1")).toBeTruthy();
    expect(screen.getByText("Little-endian (DCBA)")).toBeTruthy();
    expect(screen.queryByPlaceholderText("localhost")).toBeNull();
  });

  it("should keep the unit ID of a serial device from 1 to 247", async () => {
    await renderModalOpener(Modbus.Device.useConnectModal, [{}], { client });
    fireEvent.click(await screen.findByText("Serial"));
    commitFieldInput(await screen.findByDisplayValue<HTMLInputElement>("1"), "300");
    expect(await screen.findByDisplayValue("247")).toBeTruthy();
  });

  it("should save a serial device with the line settings entered", async () => {
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
    fireEvent.click(await screen.findByText("Serial"));
    fireEvent.change(await screen.findByPlaceholderText("/dev/ttyUSB0"), {
      target: { value: "/dev/ttyUSB1" },
    });
    fireEvent.click(screen.getByText("19200"));
    fireEvent.click(await screen.findByText("230400"));
    fireEvent.click(screen.getByText("Even"));
    fireEvent.click(await screen.findByText("None"));
    const { answered } = await answerNextCommand(client, scan);
    pressSaveTrigger();
    await answered;
    const key = await result();
    if (key == null) throw new Error("the modal closed without a device");
    const dev = await client.devices.retrieve({ key, schemas: Modbus.Device.SCHEMAS });
    expect(dev.location).toBe("/dev/ttyUSB1");
    expect(dev.properties.connection).toEqual({
      transport: "rtu",
      serialPort: "/dev/ttyUSB1",
      baudRate: 230400,
      parity: "none",
      stopBits: 1,
      swapBytes: false,
      swapWords: false,
      unitId: 1,
      timeoutMs: 1000,
      retries: 1,
    });
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
