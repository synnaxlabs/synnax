// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax as Client } from "@synnaxlabs/client";
import { describe, expect, it, vi } from "vitest";

import { Device } from "@/hardware/device";
import { Layout } from "@/layout";
import { renderLinkHook } from "@/testUtils";

const clientReturning = (device: object): Client =>
  ({ devices: { retrieve: vi.fn(async () => device) } }) as unknown as Client;

describe("Device.useLink", () => {
  it("should place a configure layout for a device with a known make", async () => {
    const key = "device-1";
    const client = clientReturning({ key, name: "cDAQ Chassis", make: "NI" });
    const { handler, store } = renderLinkHook(Device.useLink);
    await handler({ client, key });
    expect(Layout.select(store.getState(), key)?.name).toBe("cDAQ Chassis");
  });

  it("should place nothing for a device with an unknown make", async () => {
    const key = "device-2";
    const client = clientReturning({ key, name: "Mystery", make: "not-a-make" });
    const { handler, store } = renderLinkHook(Device.useLink);
    await handler({ client, key });
    expect(Layout.select(store.getState(), key)).toBeUndefined();
  });
});
