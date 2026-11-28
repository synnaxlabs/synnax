// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { labjack } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import type { Status } from "@synnaxlabs/lyra/status";
import { uuid } from "@synnaxlabs/x";
import { uniqueName } from "@synnaxlabs/x/testutil";
import { render, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { LabJack } from "@/feature/labjack";
import { CaptureStatuses, createConsoleWrapper } from "@/testutil";

const client = createTestClient();

const Listener = (): null => {
  LabJack.Device.useListenForChanges();
  return null;
};

const renderListener = async (): Promise<{
  statuses: () => Status.NotificationSpec[];
}> => {
  const { wrapper } = await createConsoleWrapper({ client });
  let latest: Status.NotificationSpec[] = [];
  render(
    <>
      <Listener />
      <CaptureStatuses onStatuses={(s) => (latest = s)} />
    </>,
    { wrapper },
  );
  return { statuses: () => latest };
};

describe("LabJack.Device.useListenForChanges", async () => {
  const rack = await client.racks.create({ name: uniqueName("labjack_rack") });
  const create = async (identifier: string): Promise<labjack.device.Device> =>
    await client.labjack.devices.create({
      rack: rack.key,
      name: uniqueName("labjack_dev"),
      serial: Math.floor(Math.random() * 400_000) * 10_000 + 4,
      connection: "LJM_ctUSB",
      model: "t7",
      ports: { model: "t7" },
      identifier,
    });
  const notified = (
    statuses: Status.NotificationSpec[],
    dev: labjack.device.Device,
  ): Status.NotificationSpec | undefined =>
    statuses.find((s) => s.message === "LabJack T7 connected" && s.name === dev.name);

  it("should notify once when an unconfigured device connects", async () => {
    const { statuses } = await renderListener();
    const dev = await create("");
    await waitFor(() => expect(notified(statuses(), dev)).toBeDefined());
    await client.labjack.devices.dispatch(
      dev.key,
      labjack.device.set({ path: ["serial"], value: dev.serial + 1 }),
    );
    const other = await create("");
    await waitFor(() => expect(notified(statuses(), other)).toBeDefined());
    expect(notified(statuses(), dev)?.count).toBe(1);
  });

  it("should not notify for a device that has an identifier", async () => {
    const { statuses } = await renderListener();
    const configured = await create("dev1");
    const dev = await create("");
    await waitFor(() => expect(notified(statuses(), dev)).toBeDefined());
    expect(notified(statuses(), configured)).toBeUndefined();
  });

  describe("NOTIFICATION_RULE.inUse", () => {
    it("should count a device with an identifier as in use", async () => {
      const dev = await create("dev1");
      expect(
        await LabJack.Device.NOTIFICATION_RULE.inUse?.({ client, key: dev.key }),
      ).toBe(true);
    });

    it("should not count a device without an identifier as in use", async () => {
      const dev = await create("");
      expect(
        await LabJack.Device.NOTIFICATION_RULE.inUse?.({ client, key: dev.key }),
      ).toBe(false);
    });

    it("should not count a missing device as in use", async () => {
      expect(
        await LabJack.Device.NOTIFICATION_RULE.inUse?.({ client, key: uuid.create() }),
      ).toBe(false);
    });
  });
});
