// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Modbus } from "@/feature/modbus";
import { createModbusDevice } from "@/feature/modbus/testutil";
import { createTestSink } from "@/platform/analytics/testutil";
import { pressSaveTrigger, renderModalOpener } from "@/platform/modals/testutil";
import { answerNextCommand, selectFromDropdown } from "@/platform/task/testutil";
import { uniqueName } from "@/testutil";

const client = createTestClient();

describe("Modbus.Device.useConnectModal", () => {
  it("should populate the form from an existing device", async () => {
    const dev = await createModbusDevice(client, {
      properties: {
        connection: {
          host: "modbus-existing.local",
          port: 1502,
          swapBytes: true,
          swapWords: false,
        },
      },
    });
    await renderModalOpener(Modbus.Device.useConnectModal, [{ deviceKey: dev.key }], {
      client,
    });
    await screen.findByDisplayValue(dev.name);
    expect(screen.getByDisplayValue("modbus-existing.local")).toBeTruthy();
    expect(screen.getByDisplayValue("1502")).toBeTruthy();
  });

  it("should report a device it connected", async () => {
    const rack = await client.racks.create({
      name: uniqueName("modbus_rack"),
      integrations: ["modbus"],
    });
    const scan = await rack.createTask({
      name: uniqueName("modbus_scan"),
      type: Modbus.Task.SCAN_TYPE,
      config: {},
    });
    const analytics = createTestSink();
    await renderModalOpener(Modbus.Device.useConnectModal, [{}], { client, analytics });
    await selectFromDropdown("Select Driver", rack.name);
    fireEvent.change(screen.getByPlaceholderText("localhost"), {
      target: { value: "localhost" },
    });
    const { answered } = await answerNextCommand(client, scan);
    pressSaveTrigger();
    await answered;
    await waitFor(() =>
      expect(analytics.capture).toHaveBeenCalledWith("device_connected", {
        integration: "modbus",
      }),
    );
  });

  // Submitting with no rack chosen fails validation. That error is the proof the keys
  // reached the same save path the Connect button uses.
  it("should submit on the shortcut its footer advertises", async () => {
    await renderModalOpener(Modbus.Device.useConnectModal, [{}], { client });
    await screen.findByRole("dialog");
    pressSaveTrigger();
    expect(await screen.findByText(/rack is required/i)).toBeTruthy();
  });
});
