// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device, task } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Icon } from "@synnaxlabs/lyra/icon";
import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Device } from "@/platform/device";
import { createRunningRack, createTestDevice } from "@/platform/device/testutil";
import { findButton, renderModalOpener } from "@/platform/modals/testutil";
import { createTaskStatus } from "@/platform/task/testutil";
import { uniqueName } from "@/testutil";

const client = createTestClient();

const createHardware = (): Device.Hardware => ({
  key: uniqueName("integration"),
  name: "Test",
  title: "Test hardware",
  description: "",
  icon: <Icon.Device />,
  make: uniqueName("make"),
  scanType: uniqueName("scan"),
  checks: [],
  zeroProperties: {},
});

describe("Device.useFindModal", () => {
  it("should resolve with the device the user configures", async () => {
    const hardware = createHardware();
    const rack = await createRunningRack(client, hardware.key);
    const dev = await createTestDevice(client, {
      make: hardware.make,
      configured: false,
      rack: rack.key,
    });
    const { result } = await renderModalOpener(Device.useFindModal, [{ hardware }], {
      client,
    });
    await screen.findByText(dev.name);
    fireEvent.click(findButton("Configure"));
    fireEvent.change(await screen.findByPlaceholderText("dev1"), {
      target: { value: "dev" },
    });
    fireEvent.click(findButton("Save"));
    expect(await result()).toBe(dev.key);
  });

  it("should show a warning from the scanner", async () => {
    // The Core creates only the task types it knows.
    const hardware = { ...createHardware(), scanType: "ethercat_scan" };
    const rack = await createRunningRack(client, hardware.key);
    const scanner = await rack.createTask({
      name: uniqueName("scanner"),
      type: hardware.scanType,
      config: {},
      internal: true,
    });
    await client.statuses.set(
      createTaskStatus({
        key: task.statusKey(scanner.key),
        variant: "warning",
        message: "Failed to scan for devices",
        description: "Cannot open en1 for raw Ethernet access.",
        details: { task: scanner.key, running: true, rack: rack.key },
      }),
    );
    await renderModalOpener(Device.useFindModal, [{ hardware }], { client });
    expect(await screen.findByText("Failed to scan for devices")).toBeTruthy();
    expect(screen.getByText("Cannot open en1 for raw Ethernet access.")).toBeTruthy();
  });

  it("should list only the devices the filter accepts", async () => {
    const hardware = createHardware();
    const rack = await createRunningRack(client, hardware.key);
    const create = async () =>
      await createTestDevice(client, {
        make: hardware.make,
        configured: false,
        rack: rack.key,
      });
    const accepted = await create();
    const rejected = await create();
    const filter = (d: device.Device) => d.key === accepted.key;
    await renderModalOpener(Device.useFindModal, [{ hardware, filter }], { client });
    await screen.findByText(accepted.name);
    expect(screen.queryByText(rejected.name)).toBeNull();
  });

  it("should say the integration is off when a running Driver does not have it", async () => {
    const hardware = createHardware();
    await createRunningRack(client, uniqueName("integration"));
    await renderModalOpener(Device.useFindModal, [{ hardware }], { client });
    await screen.findByText("Your Driver is running, but Test is off");
  });
});
