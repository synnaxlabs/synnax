// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Form } from "@synnaxlabs/pluto";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";

import { Device } from "@/platform/device";
import { createTestDevice, renderWithDeviceForm } from "@/platform/device/testutil";
import { findDialogTrigger, uniqueName } from "@/testutil";

const client = createTestClient();

const RackProbe = (): ReactElement => {
  const value = Form.useFieldValue<number>("rack", { optional: true });
  return <div>{`rack:${value ?? "none"}`}</div>;
};
RackProbe.displayName = "RackProbe";

const renderSelect = async (
  props: Device.SelectProps,
): ReturnType<typeof renderWithDeviceForm> =>
  await renderWithDeviceForm(
    <>
      <Device.Select {...props} />
      <RackProbe />
    </>,
    { deviceKey: "", client },
  );

const openAndClick = async (name: string): Promise<void> => {
  fireEvent.click(await findDialogTrigger());
  fireEvent.click(await screen.findByText(name));
};

describe("Device.Select", () => {
  it("should write the selected device's rack into the form and skip onConfigure for a configured device", async () => {
    const make = uniqueName("make");
    const dev = await createTestDevice(client, { make, configured: true });
    const onConfigure = vi.fn();
    await renderSelect({ make, onConfigure });
    await openAndClick(dev.name);
    await waitFor(() => expect(screen.getByText(`rack:${dev.rack}`)).toBeTruthy());
    const trigger = await findDialogTrigger();
    expect(trigger.textContent).toContain(dev.name);
    expect(onConfigure).not.toHaveBeenCalled();
  });

  it("should call onConfigure with the device key when an unconfigured device is selected", async () => {
    const make = uniqueName("make");
    const dev = await createTestDevice(client, { make, configured: false });
    const onConfigure = vi.fn();
    await renderSelect({ make, onConfigure });
    await openAndClick(dev.name);
    await waitFor(() => expect(onConfigure).toHaveBeenCalledWith(dev.key));
  });

  it("should only list devices matching the given make and model", async () => {
    const make = uniqueName("make");
    const matching = await createTestDevice(client, { make, model: "m1" });
    const other = await createTestDevice(client, { make, model: "m2" });
    await renderSelect({ make, model: "m1", onConfigure: vi.fn() });
    fireEvent.click(await findDialogTrigger());
    await screen.findByText(matching.name);
    expect(screen.queryByText(other.name)).toBeNull();
  });

  it("should intersect the base make filter with a custom filter prop", async () => {
    const make = uniqueName("make");
    const excluded = await createTestDevice(client, { make });
    const included = await createTestDevice(client, { make });
    const filter = (d: device.Device): boolean => d.key === included.key;
    await renderSelect({ make, filter, onConfigure: vi.fn() });
    fireEvent.click(await findDialogTrigger());
    await screen.findByText(included.name);
    expect(screen.queryByText(excluded.name)).toBeNull();
  });

  it("should show the empty content when no devices match the make", async () => {
    await renderSelect({ make: uniqueName("make"), onConfigure: vi.fn() });
    fireEvent.click(await findDialogTrigger());
    await screen.findByText("No devices connected.");
  });
});
