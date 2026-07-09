// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device, type status } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Status } from "@synnaxlabs/pluto";
import { TimeStamp } from "@synnaxlabs/x";
import { fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it } from "vitest";

import { Device } from "@/feature/device";
import { NI } from "@/feature/ni";
import { OPC } from "@/feature/opc";
import { createTestDevice } from "@/platform/device/testutil";
import { Modals } from "@/platform/modals";
import { Notifications } from "@/platform/notifications";
import { createConsoleWrapper, renderWithConsole, uniqueName } from "@/testutil";

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

const silence = (): void => {};

describe("device/notifications", () => {
  it("does not match statuses for devices that are already configured", async () => {
    const dev = await createTestDevice(client, { configured: true });
    expect(Device.Notification.match(createNotification(dev))).toBe(false);
  });

  it("renders the device message without a configure action for OPC", async () => {
    const dev = await createTestDevice(client, {
      configured: false,
      make: OPC.Device.MAKE,
    });
    const { wrapper } = await createConsoleWrapper({ client });
    render(<Device.Notification status={createNotification(dev)} silence={silence} />, {
      wrapper,
    });
    expect(await screen.findByText(`New ${dev.model} connected`)).toBeTruthy();
    expect(screen.queryByText("Configure")).toBeNull();
  });

  it("offers a configure action that opens the vendor configure modal", async () => {
    const dev = await createTestDevice(client, {
      configured: false,
      make: NI.Device.MAKE,
    });
    const { wrapper } = await createConsoleWrapper({ client });
    render(
      <>
        <Device.Notification status={createNotification(dev)} silence={silence} />
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

  describe("NOTIFICATIONS", () => {
    interface HarnessProps {
      crude: status.Crude;
    }

    const RoutineHarness = ({ crude }: HarnessProps) => {
      const add = Status.useAdder();
      return (
        <>
          <button onClick={() => add(crude)}>add</button>
          <Notifications.Notifications notifications={Device.NOTIFICATIONS} />
        </>
      );
    };
    RoutineHarness.displayName = "RoutineHarness";

    const addRoutineStatus = async (crude: status.Crude): Promise<void> => {
      await renderWithConsole(<RoutineHarness crude={crude} />);
      fireEvent.click(screen.getByText("add"));
    };

    beforeEach(() => {
      const root = document.createElement("div");
      root.id = "root";
      document.body.appendChild(root);
    });
    afterEach(() => document.getElementById("root")?.remove());

    it("suppresses a routine loading status for a device", async () => {
      await addRoutineStatus({
        key: "device:1",
        variant: "loading",
        message: "Device connecting",
      });
      expect(screen.queryByText("Device connecting")).toBeNull();
    });

    it("suppresses a routine success status for a device", async () => {
      await addRoutineStatus({
        key: "device:1",
        variant: "success",
        message: "Device connected",
      });
      expect(screen.queryByText("Device connected")).toBeNull();
    });

    it("does not suppress a device error status", async () => {
      await addRoutineStatus({
        key: "device:1",
        variant: "error",
        message: "Device disconnected",
      });
      expect(screen.getByText("Device disconnected")).toBeTruthy();
    });

    it("does not suppress a routine status for a non-device key", async () => {
      await addRoutineStatus({ key: "rack:1", variant: "success", message: "Rack ok" });
      expect(screen.getByText("Rack ok")).toBeTruthy();
    });
  });
});
