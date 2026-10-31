// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { device, rack, status } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Status } from "@synnaxlabs/lyra/status";
import { TimeStamp } from "@synnaxlabs/x";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it } from "vitest";
import { type z } from "zod";

import { Device } from "@/feature/device";
import { NI } from "@/feature/ni";
import { OPCUA } from "@/feature/opcua";
import { createTestDevice } from "@/platform/device/testutil";
import { Modals } from "@/platform/modals";
import { Notifications } from "@/platform/notifications";
import {
  checkInUse,
  renderForward,
  settleWrites,
} from "@/platform/notifications/testutil";
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

  describe("NOTIFICATION_RULE", () => {
    it("should hold a configured device in use", async () => {
      const dev = await createTestDevice(client, { configured: true });
      expect(await checkInUse(client, Device.NOTIFICATION_RULE, dev.key)).toBe(true);
    });

    it("should not hold a device that is not configured in use", async () => {
      const dev = await createTestDevice(client, { configured: false });
      expect(await checkInUse(client, Device.NOTIFICATION_RULE, dev.key)).toBe(false);
    });

    it("should not hold a device that no longer exists in use", async () => {
      expect(
        await checkInUse(client, Device.NOTIFICATION_RULE, uniqueName("device")),
      ).toBe(false);
    });

    describe("onUseChange", () => {
      it("should list a problem once its device is configured", async () => {
        const { result } = await renderForward(client, {
          device: Device.NOTIFICATION_RULE,
        });
        const dev = await createTestDevice(client, { configured: false });
        const { key, message } = await client.statuses.set({
          key: device.statusKey(dev.key),
          name: dev.name,
          variant: "error",
          message: uniqueName("message"),
          time: TimeStamp.now(),
        });
        await settleWrites(client);
        await client.devices.create({ ...dev, configured: true });
        await waitFor(() =>
          expect(result.current.map(({ status: s }) => s.key)).toContain(key),
        );
        expect(screen.queryByText(message)).toBeNull();
      });
    });

    describe("coveredBy", () => {
      const DISCONNECTED = status.create({
        variant: "warning",
        message: "Device disconnected",
      });
      const coveredBy = (stat: status.Status): status.Key | null => {
        const rule = Device.NOTIFICATION_RULE;
        if (rule.coveredBy == null)
          throw new Error("the device rule must name a cover");
        return rule.coveredBy(stat);
      };

      it("should name the status of the device's rack", () => {
        const stat: device.Status = {
          ...DISCONNECTED,
          details: { rack: 7, device: "dev1" },
        };
        expect(coveredBy(stat)).toBe(rack.statusKey(7));
      });

      it("should name nothing for a status with no rack", () => {
        expect(coveredBy(DISCONNECTED)).toBeNull();
      });
    });
  });
});
