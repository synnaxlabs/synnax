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

const openConfigure = async (location: string, name?: string): Promise<void> => {
  const dev = await createTestDevice(client, {
    configured: false,
    make: NI.Device.MAKE,
    location,
    name,
  });
  await renderModalOpener(
    Device.useConfigureModal,
    [{ deviceKey: dev.key, hardware: NI.Device.HARDWARE }],
    { client },
  );
  await screen.findByText(location);
};

describe("NI.Device.HARDWARE.Configure", () => {
  it("should show the NI MAX name of the device and suggest identifiers from it", async () => {
    await openConfigure("cDAQ1Mod1");
    findButton("c1m1");
    fireEvent.click(findButton("cdaq1mod1"));
    expect(
      screen.getByRole<HTMLInputElement>("textbox", { name: /^Identifier/ }).value,
    ).toEqual("cdaq1mod1");
  });

  it("should leave the chassis serial number out of the NI MAX name of a module", async () => {
    await openConfigure("cDAQ9189-1D71E7CMod1");
    findButton("cdaq9189mod1");
    findButton("c9189m1");
  });

  it("should suggest identifiers from the device name when no form of the NI MAX name fits", async () => {
    await openConfigure("9205_1", "NI 9205");
    findButton("n9205");
  });
});
