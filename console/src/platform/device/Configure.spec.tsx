// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { Icon } from "@synnaxlabs/lyra/icon";
import { id, strings } from "@synnaxlabs/x";
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { type Analytics } from "@/platform/analytics";
import { createTestSink } from "@/platform/analytics/testutil";
import { Device } from "@/platform/device";
import { createTestDevice } from "@/platform/device/testutil";
import { Errors } from "@/platform/errors";
import { Modals } from "@/platform/modals";
import { findButton, renderModalOpener } from "@/platform/modals/testutil";
import { renderWithConsole, uniqueName } from "@/testutil";

const client = createTestClient();

const HINT = /A short prefix for every channel/;

const onCreateTask = vi.fn();
const onBack = vi.fn();

const HARDWARE: Device.Hardware = {
  key: "acme",
  name: "Acme",
  title: "Acme hardware",
  description: "",
  icon: <Icon.Hardware />,
  make: "acme",
  scanType: "acme_scan",
  checks: [],
  zeroProperties: { vendor: "acme" },
};

const useConfigureWithActionsModal = Modals.create<Device.ConfigureModalParams>(
  ({ deviceKey, close }) => (
    <Device.Configure
      deviceKey={deviceKey}
      close={close}
      icon={<Icon.Hardware />}
      initialProperties={{}}
      onCreateTask={onCreateTask}
      onBack={onBack}
    />
  ),
);

const setup = async (
  useOpen: typeof Device.useConfigureModal = Device.useConfigureModal,
  analytics?: Analytics.Sink,
) => {
  const dev = await createTestDevice(client, {
    configured: false,
    name: uniqueName("my_sensor"),
    make: "acme",
  });
  const handle = await renderModalOpener(
    useOpen,
    [{ deviceKey: dev.key, hardware: HARDWARE }],
    {
      client,
      analytics,
    },
  );
  await waitFor(() => expect(nameInput().value).toEqual(dev.name));
  return { dev, ...handle };
};

const nameInput = (): HTMLInputElement =>
  screen.getByRole<HTMLInputElement>("textbox", { name: /^Name/ });
const identifierInput = (): HTMLInputElement =>
  screen.getByRole<HTMLInputElement>("textbox", { name: /^Identifier/ });

const save = async (label = "Save") => {
  await act(async () => {
    fireEvent.click(findButton(label));
  });
};

describe("device Configure", () => {
  beforeEach(() => {
    onCreateTask.mockClear();
    onBack.mockClear();
  });

  it("should not render the form while the device has not been retrieved", async () => {
    await renderWithConsole(
      <Errors.SuspenseBoundary>
        <Device.Configure
          deviceKey={id.create()}
          close={vi.fn()}
          icon={<Icon.Hardware />}
          initialProperties={{}}
        />
      </Errors.SuspenseBoundary>,
      { onCaughtError: () => {} },
    );
    expect(screen.queryByText(HINT)).toBeNull();
  });

  it("should leave the device unconfigured when the name is cleared", async () => {
    const { dev } = await setup();
    fireEvent.change(nameInput(), { target: { value: "" } });
    fireEvent.change(identifierInput(), { target: { value: "sensor_1" } });
    await save();
    expect(screen.getByText(HINT)).toBeTruthy();
    const unchanged = await client.devices.retrieve(dev.key);
    expect(unchanged.configured).toBe(false);
  });

  it("should leave the device unconfigured when the identifier is invalid", async () => {
    const { dev } = await setup();
    fireEvent.change(identifierInput(), { target: { value: "1x" } });
    await save();
    expect(screen.getByText(HINT)).toBeTruthy();
    const unchanged = await client.devices.retrieve(dev.key);
    expect(unchanged.configured).toBe(false);
  });

  it("should configure the device with the edited name, identifier, and initial properties, then close", async () => {
    const { dev } = await setup();
    const newName = uniqueName("renamed_sensor");
    fireEvent.change(nameInput(), { target: { value: newName } });
    fireEvent.change(identifierInput(), { target: { value: "sensor_1" } });
    await save();
    await waitFor(async () => {
      const updated = await client.devices.retrieve(dev.key);
      expect(updated.configured).toBe(true);
      expect(updated.name).toEqual(newName);
      expect(updated.properties.identifier).toEqual("sensor_1");
      expect(updated.properties.vendor).toEqual("acme");
    });
    await waitFor(() => expect(screen.queryByText(HINT)).toBeNull());
  });

  it("should fill the identifier from a suggestion based on the name", async () => {
    const { dev } = await setup();
    const [suggestion] = strings.createShortIdentifiers(dev.name);
    fireEvent.click(findButton(suggestion));
    expect(identifierInput().value).toEqual(suggestion);
  });

  it("should report the make of the device it configured", async () => {
    const analytics = createTestSink();
    await setup(Device.useConfigureModal, analytics);
    fireEvent.change(identifierInput(), { target: { value: "sensor_1" } });
    await save();
    await waitFor(() =>
      expect(analytics.capture).toHaveBeenCalledWith("device_configured", {
        make: "acme",
      }),
    );
  });

  describe("actions", () => {
    it("should not offer a read task or Back without their handlers", async () => {
      await setup();
      expect(findButton("Save")).toBeTruthy();
      expect(screen.queryByText("Save and create read task")).toBeNull();
      expect(screen.queryByText("Back")).toBeNull();
    });

    it("should create a read task for the device only after it saves", async () => {
      const { dev } = await setup(useConfigureWithActionsModal);
      fireEvent.change(identifierInput(), { target: { value: "sensor_1" } });
      await save("Save and create read task");
      await waitFor(() => expect(onCreateTask).toHaveBeenCalledWith(dev.key));
      const updated = await client.devices.retrieve(dev.key);
      expect(updated.configured).toBe(true);
    });

    it("should not create a read task when the save is rejected", async () => {
      await setup(useConfigureWithActionsModal);
      fireEvent.change(identifierInput(), { target: { value: "1x" } });
      await save("Save and create read task");
      expect(screen.getByText(HINT)).toBeTruthy();
      expect(onCreateTask).not.toHaveBeenCalled();
    });

    it("should go back without saving", async () => {
      const { dev } = await setup(useConfigureWithActionsModal);
      fireEvent.change(identifierInput(), { target: { value: "sensor_1" } });
      fireEvent.click(findButton("Back"));
      expect(onBack).toHaveBeenCalledOnce();
      const unchanged = await client.devices.retrieve(dev.key);
      expect(unchanged.configured).toBe(false);
    });
  });
});
