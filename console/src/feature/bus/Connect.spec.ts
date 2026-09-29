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
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { createBusDevice } from "@/feature/bus/testutil";
import { CAN } from "@/feature/can";
import { Serial } from "@/feature/serial";
import { TCP } from "@/feature/tcp";
import { UDP } from "@/feature/udp";
import { findButton, renderModalOpener } from "@/platform/modals/testutil";

const client = createTestClient();

const openConnect = async (
  useConnectModal: typeof CAN.Device.useConnectModal,
  deviceKey: device.Key,
) => {
  await renderModalOpener(useConnectModal, [{ deviceKey }], { client });
  await screen.findByRole("dialog");
};

const enter = async (label: string, value: string) => {
  const input = await screen.findByLabelText(label);
  fireEvent.change(input, { target: { value } });
  fireEvent.blur(input);
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
    const dev = await createBusDevice(
      client,
      Serial.Device.MAKE,
      serial.propertiesZ.parse({ port: "/dev/ttyUSB3" }),
      { configured: false, model: "Serial port" },
    );
    await openConnect(Serial.Device.useConnectModal, dev.key);
    await screen.findByDisplayValue("/dev/ttyUSB3");
    await enter("Baud rate", "115200");
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
    const dev = await createBusDevice(
      client,
      CAN.Device.MAKE,
      can.propertiesZ.parse({}),
      { configured: false },
    );
    await openConnect(CAN.Device.useConnectModal, dev.key);
    await enter("Channel", "vcan0");
    const saved = await connectAndRetrieve(dev.key);
    expect(saved.model).toBe("socketcan");
    expect(saved.location).toBe("vcan0");
    expect(can.propertiesZ.parse(saved.properties).channel).toBe("vcan0");
  });

  it("should locate a TCP device at its host and port", async () => {
    const dev = await createBusDevice(
      client,
      TCP.Device.MAKE,
      tcp.propertiesZ.parse({}),
      { configured: false },
    );
    await openConnect(TCP.Device.useConnectModal, dev.key);
    await enter("Host", "10.0.0.2");
    await enter("Port", "5025");
    const saved = await connectAndRetrieve(dev.key);
    expect(saved.model).toBe("TCP server");
    expect(saved.location).toBe("10.0.0.2:5025");
  });

  it("should locate a UDP device at its local port", async () => {
    const dev = await createBusDevice(
      client,
      UDP.Device.MAKE,
      udp.propertiesZ.parse({}),
      { configured: false },
    );
    await openConnect(UDP.Device.useConnectModal, dev.key);
    await enter("Local port", "5000");
    const saved = await connectAndRetrieve(dev.key);
    expect(saved.model).toBe("UDP socket");
    expect(saved.location).toBe(":5000");
  });

  it("should show an empty required property on its field", async () => {
    const dev = await createBusDevice(
      client,
      TCP.Device.MAKE,
      tcp.propertiesZ.parse({}),
      { configured: false },
    );
    await openConnect(TCP.Device.useConnectModal, dev.key);
    await screen.findByDisplayValue(dev.name);
    fireEvent.click(findButton("Connect"));
    await screen.findByText("Host is required");
    expect((await client.devices.retrieve({ key: dev.key })).configured).toBe(false);
  });
});
