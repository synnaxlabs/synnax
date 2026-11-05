// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Errors } from "@synnaxlabs/pluto";
import { deep } from "@synnaxlabs/x";
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";

import { Device } from "@/platform/device";
import { createTestDevice, renderWithDeviceForm } from "@/platform/device/testutil";
import { uniqueName } from "@/testutil";

const client = createTestClient();

const path = deep.path<{ rack: number; config: { device: string } }>();

const FormProbe = (): ReactElement => {
  const deviceKey = Form.useFieldValue(path.config.device);
  const rack = Form.useFieldValue(path.rack, { optional: true });
  return <div>{`device:${deviceKey} rack:${rack ?? "none"}`}</div>;
};
FormProbe.displayName = "FormProbe";

interface RenderNoneSelectedOptions extends Pick<
  Device.NoneSelectedProps,
  "make" | "connect"
> {
  onConfigure?: Device.NoneSelectedProps["onConfigure"];
  mode?: Form.Mode;
}

const renderNoneSelected = async ({
  make,
  connect,
  onConfigure = vi.fn(),
  mode,
}: RenderNoneSelectedOptions): ReturnType<typeof renderWithDeviceForm> =>
  await renderWithDeviceForm(
    <>
      <Device.Select make={make} icon={<Icon.Device />} onConfigure={vi.fn()} />
      <Errors.SuspenseBoundary>
        <Device.NoneSelected
          make={make}
          emptyContent="No test devices connected"
          connect={connect}
          onConfigure={onConfigure}
        />
      </Errors.SuspenseBoundary>
      <FormProbe />
    </>,
    { deviceKey: "", client, mode },
  );

const DEVICE = { key: "dev-key", name: "My Device" };

describe("device empty states", () => {
  describe("NoneSelected", () => {
    it("should offer to connect a device when no device of the make exists", async () => {
      const open = vi.fn(async () => await Promise.resolve(null));
      await renderNoneSelected({
        make: uniqueName("make"),
        connect: { title: "test hardware", open },
      });
      await screen.findByText("No test devices connected");
      fireEvent.click(await screen.findByText("Connect test hardware"));
      await waitFor(() => expect(open).toHaveBeenCalledTimes(1));
      expect(screen.getByText("device: rack:none")).toBeTruthy();
    });

    it("should select the device that its connect action resolves with", async () => {
      const make = uniqueName("make");
      let rack = 0;
      const open = vi.fn(async () => {
        const dev = await createTestDevice(client, { make });
        rack = dev.rack;
        return dev.key;
      });
      await renderNoneSelected({ make, connect: { title: "test hardware", open } });
      fireEvent.click(await screen.findByText("Connect test hardware"));
      const key = await open.mock.results[0].value;
      await screen.findByText(`device:${key} rack:${rack}`);
    });

    it("should configure a connected device that is not configured", async () => {
      const make = uniqueName("make");
      const onConfigure = vi.fn();
      const open = vi.fn(
        async () => (await createTestDevice(client, { make, configured: false })).key,
      );
      await renderNoneSelected({
        make,
        connect: { title: "test hardware", open },
        onConfigure,
      });
      fireEvent.click(await screen.findByText("Connect test hardware"));
      const key = await open.mock.results[0].value;
      await waitFor(() => expect(onConfigure).toHaveBeenCalledWith(key));
    });

    it("should open the device select when a device of the make exists", async () => {
      const make = uniqueName("make");
      const dev = await createTestDevice(client, { make });
      await renderNoneSelected({
        make,
        connect: { title: "test hardware", open: vi.fn() },
      });
      expect(screen.queryByText(dev.name)).toBeNull();
      const message = await screen.findByText("No device selected");
      fireEvent.click(within(message).getByText("Select device"));
      await screen.findByText(dev.name);
    });

    it("should show a failed device query in the error boundary", async () => {
      const Fallback = ({ error }: Errors.FallbackProps): ReactElement => (
        <p>{`failed: ${error.message}`}</p>
      );
      await renderWithDeviceForm(
        <Errors.SuspenseBoundary FallbackComponent={Fallback}>
          <Device.NoneSelected
            make={uniqueName("make")}
            emptyContent="No test devices connected"
            connect={{ title: "test hardware", open: vi.fn() }}
            onConfigure={vi.fn()}
          />
        </Errors.SuspenseBoundary>,
        { deviceKey: "", client: null, onCaughtError: () => {} },
      );
      await screen.findByText("failed: Cannot retrieve devices: no Core connected.");
    });

    it("should not offer to connect a device in a read-only form", async () => {
      await renderNoneSelected({
        make: uniqueName("make"),
        connect: { title: "test hardware", open: vi.fn() },
        mode: "preview",
      });
      await screen.findByText("No test devices connected");
      expect(screen.queryByText("Connect test hardware")).toBeNull();
    });

    it("should not offer to select a device in a read-only form", async () => {
      const make = uniqueName("make");
      await createTestDevice(client, { make });
      await renderNoneSelected({
        make,
        connect: { title: "test hardware", open: vi.fn() },
        mode: "preview",
      });
      const message = await screen.findByText("No device selected");
      expect(within(message).queryByText("Select device")).toBeNull();
    });
  });

  it("should prompt to configure and hand the device key to onConfigure", () => {
    const onConfigure = vi.fn();
    render(
      <Device.Unconfigured device={DEVICE} canConfigure onConfigure={onConfigure} />,
    );
    expect(screen.getByText(`${DEVICE.name} is not configured`)).toBeTruthy();
    fireEvent.click(screen.getByText(`Configure ${DEVICE.name}`));
    expect(onConfigure).toHaveBeenCalledWith(DEVICE.key);
  });

  it("should hide the configure action when configuration is not allowed", () => {
    render(
      <Device.Unconfigured
        device={DEVICE}
        canConfigure={false}
        onConfigure={vi.fn()}
      />,
    );
    expect(screen.getByText(`${DEVICE.name} is not configured`)).toBeTruthy();
    expect(screen.queryByText(`Configure ${DEVICE.name}`)).toBeNull();
  });
});
