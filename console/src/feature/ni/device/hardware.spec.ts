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

import { NI } from "@/feature/ni";
import { Device } from "@/platform/device";
import { createTestDevice } from "@/platform/device/testutil";
import { findButton, renderModalOpener } from "@/platform/modals/testutil";

const client = createTestClient();

describe("NI.Device.HARDWARE.Configure", () => {
  it("should show the NI MAX name of the device and suggest identifiers from it", async () => {
    const dev = await createTestDevice(client, {
      configured: false,
      make: NI.Device.MAKE,
      location: "cDAQ1Mod1",
    });
    await renderModalOpener(
      Device.useConfigureModal,
      [{ deviceKey: dev.key, hardware: NI.Device.HARDWARE }],
      { client },
    );
    await screen.findByText("cDAQ1Mod1");
    findButton("c1m1");
    fireEvent.click(findButton("cdaq1mod1"));
    expect(
      screen.getByRole<HTMLInputElement>("textbox", { name: /^Identifier/ }).value,
    ).toEqual("cdaq1mod1");
  });
});
