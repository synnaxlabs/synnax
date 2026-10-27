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
import { Status } from "@synnaxlabs/lyra/status";
import { fireEvent, render, screen } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it } from "vitest";
import { type z } from "zod";

import { Device } from "@/feature/device";
import { NI } from "@/feature/ni";
import { OPCUA } from "@/feature/opcua";
import { createTestDevice } from "@/platform/device/testutil";
import { Modals } from "@/platform/modals";
import { Notifications } from "@/platform/notifications";
import { createConsoleWrapper, installPortalRoot, uniqueName } from "@/testutil";

const client = createTestClient();

type DeviceCrude = status.Crude<ReturnType<typeof device.deviceZ>>;

const createCrude = (dev: device.Device): DeviceCrude => ({
  key: uniqueName("status"),
  variant: "info",
  message: `New ${dev.model} connected`,
  details: dev,
});

const Harness = ({ onAdd }: { onAdd: (add: Status.Adder) => void }): ReactElement => {
  const add = Status.useAdder();
  return (
    <>
      <button onClick={() => onAdd(add)}>add</button>
      <Notifications.Feed notifications={Device.NOTIFICATIONS} />
      <Modals.Stack />
    </>
  );
};
Harness.displayName = "Harness";

const addStatus = async <Details extends z.ZodType = z.ZodNever>(
  crude: status.Crude<Details>,
): Promise<void> => {
  const { wrapper } = await createConsoleWrapper({ client });
  render(<Harness onAdd={(add) => add(crude)} />, { wrapper });
  fireEvent.click(screen.getByText("add"));
};

describe("device/notifications", () => {
  installPortalRoot();

  it("does not offer a configure action for a configured device", async () => {
    const dev = await createTestDevice(client, {
      configured: true,
      make: NI.Device.MAKE,
    });
    await addStatus(createCrude(dev));
    expect(await screen.findByText(`New ${dev.model} connected`)).toBeTruthy();
    expect(screen.queryByText("Configure")).toBeNull();
  });

  it("renders the device message without a configure action for OPC UA", async () => {
    const dev = await createTestDevice(client, {
      configured: false,
      make: OPCUA.Device.MAKE,
    });
    await addStatus(createCrude(dev));
    expect(await screen.findByText(`New ${dev.model} connected`)).toBeTruthy();
    expect(screen.queryByText("Configure")).toBeNull();
  });

  it("offers a configure action that opens the vendor configure modal", async () => {
    const dev = await createTestDevice(client, {
      configured: false,
      make: NI.Device.MAKE,
    });
    await addStatus(createCrude(dev));
    fireEvent.click(await screen.findByText("Configure"));
    expect(await screen.findByText(dev.name)).toBeTruthy();
  });

  describe("getKeyFromStatus", () => {
    it("returns the device key for unconfigured devices", async () => {
      const dev = await createTestDevice(client, { configured: false });
      expect(Device.getKeyFromStatus(createCrude(dev))).toBe(dev.key);
    });

    it("returns null for configured devices", async () => {
      const dev = await createTestDevice(client, { configured: true });
      expect(Device.getKeyFromStatus(createCrude(dev))).toBeNull();
    });
  });

  describe("NOTIFICATIONS", () => {
    it("suppresses a routine loading status for a device", async () => {
      await addStatus({
        key: "device:1",
        variant: "loading",
        message: "Device connecting",
      });
      expect(screen.queryByText("Device connecting")).toBeNull();
    });

    it("suppresses a routine success status for a device", async () => {
      await addStatus({
        key: "device:1",
        variant: "success",
        message: "Device connected",
      });
      expect(screen.queryByText("Device connected")).toBeNull();
    });

    it("does not suppress a device error status", async () => {
      await addStatus({
        key: "device:1",
        variant: "error",
        message: "Device disconnected",
      });
      expect(screen.getByText("Device disconnected")).toBeTruthy();
    });

    it("does not suppress a routine status for a non-device key", async () => {
      await addStatus({ key: "rack:1", variant: "success", message: "Rack ok" });
      expect(screen.getByText("Rack ok")).toBeTruthy();
    });
  });
});
