// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { uniqueName } from "@synnaxlabs/x/testutil";
import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Serial } from "@/feature/serial";
import { createTestSink } from "@/platform/analytics/testutil";
import { createRunningRack, createTestDevice } from "@/platform/device/testutil";
import { pressSaveTrigger, renderModalOpener } from "@/platform/modals/testutil";
import { commitFieldInput, selectFromDropdown } from "@/platform/task/testutil";
import { findDialogTriggerByText } from "@/testutil";

const client = createTestClient();

const PROPERTIES = {
  port: "/dev/ttyUSB3",
  baudRate: 115200,
  dataBits: 7,
  parity: "even",
  stopBits: "2",
  flowControl: "hardware",
} as const;

describe("Serial.Device.useConnectModal", () => {
  it("should populate the form from an existing device", async () => {
    const dev = await createTestDevice(client, {
      make: Serial.Device.MAKE,
      model: Serial.Device.MAKE,
      properties: PROPERTIES,
    });
    await renderModalOpener(Serial.Device.useConnectModal, [{ deviceKey: dev.key }], {
      client,
    });
    await screen.findByDisplayValue(dev.name);
    expect(screen.getByDisplayValue("/dev/ttyUSB3")).toBeTruthy();
    expect(screen.getByDisplayValue("115200")).toBeTruthy();
    expect(screen.getByDisplayValue("7")).toBeTruthy();
    expect(await findDialogTriggerByText("Even")).toBeTruthy();
    expect(await findDialogTriggerByText("2")).toBeTruthy();
    expect(await findDialogTriggerByText("RTS/CTS")).toBeTruthy();
  });

  it("should start with an empty, focused name", async () => {
    await renderModalOpener(Serial.Device.useConnectModal, [{}], { client });
    const name = await screen.findByPlaceholderText("Device name");
    expect(name).toHaveProperty("value", "");
    expect(document.activeElement).toBe(name);
  });

  it("should save the trimmed port and line settings of a new device and report it", async () => {
    const rack = await createRunningRack(client, Serial.Device.INTEGRATION_KEY);
    const analytics = createTestSink();
    const { result } = await renderModalOpener(
      Serial.Device.useConnectModal,
      [{ rack: rack.key }],
      { client, analytics },
    );
    const name = uniqueName("serial");
    fireEvent.change(await screen.findByPlaceholderText("Device name"), {
      target: { value: name },
    });
    fireEvent.change(screen.getByPlaceholderText("/dev/ttyUSB0"), {
      target: { value: " /dev/ttyUSB3\n" },
    });
    commitFieldInput(screen.getByDisplayValue<HTMLInputElement>("9600"), "115200");
    commitFieldInput(screen.getByDisplayValue<HTMLInputElement>("8"), "7");
    await selectFromDropdown("None", "Even");
    await selectFromDropdown("1", "2");
    await selectFromDropdown("None", "RTS/CTS");
    pressSaveTrigger();
    const key = await result();
    if (key == null) throw new Error("the modal closed without a device");
    const dev = await client.devices.retrieve({ key });
    expect(dev).toMatchObject({
      name,
      make: "serial",
      model: "serial",
      rack: rack.key,
      location: "/dev/ttyUSB3",
      properties: PROPERTIES,
    });
    await waitFor(() =>
      expect(analytics.capture).toHaveBeenCalledWith("device_connected", {
        integration: "serial",
      }),
    );
  });

  it("should show the schema error of a blank port at its input", async () => {
    const rack = await createRunningRack(client, Serial.Device.INTEGRATION_KEY);
    await renderModalOpener(Serial.Device.useConnectModal, [{ rack: rack.key }], {
      client,
    });
    fireEvent.change(await screen.findByPlaceholderText("Device name"), {
      target: { value: uniqueName("serial") },
    });
    fireEvent.change(screen.getByPlaceholderText("/dev/ttyUSB0"), {
      target: { value: "   " },
    });
    pressSaveTrigger();
    expect(
      await screen.findByText(/expected string to have >=1 characters/),
    ).toBeTruthy();
  });

  it("should put the port with the baud rate and the line settings in pairs", async () => {
    await renderModalOpener(Serial.Device.useConnectModal, [{}], { client });
    const box = (el: HTMLElement) => el.getBoundingClientRect();
    const sameRow = (a: DOMRect, b: DOMRect) => a.top < b.bottom && b.top < a.bottom;
    const port = box(await screen.findByPlaceholderText("/dev/ttyUSB0"));
    const baudRate = box(screen.getByDisplayValue("9600"));
    const dataBits = box(screen.getByDisplayValue("8"));
    const stopBits = box(await findDialogTriggerByText("1"));
    // Parity and flow control both start at None, in that order.
    const [parity, flowControl] = screen
      .getAllByRole("button")
      .filter(
        (b) =>
          b.getAttribute("aria-haspopup") === "dialog" &&
          within(b).queryAllByText("None").length > 0,
      )
      .map(box);
    expect(sameRow(port, baudRate)).toBe(true);
    expect(sameRow(dataBits, parity)).toBe(true);
    expect(sameRow(stopBits, flowControl)).toBe(true);
    expect(dataBits.top).toBeGreaterThanOrEqual(port.bottom);
    expect(stopBits.top).toBeGreaterThanOrEqual(dataBits.bottom);
  });
});
