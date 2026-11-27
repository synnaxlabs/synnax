// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ni, status } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Form } from "@synnaxlabs/lyra/form";
import { deep, id } from "@synnaxlabs/x";
import { uniqueName } from "@synnaxlabs/x/testutil";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import type { ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";

import { NI } from "@/feature/ni";
import { Device } from "@/platform/device";
import { renderWithDeviceForm } from "@/platform/device/testutil";
import { findDialogTrigger } from "@/testutil";

const path = deep.path<{ rack: number }>();

const client = createTestClient();

const RackProbe = (): ReactElement => {
  const value = Form.useFieldValue(path.rack, { optional: true });
  return <div>{`rack:${value ?? "none"}`}</div>;
};
RackProbe.displayName = "RackProbe";

const renderSelect = async (onConfigure: (key: string) => void) =>
  await renderWithDeviceForm(
    <>
      <Device.Select
        make={NI.Device.MAKE}
        icon={NI.Device.TYPED_HARDWARE.icon}
        onConfigure={onConfigure}
        source={NI.Device.TYPED_SOURCE}
      />
      <RackProbe />
    </>,
    { deviceKey: "", client },
  );

describe("NI.Device.TYPED_SOURCE", async () => {
  const rack = await client.racks.create({ name: uniqueName("ni_rack") });
  const create = async (identifier: string) =>
    await client.ni.devices.create({
      rack: rack.key,
      name: uniqueName("ni_dev"),
      resourceName: id.create(),
      model: "PXIe-4300",
      identifier,
    });

  it("should set the rack and open the configure form for a device without an identifier", async () => {
    const dev = await create("");
    const onConfigure = vi.fn();
    await renderSelect(onConfigure);
    fireEvent.click(await findDialogTrigger());
    fireEvent.click(await screen.findByText(dev.name));
    await waitFor(() => expect(screen.getByText(`rack:${rack.key}`)).toBeTruthy());
    await waitFor(() => expect(onConfigure).toHaveBeenCalledWith(dev.key));
  });

  it("should set the rack and skip the configure form for a configured device", async () => {
    const dev = await create("dev1");
    const onConfigure = vi.fn();
    await renderSelect(onConfigure);
    fireEvent.click(await findDialogTrigger());
    fireEvent.click(await screen.findByText(dev.name));
    await waitFor(() => expect(screen.getByText(`rack:${rack.key}`)).toBeTruthy());
    expect((await findDialogTrigger()).textContent).toContain(dev.name);
    expect(onConfigure).not.toHaveBeenCalled();
  });

  it("should badge the selected device from its connected flag", async () => {
    const dev = await create("dev2");
    await renderSelect(vi.fn());
    fireEvent.click(await findDialogTrigger());
    fireEvent.click(await screen.findByText(dev.name));
    await waitFor(async () =>
      expect((await findDialogTrigger()).textContent).toContain("Disconnected"),
    );
    await client.statuses.set(
      status.create<typeof ni.device.statusDetailsZ>({
        owner: ni.device.resourceID(dev.key),
        variant: "warning",
        message: "Device connected",
        details: { rack: rack.key, device: dev.key, connected: true },
      }),
    );
    await waitFor(async () =>
      expect((await findDialogTrigger()).textContent).toContain("Connected"),
    );
    expect((await findDialogTrigger()).textContent).not.toContain("Disconnected");
  });
});
