// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { device, status } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { id } from "@synnaxlabs/x";
import { render } from "@testing-library/react";
import { type FC, type PropsWithChildren } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { Device } from "@/device";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();

const END_CLASS = "lyra-select-single-trigger__end";

describe("Device select", () => {
  let wrapper: FC<PropsWithChildren>;
  beforeEach(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
  });

  const createDevice = async (): Promise<device.Device> => {
    const rack = await client.racks.create({ name: "test" });
    const dev = await client.devices.create({
      key: id.create(),
      name: "Boiler PLC",
      rack: rack.key,
      location: "test",
      make: "test",
      model: "test",
      properties: {},
    });
    await client.statuses.set(
      status.create<typeof device.statusDetailsZ>({
        key: device.statusKey(dev.key),
        variant: "error",
        message: "Failed to reach device",
        details: { rack: rack.key, device: dev.key, connected: false },
      }),
    );
    return dev;
  };

  it("should show the selected device's status at the trigger's end", async () => {
    const dev = await createDevice();
    const c = render(<Device.SelectSingle value={dev.key} onChange={vi.fn()} />, {
      wrapper,
    });
    const badge = await c.findByText("Disconnected");
    expect(badge.closest(`.${END_CLASS}`)).not.toBeNull();
  });

  it("should show no status without a selected device", async () => {
    const c = render(<Device.SelectSingle value={undefined} onChange={vi.fn()} />, {
      wrapper,
    });
    await c.findByRole("button", { name: "Device" });
    expect(c.container.querySelector(`.${END_CLASS}`)).toBeNull();
  });

  it("should keep the status beside the caller's trigger props", async () => {
    const dev = await createDevice();
    const c = render(
      <Device.SelectSingle
        value={dev.key}
        onChange={vi.fn()}
        triggerProps={{ className: "custom" }}
      />,
      { wrapper },
    );
    const badge = await c.findByText("Disconnected");
    expect(badge.closest(".custom")).not.toBeNull();
  });
});
