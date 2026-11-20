// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { device, type task, TimeStamp } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Icon } from "@synnaxlabs/lyra/icon";
import { uniqueName } from "@synnaxlabs/x/testutil";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Device } from "@/platform/device";
import {
  createRunningRack,
  createTestDevice,
  createWarningScanner,
  startRackHeartbeat,
} from "@/platform/device/testutil";
import { findButton, renderModalOpener } from "@/platform/modals/testutil";
import { answerNextCommand, type CommandAnswer } from "@/platform/task/testutil";

const client = createTestClient();

const createHardware = (): Device.Hardware => ({
  key: uniqueName("integration"),
  name: "Test",
  title: "Test hardware",
  description: "",
  icon: <Icon.Device />,
  make: uniqueName("make"),
  scanType: uniqueName("scan"),
  location: { shape: "plain" },
  checks: [],
  Configure: Device.createConfigure({
    icon: <Icon.Device />,
    zeroProperties: {},
    suggestIdentifiers: () => [],
  }),
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
    await createWarningScanner(client, hardware);
    await renderModalOpener(Device.useFindModal, [{ hardware }], { client });
    expect(await screen.findByText("Failed to scan for devices")).toBeTruthy();
    expect(screen.getByText("Cannot open en1 for raw Ethernet access.")).toBeTruthy();
  });

  it("should show the scan action beside a scanner warning", async () => {
    const hardware: Device.Hardware = {
      ...createHardware(),
      scanType: "ethercat_scan",
      ScanAction: ({ scanner }) => <span>{`Fix ${scanner.name}`}</span>,
    };
    const scanner = await createWarningScanner(client, hardware);
    await renderModalOpener(Device.useFindModal, [{ hardware }], { client });
    expect(await screen.findByText(`Fix ${scanner.name}`)).toBeTruthy();
  });

  it("should show the name that the integration gives a model", async () => {
    const hardware = { ...createHardware(), modelNames: { raw_model: "Module" } };
    const rack = await createRunningRack(client, hardware.key);
    await createTestDevice(client, {
      make: hardware.make,
      model: "raw_model",
      configured: false,
      rack: rack.key,
    });
    await renderModalOpener(Device.useFindModal, [{ hardware }], { client });
    expect(await screen.findByText("Module")).toBeTruthy();
    expect(screen.queryByText("raw_model")).toBeNull();
  });

  it("should turn on a disabled scanner", async () => {
    const hardware = { ...createHardware(), scanType: "ethercat_scan" };
    const rack = await createRunningRack(client, hardware.key);
    const scanner = await rack.createTask({
      name: uniqueName("scanner"),
      type: hardware.scanType,
      config: { disabled: true },
      internal: true,
    });
    // A client that did not create the scanner caches it only through the modal.
    await renderModalOpener(Device.useFindModal, [{ hardware }], {
      client: createTestClient(),
    });
    await screen.findByText("Scanning is off on this Driver");
    fireEvent.click(findButton("Turn on"));
    await waitFor(async () => {
      const { config } = await createTestClient().tasks.retrieve(scanner.key);
      expect(config).toMatchObject({ disabled: false });
    });
  });

  describe("Scan now", () => {
    const renderScan = async (connected: number, disconnected: number) => {
      const hardware = { ...createHardware(), scanType: "ethercat_scan" };
      const rack = await createRunningRack(client, hardware.key);
      // The Core clears the connected mark of a device on a silent rack.
      await startRackHeartbeat(client, rack);
      const scanner = await rack.createTask({
        name: uniqueName("scanner"),
        type: hardware.scanType,
        config: {},
        internal: true,
      });
      const create = async (marked: boolean) => {
        const dev = await createTestDevice(client, {
          make: hardware.make,
          rack: rack.key,
        });
        if (marked)
          await client.statuses.set({
            owner: device.resourceID(dev.key),
            name: dev.name,
            variant: "success",
            message: "Device connected",
            time: TimeStamp.now(),
            details: { rack: rack.key, connected: true },
          });
        return dev;
      };
      const devices = [
        ...(await Promise.all(Array.from({ length: connected }, () => create(true)))),
        ...(await Promise.all(
          Array.from({ length: disconnected }, () => create(false)),
        )),
      ];
      await renderModalOpener(Device.useFindModal, [{ hardware }], { client });
      await screen.findByText(devices[0].name);
      await scan(scanner);
      return scanner;
    };

    // The button is disabled until the scanner loads and while a scan runs.
    const findScanNow = async (): Promise<HTMLElement> =>
      await waitFor(() => {
        const b = findButton("Scan now");
        if (b.getAttribute("aria-disabled") === "true")
          throw new Error("Scan now is disabled");
        return b;
      });

    const scan = async (scanner: task.Task, answer: CommandAnswer = {}) => {
      const { answered } = await answerNextCommand(client, scanner, {
        ...answer,
        type: "scan",
      });
      fireEvent.click(await findScanNow());
      await answered;
    };

    it("should say how many devices are connected after a scan", async () => {
      await renderScan(1, 1);
      expect(await screen.findByText("Scan complete")).toBeTruthy();
      expect(screen.getByText("1 Test device connected.")).toBeTruthy();
    });

    it("should say that no device is connected after a scan", async () => {
      await renderScan(0, 1);
      expect(await screen.findByText("Scan complete")).toBeTruthy();
      expect(screen.getByText("No Test devices connected.")).toBeTruthy();
    });

    it("should clear the result of a scan when a later scan fails", async () => {
      const scanner = await renderScan(1, 0);
      await screen.findByText("Scan complete");
      await scan(scanner, { variant: "error", message: "Failed to scan" });
      expect(await screen.findByText("Failed to scan")).toBeTruthy();
      await findScanNow();
      expect(screen.queryByText("Scan complete")).toBeNull();
    });
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

  it("should list slots in the order of the bus", async () => {
    const hardware: Device.Hardware = {
      ...createHardware(),
      location: { shape: "slot" },
    };
    const rack = await createRunningRack(client, hardware.key);
    const create = async (location: string, configured: boolean) =>
      await createTestDevice(client, {
        make: hardware.make,
        configured,
        rack: rack.key,
        location,
      });
    const tenth = await create("eth0.Slot 10", false);
    const first = await create("eth0.Slot 1", true);
    const second = await create("eth0.Slot 2", false);
    await renderModalOpener(Device.useFindModal, [{ hardware }], { client });
    await screen.findByText(tenth.name);
    const names = [first, second, tenth].map(({ name }) => name);
    const shown = screen
      .getAllByText((content) => names.includes(content))
      .map((el) => el.textContent);
    expect(shown).toEqual(names);
  });

  it("should show the location that the hardware gives a found device", async () => {
    const hardware: Device.Hardware = {
      ...createHardware(),
      formatLocation: (d) => `${d.location} given`,
    };
    const rack = await createRunningRack(client, hardware.key);
    const dev = await createTestDevice(client, {
      make: hardware.make,
      configured: false,
      rack: rack.key,
      location: "en5.Slot 3",
    });
    await renderModalOpener(Device.useFindModal, [{ hardware }], { client });
    await screen.findByText(dev.name);
    expect(screen.getByText("en5.Slot 3 given")).toBeTruthy();
  });

  it("should say the integration is off when a running Driver does not have it", async () => {
    const hardware = createHardware();
    await createRunningRack(client, uniqueName("integration"));
    await renderModalOpener(Device.useFindModal, [{ hardware }], { client });
    await screen.findByText("Your Driver is running, but Test is off");
  });
});
