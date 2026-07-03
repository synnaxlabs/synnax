// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Device } from "@/platform/device";
import { createTestDevice, renderWithDeviceForm } from "@/platform/device/testutil";

const client = createTestClient();

describe("device Provider", () => {
  it("should render custom none-selected content when no device is set", async () => {
    await renderWithDeviceForm(
      <Device.Provider
        canConfigure
        onConfigure={vi.fn()}
        noneSelectedContent={<div>pick something</div>}
      >
        {({ device }) => <div>{device.name}</div>}
      </Device.Provider>,
      { deviceKey: "" },
    );
    expect(screen.getByText("pick something")).toBeTruthy();
  });

  it("should gate on selection: none-selected content until a device is picked, then the children with that device", async () => {
    const dev = await createTestDevice(client, { configured: true });
    await renderWithDeviceForm(
      <Device.Provider canConfigure onConfigure={vi.fn()}>
        {({ device }) => <div>rendered {device.name}</div>}
      </Device.Provider>,
      { deviceKey: "", selectableKey: dev.key, client },
    );
    expect(screen.getByText("No device selected.")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "select device" }));
    await waitFor(() => expect(screen.getByText(`rendered ${dev.name}`)).toBeTruthy());
    expect(screen.queryByText("No device selected.")).toBeNull();
  });

  it("should prompt to configure an unconfigured device and hand its key to onConfigure", async () => {
    const dev = await createTestDevice(client, { configured: false });
    const onConfigure = vi.fn();
    await renderWithDeviceForm(
      <Device.Provider canConfigure onConfigure={onConfigure}>
        {({ device }) => <div>rendered {device.name}</div>}
      </Device.Provider>,
      { deviceKey: dev.key, client },
    );
    await waitFor(() => expect(screen.getByText(`Configure ${dev.name}`)).toBeTruthy());
    expect(screen.queryByText(`rendered ${dev.name}`)).toBeNull();
    fireEvent.click(screen.getByText(`Configure ${dev.name}`));
    expect(onConfigure).toHaveBeenCalledWith(dev.key);
  });

  it("should hide the configure action when configuration is not allowed", async () => {
    const dev = await createTestDevice(client, { configured: false });
    await renderWithDeviceForm(
      <Device.Provider canConfigure={false} onConfigure={vi.fn()}>
        {({ device }) => <div>rendered {device.name}</div>}
      </Device.Provider>,
      { deviceKey: dev.key, client },
    );
    await waitFor(() =>
      expect(screen.getByText(`${dev.name} is not configured.`)).toBeTruthy(),
    );
    expect(screen.queryByText(`Configure ${dev.name}`)).toBeNull();
  });
});
