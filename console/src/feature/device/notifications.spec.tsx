// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, type device } from "@synnaxlabs/client";
import { type Status } from "@synnaxlabs/pluto";
import { TimeStamp } from "@synnaxlabs/x";
import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Device } from "@/feature/device";
import { NI } from "@/feature/ni";
import { OPC } from "@/feature/opc";
import { createTestDevice } from "@/platform/device/testutil";
import { Modals } from "@/platform/modals";
import { createConsoleWrapper, uniqueName } from "@/testutil";

const client = createTestClient();

type DeviceNotification = Status.NotificationSpec<ReturnType<typeof device.deviceZ>>;

const createNotification = (dev: device.Device): DeviceNotification => ({
  key: uniqueName("status"),
  name: "",
  variant: "info",
  message: `New ${dev.model} connected`,
  description: "",
  time: TimeStamp.now(),
  details: dev,
  count: 1,
});

const adapter = Device.NOTIFICATION_ADAPTERS[0];
const silence = (): void => {};

describe("device/notifications", () => {
  it("ignores statuses for devices that are already configured", async () => {
    const dev = await createTestDevice(client, { configured: true });
    expect(adapter(createNotification(dev), silence)).toBeNull();
  });

  it("sugars the status with the device message", async () => {
    const dev = await createTestDevice(client, {
      configured: false,
      make: OPC.Device.MAKE,
    });
    const sugared = adapter(createNotification(dev), silence);
    if (sugared == null) throw new Error("expected a sugared notification");
    expect(sugared.actions).toBeUndefined();
    if (sugared.content == null) throw new Error("expected sugared content");
    render(sugared.content);
    expect(screen.getByText(`New ${dev.model} connected`)).toBeTruthy();
  });

  it("offers a configure action that opens the vendor configure modal", async () => {
    const dev = await createTestDevice(client, {
      configured: false,
      make: NI.Device.MAKE,
    });
    const sugared = adapter(createNotification(dev), silence);
    if (sugared == null) throw new Error("expected a sugared notification");
    const { wrapper } = await createConsoleWrapper({ client });
    render(
      <>
        {sugared.actions}
        <Modals.Stack />
      </>,
      { wrapper },
    );
    fireEvent.click(await screen.findByText("Configure"));
    expect(await screen.findByText(dev.name)).toBeTruthy();
  });

  describe("getKeyFromStatus", () => {
    it("returns the device key for unconfigured devices", async () => {
      const dev = await createTestDevice(client, { configured: false });
      expect(Device.getKeyFromStatus(createNotification(dev))).toBe(dev.key);
    });

    it("returns null for configured devices", async () => {
      const dev = await createTestDevice(client, { configured: true });
      expect(Device.getKeyFromStatus(createNotification(dev))).toBeNull();
    });
  });
});
