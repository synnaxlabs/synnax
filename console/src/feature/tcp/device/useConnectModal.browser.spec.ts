// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { uniqueName } from "@synnaxlabs/x/testutil";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { TCP } from "@/feature/tcp";
import { createTestSink } from "@/platform/analytics/testutil";
import { createRunningRack, createTestDevice } from "@/platform/device/testutil";
import { pressSaveTrigger, renderModalOpener } from "@/platform/modals/testutil";
import { commitFieldInput } from "@/platform/task/testutil";

const client = createTestClient();

describe("TCP.Device.useConnectModal", () => {
  it("should populate the form from an existing device", async () => {
    const dev = await createTestDevice(client, {
      make: TCP.Device.MAKE,
      model: TCP.Device.MAKE,
      properties: { host: "10.0.0.7", port: 5030 },
    });
    await renderModalOpener(TCP.Device.useConnectModal, [{ deviceKey: dev.key }], {
      client,
    });
    await screen.findByDisplayValue(dev.name);
    expect(screen.getByDisplayValue("10.0.0.7")).toBeTruthy();
    expect(screen.getByDisplayValue("5030")).toBeTruthy();
  });

  it("should save the trimmed server of a new device and report it", async () => {
    const rack = await createRunningRack(client, TCP.Device.INTEGRATION_KEY);
    const analytics = createTestSink();
    const { result } = await renderModalOpener(
      TCP.Device.useConnectModal,
      [{ rack: rack.key }],
      { client, analytics },
    );
    const name = uniqueName("tcp");
    fireEvent.change(await screen.findByPlaceholderText("Device name"), {
      target: { value: name },
    });
    fireEvent.change(screen.getByPlaceholderText("192.168.1.20"), {
      target: { value: " fe80::1\n" },
    });
    commitFieldInput(screen.getByDisplayValue<HTMLInputElement>("0"), "5030");
    pressSaveTrigger();
    const key = await result();
    if (key == null) throw new Error("the modal closed without a device");
    expect(await client.devices.retrieve({ key })).toMatchObject({
      name,
      make: "tcp",
      model: "tcp",
      rack: rack.key,
      location: "[fe80::1]:5030",
      properties: { host: "fe80::1", port: 5030 },
    });
    await waitFor(() =>
      expect(analytics.capture).toHaveBeenCalledWith("device_connected", {
        integration: "tcp",
      }),
    );
  });

  it("should show the schema error of a blank host at its input", async () => {
    const rack = await createRunningRack(client, TCP.Device.INTEGRATION_KEY);
    await renderModalOpener(TCP.Device.useConnectModal, [{ rack: rack.key }], {
      client,
    });
    fireEvent.change(await screen.findByPlaceholderText("Device name"), {
      target: { value: uniqueName("tcp") },
    });
    fireEvent.change(screen.getByPlaceholderText("192.168.1.20"), {
      target: { value: "   " },
    });
    pressSaveTrigger();
    expect(
      await screen.findByText(/expected string to have >=1 characters/),
    ).toBeTruthy();
  });

  it("should put the host and the port in one row", async () => {
    await renderModalOpener(TCP.Device.useConnectModal, [{}], { client });
    const host = (
      await screen.findByPlaceholderText("192.168.1.20")
    ).getBoundingClientRect();
    const port = screen.getByDisplayValue("0").getBoundingClientRect();
    expect(host.top < port.bottom && port.top < host.bottom).toBe(true);
  });

  it("should mark the host as required and not the port, which has a value", async () => {
    await renderModalOpener(TCP.Device.useConnectModal, [{}], { client });
    const mark = (input: HTMLElement) =>
      input.closest(".lyra-input__item")?.querySelector(".lyra-required-indicator");
    expect(mark(await screen.findByPlaceholderText("192.168.1.20"))).toBeTruthy();
    expect(mark(screen.getByDisplayValue("0"))).toBeFalsy();
  });
});
