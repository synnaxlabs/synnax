// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { Icon } from "@synnaxlabs/pluto";
import { id } from "@synnaxlabs/x";
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Device } from "@/platform/device";
import { createTestDevice } from "@/platform/device/testutil";
import { Modals } from "@/platform/modals";
import { findButton, renderModalOpener } from "@/platform/modals/testutil";
import { renderWithConsole, uniqueName } from "@/testutil";

const client = createTestClient();

const useConfigureModal = Modals.create<Device.ConfigureParams>(
  ({ deviceKey, close }) => (
    <Device.Configure
      deviceKey={deviceKey}
      close={close}
      icon={<Icon.Hardware />}
      initialProperties={{ vendor: "acme" }}
    />
  ),
);

const setup = async () => {
  const dev = await createTestDevice(client, {
    configured: false,
    name: uniqueName("my_sensor"),
  });
  const handle = await renderModalOpener(useConfigureModal, [{ deviceKey: dev.key }], {
    client,
  });
  await waitFor(() => expect(screen.getByText(/enter a name/)).toBeTruthy());
  return { dev, ...handle };
};

const nameInput = (): HTMLInputElement => screen.getByRole("textbox");

describe("device Configure", () => {
  it("should not render the form while the device has not been retrieved", async () => {
    await renderWithConsole(
      <Device.Configure
        deviceKey={id.create()}
        close={vi.fn()}
        icon={<Icon.Hardware />}
        initialProperties={{}}
      />,
    );
    expect(screen.queryByText(/enter a name so it's easy to look up later/)).toBeNull();
  });

  it("should stay on the name step and leave the device unconfigured when the name is cleared", async () => {
    const { dev } = await setup();
    await waitFor(() => expect(nameInput().value).toEqual(dev.name));
    fireEvent.change(nameInput(), { target: { value: "" } });
    await act(async () => {
      fireEvent.click(findButton("Next"));
    });
    expect(screen.queryByText(/short identifier/)).toBeNull();
    const unchanged = await client.devices.retrieve(dev.key);
    expect(unchanged.configured).toBe(false);
  });

  it("should keep the modal open and the device unconfigured when the identifier is invalid", async () => {
    const { dev } = await setup();
    await waitFor(() => expect(nameInput().value).toEqual(dev.name));
    await act(async () => {
      fireEvent.click(findButton("Next"));
    });
    await waitFor(() => expect(screen.getByText(/short identifier/)).toBeTruthy());
    fireEvent.change(screen.getByRole("textbox"), { target: { value: "1x" } });
    await act(async () => {
      fireEvent.click(findButton("Save"));
    });
    expect(screen.getByText(/short identifier/)).toBeTruthy();
    const unchanged = await client.devices.retrieve(dev.key);
    expect(unchanged.configured).toBe(false);
  });

  it("should configure the device with the edited name, identifier, and initial properties, then close", async () => {
    const { dev } = await setup();
    await waitFor(() => expect(nameInput().value).toEqual(dev.name));
    const newName = uniqueName("renamed_sensor");
    fireEvent.change(nameInput(), { target: { value: newName } });
    await act(async () => {
      fireEvent.click(findButton("Next"));
    });
    await waitFor(() => expect(screen.getByText(/short identifier/)).toBeTruthy());
    fireEvent.change(screen.getByRole("textbox"), { target: { value: "sensor_1" } });
    await act(async () => {
      fireEvent.click(findButton("Save"));
    });
    await waitFor(async () => {
      const updated = await client.devices.retrieve(dev.key);
      expect(updated.configured).toBe(true);
      expect(updated.name).toEqual(newName);
      expect(updated.properties.identifier).toEqual("sensor_1");
      expect(updated.properties.vendor).toEqual("acme");
    });
    await waitFor(() => expect(screen.queryByText(/short identifier/)).toBeNull());
  });
});
