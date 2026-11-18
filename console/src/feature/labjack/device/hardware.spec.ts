// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { LabJack } from "@/feature/labjack";
import { Device } from "@/platform/device";
import { createTestDevice } from "@/platform/device/testutil";
import { findButton, renderModalOpener } from "@/platform/modals/testutil";

const client = createTestClient();

describe("LabJack.Device.HARDWARE.Configure", () => {
  it("should suggest identifiers from the model and serial number of the device", async () => {
    const serial = `47${Math.floor(Math.random() * 1e7)}`.padEnd(9, "0");
    const dev = await createTestDevice(client, {
      key: serial,
      name: `LJM_dtT7-${serial.slice(-4)}`,
      configured: false,
      make: LabJack.Device.MAKE,
      model: LabJack.Device.T7_MODEL,
      location: "LJM_ctUSB",
    });
    await renderModalOpener(
      Device.useConfigureModal,
      [{ deviceKey: dev.key, hardware: LabJack.Device.HARDWARE }],
      { client },
    );
    await screen.findByText("LJM_ctUSB");
    findButton(`t7_${serial.slice(-4)}`);
    fireEvent.click(findButton("t7"));
    expect(
      screen.getByRole<HTMLInputElement>("textbox", { name: /^Identifier/ }).value,
    ).toEqual("t7");
  });
});
