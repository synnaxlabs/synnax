// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { device, status } from "@synnaxlabs/client";
import { createTestClient, rejectRequests } from "@synnaxlabs/client/testutil";
import { id } from "@synnaxlabs/x";
import { fireEvent, render } from "@testing-library/react";
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
        owner: device.resourceID(dev.key),
        variant: "error",
        message: "Health check failed",
        details: { rack: rack.key, connected: true },
      }),
    );
    return dev;
  };

  it("should show the selected device's status at the trigger's end", async () => {
    const dev = await createDevice();
    const c = render(<Device.SelectSingle value={dev.key} onChange={vi.fn()} />, {
      wrapper,
    });
    const badge = await c.findByText("Connected");
    expect(badge.classList).toContain("lyra--status-error");
    expect(badge.closest(`.${END_CLASS}`)).not.toBeNull();
  });

  it("should list each device with its location in the given format", async () => {
    const rack = await client.racks.create({ name: "test" });
    const make = id.create();
    const dev = await client.devices.create({
      key: id.create(),
      name: id.create(),
      rack: rack.key,
      location: "opc.tcp://10.0.0.5:4840",
      make,
      model: "test",
      properties: {},
    });
    const c = render(
      <Device.SelectSingle
        value={undefined}
        onChange={vi.fn()}
        query={{ where: { make: { eq: make } } }}
        locationFormat={{ shape: "address", hiddenSchemes: ["opc.tcp://"] }}
        virtual={false}
      />,
      { wrapper },
    );
    fireEvent.click(c.getByRole("button", { name: "Device" }));
    expect(await c.findByText(dev.name)).toBeTruthy();
    expect(c.getByText("10.0.0.5")).toBeTruthy();
    expect(c.queryByText(/opc\.tcp/)).toBeNull();
  });

  it("should read the devices again when the dialog reopens after a failed read", async () => {
    const rack = await client.racks.create({ name: "test" });
    const make = id.create();
    const dev = await client.devices.create({
      key: id.create(),
      name: id.create(),
      rack: rack.key,
      location: "test",
      make,
      model: "test",
      properties: {},
    });
    const local = createTestClient();
    const reject = rejectRequests(
      local,
      (target) => target === "/device/retrieve",
      new Error("boom"),
    );
    const c = render(
      <Device.SelectSingle
        value={undefined}
        onChange={vi.fn()}
        query={{ where: { make: { eq: make } } }}
        virtual={false}
      />,
      { wrapper: await createAsyncSynnaxWrapper({ client: local }) },
    );
    const trigger = c.getByRole("button", { name: "Device" });
    fireEvent.click(trigger);
    expect(await c.findByText("boom")).toBeTruthy();
    fireEvent.click(trigger);
    reject.mockRestore();
    fireEvent.click(trigger);
    expect(await c.findByText(dev.name)).toBeTruthy();
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
    const badge = await c.findByText("Connected");
    expect(badge.closest(".custom")).not.toBeNull();
  });
});
