// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client";
import { Dialog } from "@synnaxlabs/pluto";
import { id } from "@synnaxlabs/x";
import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { useLink } from "@/service/device/link";
import { NI } from "@/service/ni";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";
import { renderLinkHook } from "@/testutil/testutil";

const client = createTestClient();

const createDevice = async (make: string, name: string) => {
  const rack = await client.racks.create({ name: `test-rack-${id.create()}` });
  return await client.devices.create({
    key: id.create(),
    name,
    rack: rack.key,
    location: "test-location",
    make,
    model: "test-model",
    properties: {},
  });
};

describe("Device.useLink", () => {
  it("should open the configure modal for the linked device", async () => {
    const device = await createDevice(NI.Device.MAKE, "cDAQ Chassis");
    const { handler, modals } = renderLinkHook(useLink);
    expect(modals.isAnyOpen()).toBe(false);
    await handler({ client, key: device.key });
    const [entry] = modals.getState();
    expect(entry).toBeDefined();
    const Wrapper = await createAsyncSynnaxWrapper({ client });
    render(
      <Dialog.Frame variant="modal" visible>
        {entry.render()}
      </Dialog.Frame>,
      { wrapper: Wrapper },
    );
    expect(await screen.findByText("cDAQ Chassis")).toBeTruthy();
  });

  it("should open nothing for a device with an unrecognized make", async () => {
    const device = await createDevice("not-a-real-make", "Mystery Device");
    const { handler, modals } = renderLinkHook(useLink);
    await handler({ client, key: device.key });
    expect(modals.isAnyOpen()).toBe(false);
  });
});
