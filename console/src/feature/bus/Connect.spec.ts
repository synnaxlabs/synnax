// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { can, type device, serial, tcp, udp } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { type record } from "@synnaxlabs/x";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { createBusDevice, enterField } from "@/feature/bus/testutil";
import { CAN } from "@/feature/can";
import { Serial } from "@/feature/serial";
import { TCP } from "@/feature/tcp";
import { UDP } from "@/feature/udp";
import { createTestSink } from "@/platform/analytics/testutil";
import { createRunningRack } from "@/platform/device/testutil";
import { findButton, renderModalOpener } from "@/platform/modals/testutil";

const client = createTestClient();

const openConnect = async (
  useConnectModal: typeof CAN.Device.useConnectModal,
  deviceKey: device.Key,
) => {
  const handle = await renderModalOpener(useConnectModal, [{ deviceKey }], { client });
  await screen.findByRole("dialog");
  return handle;
};

/** Creates a device whose rack runs the integration, so the form can connect it. */
const createConnectable = async (
  integration: string,
  make: string,
  properties: record.Unknown,
  model?: string,
): Promise<device.Device> => {
  const rack = await createRunningRack(client, integration);
  return await createBusDevice(client, make, properties, {
    configured: false,
    model,
    rack: rack.key,
  });
};

const connectAndRetrieve = async (key: device.Key): Promise<device.Device> => {
  fireEvent.click(findButton("Connect"));
  return await waitFor(async () => {
    const dev = await client.devices.retrieve({ key });
    if (!dev.configured) throw new Error(`device ${key} is not configured yet`);
    return dev;
  });
};

describe("bus device connect", () => {
  it("should finish a scanned serial device with its port as its location", async () => {
    const dev = await createConnectable(
      "serial",
      Serial.Device.MAKE,
      serial.propertiesZ.parse({ port: "/dev/ttyUSB3" }),
      "Serial port",
    );
    await openConnect(Serial.Device.useConnectModal, dev.key);
    await screen.findByDisplayValue("/dev/ttyUSB3");
    await enterField("Baud rate", "115200");
    const saved = await connectAndRetrieve(dev.key);
    expect(saved.make).toBe("Serial");
    expect(saved.model).toBe("Serial port");
    expect(saved.location).toBe("/dev/ttyUSB3");
    expect(serial.propertiesZ.parse(saved.properties)).toMatchObject({
      port: "/dev/ttyUSB3",
      baudRate: 115200,
    });
  });

  it("should name a CAN device's model after its backend", async () => {
    const dev = await createConnectable(
      "can",
      CAN.Device.MAKE,
      can.propertiesZ.parse({}),
    );
    await openConnect(CAN.Device.useConnectModal, dev.key);
    await enterField("Channel", "vcan0");
    const saved = await connectAndRetrieve(dev.key);
    expect(saved.model).toBe("socketcan");
    expect(saved.location).toBe("vcan0");
    expect(can.propertiesZ.parse(saved.properties).channel).toBe("vcan0");
  });

  it("should locate a TCP device at its host and port and resolve with it", async () => {
    const dev = await createConnectable(
      "tcp",
      TCP.Device.MAKE,
      tcp.propertiesZ.parse({}),
    );
    const { result } = await openConnect(TCP.Device.useConnectModal, dev.key);
    await enterField("Host", "10.0.0.2");
    await enterField("Port", "5025");
    const saved = await connectAndRetrieve(dev.key);
    expect(saved.model).toBe("TCP server");
    expect(saved.location).toBe("10.0.0.2:5025");
    expect(await result()).toBe(dev.key);
  });

  it("should locate a UDP device at its local port", async () => {
    const dev = await createConnectable(
      "udp",
      UDP.Device.MAKE,
      udp.propertiesZ.parse({}),
    );
    await openConnect(UDP.Device.useConnectModal, dev.key);
    await enterField("Local port", "5000");
    const saved = await connectAndRetrieve(dev.key);
    expect(saved.model).toBe("UDP socket");
    expect(saved.location).toBe(":5000");
  });

  it("should mark a UDP device's connection as not monitored", async () => {
    const dev = await createConnectable(
      "udp",
      UDP.Device.MAKE,
      udp.propertiesZ.parse({}),
    );
    await openConnect(UDP.Device.useConnectModal, dev.key);
    await enterField("Local port", "5002");
    await connectAndRetrieve(dev.key);
    await waitFor(async () => {
      const { status } = await client.devices.retrieve({
        key: dev.key,
        includeStatus: true,
      });
      expect(status).toMatchObject({
        variant: "disabled",
        message: "Connection not monitored",
        details: { connected: false },
      });
    });
  });

  it("should leave the status of a scanned TCP device to its scan task", async () => {
    const dev = await createConnectable(
      "tcp",
      TCP.Device.MAKE,
      tcp.propertiesZ.parse({}),
    );
    await openConnect(TCP.Device.useConnectModal, dev.key);
    await enterField("Host", "10.0.0.3");
    await enterField("Port", "5025");
    await connectAndRetrieve(dev.key);
    const { status } = await client.devices.retrieve({
      key: dev.key,
      includeStatus: true,
    });
    expect(status?.message).toBe("Device status unknown");
  });

  it("should report a device it connected", async () => {
    const rack = await createRunningRack(client, "udp");
    const analytics = createTestSink();
    await renderModalOpener(UDP.Device.useConnectModal, [{ rack: rack.key }], {
      client,
      analytics,
    });
    await enterField("Local port", "5001");
    fireEvent.click(findButton("Connect"));
    await waitFor(() =>
      expect(analytics.capture).toHaveBeenCalledWith("device_connected", {
        integration: "udp",
      }),
    );
  });

  it("should show an empty required property on its field", async () => {
    const dev = await createConnectable(
      "tcp",
      TCP.Device.MAKE,
      tcp.propertiesZ.parse({}),
    );
    await openConnect(TCP.Device.useConnectModal, dev.key);
    await screen.findByDisplayValue(dev.name);
    fireEvent.click(findButton("Connect"));
    await screen.findByText("Host is required");
    expect((await client.devices.retrieve({ key: dev.key })).configured).toBe(false);
  });
});
