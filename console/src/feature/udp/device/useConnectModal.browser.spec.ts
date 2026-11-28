// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { device } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { uniqueName } from "@synnaxlabs/x/testutil";
import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { UDP } from "@/feature/udp";
import { createRunningRack, createTestDevice } from "@/platform/device/testutil";
import { pressSaveTrigger, renderModalOpener } from "@/platform/modals/testutil";
import { commitFieldInput } from "@/platform/task/testutil";
import { getSwitchInput } from "@/testutil";

const client = createTestClient();

const REMOTE_CHECKBOX = "Remote address";

const createUDPDevice = async (rack?: device.Device["rack"]) =>
  await createTestDevice(client, {
    rack,
    make: UDP.Device.MAKE,
    model: UDP.Device.MAKE,
    properties: {
      localAddress: "127.0.0.1",
      localPort: 5032,
      remote: { host: "10.0.0.7", port: 5031 },
    },
  });

const saveAndRetrieve = async (
  result: () => Promise<device.Key | null> | undefined,
) => {
  pressSaveTrigger();
  const key = await result();
  if (key == null) throw new Error("the modal closed without a device");
  return await client.devices.retrieve({ key });
};

describe("UDP.Device.useConnectModal", () => {
  it("should populate the form from an existing device", async () => {
    const dev = await createUDPDevice();
    await renderModalOpener(UDP.Device.useConnectModal, [{ deviceKey: dev.key }], {
      client,
    });
    await screen.findByDisplayValue(dev.name);
    expect(screen.getByDisplayValue("127.0.0.1")).toBeTruthy();
    expect(screen.getByDisplayValue("5032")).toBeTruthy();
    expect(getSwitchInput(REMOTE_CHECKBOX).checked).toBe(true);
    expect(screen.getByDisplayValue("10.0.0.7")).toBeTruthy();
    expect(screen.getByDisplayValue("5031")).toBeTruthy();
  });

  it("should save a device with no remote at its trimmed local address", async () => {
    const rack = await createRunningRack(client, UDP.Device.INTEGRATION_KEY);
    const { result } = await renderModalOpener(
      UDP.Device.useConnectModal,
      [{ rack: rack.key }],
      { client },
    );
    fireEvent.change(await screen.findByPlaceholderText("Device name"), {
      target: { value: uniqueName("udp") },
    });
    fireEvent.change(screen.getByPlaceholderText("All interfaces"), {
      target: { value: " 127.0.0.1\n" },
    });
    commitFieldInput(screen.getByDisplayValue<HTMLInputElement>("0"), "5032");
    const dev = await saveAndRetrieve(result);
    expect(dev).toMatchObject({
      make: "udp",
      model: "udp",
      location: "127.0.0.1:5032",
    });
    expect(dev.properties).toEqual({ localAddress: "127.0.0.1", localPort: 5032 });
  });

  it("should save the trimmed remote that the checkbox turns on as the location", async () => {
    const rack = await createRunningRack(client, UDP.Device.INTEGRATION_KEY);
    const { result } = await renderModalOpener(
      UDP.Device.useConnectModal,
      [{ rack: rack.key }],
      { client },
    );
    fireEvent.change(await screen.findByPlaceholderText("Device name"), {
      target: { value: uniqueName("udp") },
    });
    fireEvent.click(getSwitchInput(REMOTE_CHECKBOX));
    fireEvent.change(await screen.findByPlaceholderText("192.168.1.20"), {
      target: { value: " 10.0.0.7\n" },
    });
    commitFieldInput(screen.getAllByDisplayValue<HTMLInputElement>("0")[1], "5031");
    const dev = await saveAndRetrieve(result);
    expect(dev.location).toBe("10.0.0.7:5031");
    expect(dev.properties).toEqual({
      localAddress: "",
      localPort: 0,
      remote: { host: "10.0.0.7", port: 5031 },
    });
  });

  it("should remove the remote that the checkbox turns off", async () => {
    const rack = await createRunningRack(client, UDP.Device.INTEGRATION_KEY);
    const dev = await createUDPDevice(rack.key);
    const { result } = await renderModalOpener(
      UDP.Device.useConnectModal,
      [{ deviceKey: dev.key }],
      { client },
    );
    await screen.findByDisplayValue(dev.name);
    fireEvent.click(getSwitchInput(REMOTE_CHECKBOX));
    expect(screen.queryByDisplayValue("10.0.0.7")).toBeNull();
    const saved = await saveAndRetrieve(result);
    expect(saved.location).toBe("127.0.0.1:5032");
    expect(saved.properties).toEqual({ localAddress: "127.0.0.1", localPort: 5032 });
  });

  it("should put each address with its port in one row", async () => {
    const dev = await createUDPDevice();
    await renderModalOpener(UDP.Device.useConnectModal, [{ deviceKey: dev.key }], {
      client,
    });
    await screen.findByDisplayValue(dev.name);
    const box = (value: string) =>
      screen.getByDisplayValue(value).getBoundingClientRect();
    const sameRow = (a: DOMRect, b: DOMRect) => a.top < b.bottom && b.top < a.bottom;
    expect(sameRow(box("127.0.0.1"), box("5032"))).toBe(true);
    expect(sameRow(box("10.0.0.7"), box("5031"))).toBe(true);
    expect(box("10.0.0.7").top).toBeGreaterThanOrEqual(box("5032").bottom);
  });
});
