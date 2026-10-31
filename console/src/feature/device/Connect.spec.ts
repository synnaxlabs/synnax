// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type rack } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Device } from "@/feature/device";
import { LabJack } from "@/feature/labjack";
import { createRunningRack, createTestDevice } from "@/platform/device/testutil";
import { findButton, renderModalOpener } from "@/platform/modals/testutil";
import { createTaskStatus } from "@/platform/task/testutil";
import { uniqueName } from "@/testutil";

const client = createTestClient();

const open = async (params: Device.ConnectModalParams = {}) =>
  await renderModalOpener(Device.useConnectModal, [params], { client });

const createLabJackDevice = async (rackKey: rack.Key) =>
  await createTestDevice(client, {
    configured: false,
    name: uniqueName("t7"),
    make: LabJack.Device.MAKE,
    rack: rackKey,
  });

describe("Device.useConnectModal", () => {
  describe("picker", () => {
    it("should tell every entry that the scoped Driver is not running", async () => {
      const rack = await client.racks.create({
        name: uniqueName("rack"),
        integrations: ["opc"],
      });
      await open({ rack: rack.key });
      const reasons = await screen.findAllByText(`${rack.name} is not running`);
      expect(reasons).toHaveLength(6);
    });

    it("should name only the integrations the scoped Driver does not run", async () => {
      const rack = await createRunningRack(client, "opc");
      await open({ rack: rack.key });
      await screen.findByText(`${rack.name} does not have NI enabled`);
      expect(
        screen.queryByText(`${rack.name} does not have OPC UA enabled`),
      ).toBeNull();
    });

    it("should open a connect form at the scoped rack, then go back", async () => {
      const rack = await createRunningRack(client, "opc");
      await open({ rack: rack.key });
      fireEvent.click(await screen.findByText("OPC UA server"));
      await screen.findByText(rack.name);
      fireEvent.click(findButton("Back"));
      expect(await screen.findByText("Connect by address")).toBeTruthy();
    });
  });

  describe("find", () => {
    it("should list only the scoped rack's hardware and configure it in place", async () => {
      const scoped = await createRunningRack(client, "labjack");
      const other = await createRunningRack(client, "labjack");
      const found = await createLabJackDevice(scoped.key);
      const hidden = await createLabJackDevice(other.key);
      await open({ integration: "labjack", rack: scoped.key });
      await screen.findByText(found.name);
      expect(screen.queryByText(hidden.name)).toBeNull();
      expect(screen.queryByText(other.name)).toBeNull();
      fireEvent.click(findButton("Configure"));
      await screen.findByDisplayValue(found.name);
      fireEvent.click(findButton("Back"));
      expect(await screen.findByText(found.name)).toBeTruthy();
    });

    it("should explain a scoped Driver that does not run the integration", async () => {
      const rack = await createRunningRack(client, "opc");
      await open({ integration: "labjack", rack: rack.key });
      expect(
        await screen.findByText(`${rack.name} does not have LabJack enabled`),
      ).toBeTruthy();
    });

    it("should turn on a scanner that is off", async () => {
      const rack = await createRunningRack(client, "labjack");
      const scanner = await rack.createTask({
        name: uniqueName("labjack_scan"),
        type: LabJack.Device.SCAN_TYPE,
        config: { disabled: true },
      });
      await open({ integration: "labjack", rack: rack.key });
      await screen.findByText("Scanning is off on this Driver");
      fireEvent.click(findButton("Turn on"));
      await waitFor(async () => {
        const updated = await client.tasks.retrieve({ key: scanner.key });
        expect(updated.config).toMatchObject({ disabled: false });
      });
    });

    it("should show why a scanner failed", async () => {
      const rack = await createRunningRack(client, "labjack");
      const scanner = await rack.createTask({
        name: uniqueName("labjack_scan"),
        type: LabJack.Device.SCAN_TYPE,
        config: {},
      });
      await client.tasks.create({
        ...scanner.payload,
        status: createTaskStatus({
          variant: "error",
          message: "Failed to start scan task",
          description: "LJM library not found",
          details: { task: scanner.key, rack: rack.key },
        }),
      });
      await open({ integration: "labjack", rack: rack.key });
      await screen.findByText("Failed to start scan task");
      expect(screen.getByText("LJM library not found")).toBeTruthy();
      expect(screen.queryByText("Looking for LabJack hardware")).toBeNull();
    });
  });
});
