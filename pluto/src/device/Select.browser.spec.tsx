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
import type { FC, PropsWithChildren } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { page } from "vitest/browser";
import { render } from "vitest-browser-react";

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
    const c = await render(<Device.SelectSingle value={dev.key} onChange={vi.fn()} />, {
      wrapper,
    });
    const badge = c.getByText("Connected");
    await expect.element(badge).toHaveClass("lyra--status-error");
    expect(badge.element().closest(`.${END_CLASS}`)).not.toBeNull();
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
    const c = await render(
      <Device.SelectSingle
        value={undefined}
        onChange={vi.fn()}
        query={{ where: { make: { eq: make } } }}
        locationFormat={{ shape: "address", hiddenSchemes: ["opc.tcp://"] }}
        virtual={false}
      />,
      { wrapper },
    );
    await c.getByRole("button", { name: "Device" }).click();
    await expect.element(page.getByText(dev.name)).toBeInTheDocument();
    await expect.element(page.getByText("10.0.0.5")).toBeInTheDocument();
    await expect.element(page.getByText(/opc\.tcp/)).not.toBeInTheDocument();
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
    const c = await render(
      <Device.SelectSingle
        value={undefined}
        onChange={vi.fn()}
        query={{ where: { make: { eq: make } } }}
        virtual={false}
      />,
      { wrapper: await createAsyncSynnaxWrapper({ client: local }) },
    );
    const trigger = c.getByRole("button", { name: "Device" });
    await trigger.click();
    await expect.element(page.getByText("boom")).toBeInTheDocument();
    await trigger.click();
    reject.mockRestore();
    await trigger.click();
    await expect.element(page.getByText(dev.name)).toBeInTheDocument();
  });

  it("should show no status without a selected device", async () => {
    const c = await render(
      <Device.SelectSingle value={undefined} onChange={vi.fn()} />,
      {
        wrapper,
      },
    );
    await expect.element(c.getByRole("button", { name: "Device" })).toBeInTheDocument();
    expect(c.container.querySelector(`.${END_CLASS}`)).toBeNull();
  });

  it("should call a device by the given resource name", async () => {
    const c = await render(
      <Device.SelectSingle
        value={undefined}
        onChange={vi.fn()}
        resourceName="connection"
      />,
      { wrapper },
    );
    await c.getByRole("button", { name: "Connection" }).click();
    await expect
      .element(page.getByPlaceholder("Search connections..."))
      .toBeInTheDocument();
  });

  it("should keep the status beside the caller's trigger props", async () => {
    const dev = await createDevice();
    const c = await render(
      <Device.SelectSingle
        value={dev.key}
        onChange={vi.fn()}
        triggerProps={{ className: "custom" }}
      />,
      { wrapper },
    );
    const badge = c.getByText("Connected");
    await expect.element(badge).toBeInTheDocument();
    expect(badge.element().closest(".custom")).not.toBeNull();
  });
});
