// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client";
import { id } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { Device } from "@/hardware/device";
import { NI } from "@/hardware/ni";
import { Layout } from "@/layout";
import { renderLinkHook } from "@/testUtils";

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
  it("should place a configure layout for a device with a known make", async () => {
    const device = await createDevice(NI.Device.MAKE, "cDAQ Chassis");
    const { handler, store } = renderLinkHook(Device.useLink);
    await handler({ client, key: device.key });
    expect(Layout.select(store.getState(), device.key)?.name).toBe("cDAQ Chassis");
  });

  it("should place nothing for a device with an unrecognized make", async () => {
    const device = await createDevice("not-a-real-make", "Mystery Device");
    const { handler, store } = renderLinkHook(Device.useLink);
    await handler({ client, key: device.key });
    expect(Layout.select(store.getState(), device.key)).toBeUndefined();
  });
});
