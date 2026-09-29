// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arinc429, can, type device, serial, tcp, udp } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { ARINC429 } from "@/feature/arinc429";
import { createBusDevice, enterField, findOpenDialog } from "@/feature/bus/testutil";
import { CAN } from "@/feature/can";
import { Serial } from "@/feature/serial";
import { TCP } from "@/feature/tcp";
import { UDP } from "@/feature/udp";
import { findButton, renderModalOpener } from "@/platform/modals/testutil";
import { selectFromDropdown } from "@/platform/task/testutil";
import { findDialogTriggerByText } from "@/testutil";

const client = createTestClient();

const openConnect = async (
  useConnectModal: typeof CAN.Device.useConnectModal,
  deviceKey: device.Key,
) => {
  await renderModalOpener(useConnectModal, [{ deviceKey }], { client });
  await screen.findByRole("dialog");
};

const connectAndRetrieve = async (key: device.Key): Promise<device.Device> => {
  fireEvent.click(findButton("Connect"));
  return await waitFor(async () => {
    const dev = await client.devices.retrieve({ key });
    if (!dev.configured) throw new Error(`device ${key} is not configured yet`);
    return dev;
  });
};

const listBackends = async (): Promise<string[]> => {
  fireEvent.click(await screen.findByRole("button", { name: "Backend" }));
  const options = await within(await findOpenDialog()).findAllByRole("option");
  return options.map((o) => o.textContent);
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
    const dev = await createBusDevice(
      client,
      CAN.Device.MAKE,
      can.propertiesZ.parse({}),
      { configured: false },
    );
    await openConnect(CAN.Device.useConnectModal, dev.key);
    await enterField("Channel", "vcan0");
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
    await enterField("Host", "10.0.0.2");
    await enterField("Port", "5025");
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
    await enterField("Local port", "5000");
    const saved = await connectAndRetrieve(dev.key);
    expect(saved.model).toBe("UDP socket");
    expect(saved.location).toBe(":5000");
  });

  it("should locate an ARINC 429 device at its card and channel", async () => {
    const dev = await createBusDevice(
      client,
      ARINC429.Device.MAKE,
      arinc429.propertiesZ.parse({ backend: "simulated" }),
      { configured: false },
    );
    await openConnect(ARINC429.Device.useConnectModal, dev.key);
    await enterField("Card", "1");
    await enterField("Channel", "2");
    await selectFromDropdown("High (100 kbit/s)", "Low (12.5 kbit/s)");
    const saved = await connectAndRetrieve(dev.key);
    expect(saved.make).toBe("ARINC 429");
    expect(saved.model).toBe("simulated");
    expect(saved.location).toBe("Card 1, channel 2");
    expect(arinc429.propertiesZ.parse(saved.properties)).toEqual({
      backend: "simulated",
      card: 1,
      channel: 2,
      speed: "low",
    });
  });

  it("should require a backend before connecting a new ARINC 429 device", async () => {
    await renderModalOpener(ARINC429.Device.useConnectModal, [{}], { client });
    await screen.findByRole("button", { name: "Backend" });
    fireEvent.click(findButton("Connect"));
    await screen.findByText("Select a backend");
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

  describe("backends", () => {
    it("should offer only the ARINC 429 backends the Driver drives", async () => {
      const dev = await createBusDevice(
        client,
        ARINC429.Device.MAKE,
        arinc429.propertiesZ.parse({ backend: "simulated" }),
        { configured: false },
      );
      await openConnect(ARINC429.Device.useConnectModal, dev.key);
      expect(await listBackends()).toEqual(["Simulated", "DDC"]);
    });

    it("should keep a stored ARINC 429 backend that is not offered", async () => {
      const dev = await createBusDevice(
        client,
        ARINC429.Device.MAKE,
        arinc429.propertiesZ.parse({ backend: "ballard" }),
        { configured: false },
      );
      await openConnect(ARINC429.Device.useConnectModal, dev.key);
      await findDialogTriggerByText("Astronics Ballard");
      expect(await listBackends()).toEqual(["Simulated", "DDC", "Astronics Ballard"]);
      const saved = await connectAndRetrieve(dev.key);
      expect(saved.model).toBe("ballard");
      expect(arinc429.propertiesZ.parse(saved.properties).backend).toBe("ballard");
    });
  });
});
