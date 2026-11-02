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
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { MQTT } from "@/feature/mqtt";
import { createBroker, openScanner } from "@/feature/mqtt/testutil";
import { createTestSink } from "@/platform/analytics/testutil";
import { createRunningRack } from "@/platform/device/testutil";
import { pressSaveTrigger, renderModalOpener } from "@/platform/modals/testutil";
import { getSwitchInput } from "@/testutil";

const client = createTestClient();

const createRack = async () => await createRunningRack(client, MQTT.Device.INTEGRATION);

const renderConnectModal = async () => {
  const rack = await createRack();
  const handle = await renderModalOpener(
    MQTT.Device.useConnectModal,
    [{ rack: rack.key }],
    { client },
  );
  await screen.findByPlaceholderText("broker.example.com");
  return { ...handle, rack };
};

const renderEditModal = async (deviceKey: device.Key) => {
  const handle = await renderModalOpener(MQTT.Device.useConnectModal, [{ deviceKey }], {
    client,
  });
  await screen.findByPlaceholderText("broker.example.com");
  return handle;
};

const fillHost = () =>
  fireEvent.change(screen.getByPlaceholderText("broker.example.com"), {
    target: { value: "localhost" },
  });

describe("MQTT.Device.useConnectModal", () => {
  it("should populate the form from an existing device", async () => {
    const dev = await createBroker(client, {
      properties: { username: "plant_user", clientId: "console_client", port: 1884 },
    });
    await renderEditModal(dev.key);
    await screen.findByDisplayValue(dev.name);
    expect(screen.getByDisplayValue(dev.location)).toBeTruthy();
    expect(await screen.findByDisplayValue("plant_user")).toBeTruthy();
    expect(screen.getByDisplayValue("console_client")).toBeTruthy();
    expect(screen.getByDisplayValue("1884")).toBeTruthy();
  });

  it("should show the default port and keep alive as placeholders", async () => {
    await renderConnectModal();
    expect(screen.getByPlaceholderText<HTMLInputElement>("1883").value).toBe("");
    expect(screen.getByPlaceholderText<HTMLInputElement>("30").value).toBe("");
  });

  it("should reveal the certificate fields and the TLS port when TLS is on", async () => {
    await renderConnectModal();
    expect(screen.queryByText("CA file")).toBeNull();
    fireEvent.click(getSwitchInput("TLS"));
    await screen.findByText("CA file");
    expect(screen.getByText("Skip verification")).toBeTruthy();
    expect(screen.getByText("Client certificate")).toBeTruthy();
    expect(screen.getByText("Client key")).toBeTruthy();
    expect(screen.getByText(/paths on the Core host/)).toBeTruthy();
    expect(screen.getByPlaceholderText("8883")).toBeTruthy();
    fireEvent.click(getSwitchInput("TLS"));
    await waitFor(() => expect(screen.queryByText("CA file")).toBeNull());
    expect(screen.getByPlaceholderText("1883")).toBeTruthy();
  });

  it("should open the certificate fields for a device saved with TLS", async () => {
    const dev = await createBroker(client, {
      properties: { secure: true, caFile: "/etc/ssl/plant_ca.pem" },
    });
    await renderEditModal(dev.key);
    expect(await screen.findByDisplayValue("/etc/ssl/plant_ca.pem")).toBeTruthy();
    expect(getSwitchInput("TLS").checked).toBe(true);
  });

  it("should populate the Sparkplug B host ID and groups of a device", async () => {
    const dev = await createBroker(client, {
      properties: {
        sparkplug: { hostId: "synnax_core", groups: ["plant", "utilities"] },
      },
    });
    await renderEditModal(dev.key);
    expect(await screen.findByDisplayValue("synnax_core")).toBeTruthy();
    expect(screen.getByDisplayValue("plant, utilities")).toBeTruthy();
  });

  it("should reject a host ID that holds a topic character", async () => {
    await renderConnectModal();
    const input = screen.getByPlaceholderText("Passive host");
    fireEvent.change(input, { target: { value: "core/1" } });
    pressSaveTrigger();
    expect(await screen.findByText("Host ID must not hold /, +, or #")).toBeTruthy();
  });

  it("should split the groups at the commas and reject a bad group", async () => {
    await renderConnectModal();
    const input = screen.getByPlaceholderText<HTMLInputElement>("All groups");
    fireEvent.focus(input);
    fireEvent.change(input, { target: { value: " plant ,, line#1" } });
    fireEvent.blur(input);
    await waitFor(() => expect(input.value).toBe("plant, line#1"));
    pressSaveTrigger();
    expect(
      await screen.findByText('Group "line#1" must not hold /, +, or #'),
    ).toBeTruthy();
  });

  it("should reject a port and keep alive out of range", async () => {
    await renderConnectModal();
    const port = screen.getByPlaceholderText<HTMLInputElement>("1883");
    fireEvent.change(port, { target: { value: "70000" } });
    fireEvent.blur(port);
    await waitFor(() => expect(port.value).toBe("70000"));
    const keepAlive = screen.getByPlaceholderText<HTMLInputElement>("30");
    fireEvent.change(keepAlive, { target: { value: "0" } });
    fireEvent.blur(keepAlive);
    await waitFor(() => expect(keepAlive.value).toBe("0"));
    pressSaveTrigger();
    expect(await screen.findByText("Port must be 1 to 65535")).toBeTruthy();
    expect(screen.getByText("Keep alive must be 1 to 65535")).toBeTruthy();
  });

  it("should resolve with the device it connected", async () => {
    const { result, rack } = await renderConnectModal();
    const scanner = await openScanner(client, rack.key);
    fillHost();
    pressSaveTrigger();
    await scanner.answer({});
    scanner.close();
    const key = await result();
    if (key == null) throw new Error("the modal closed without a device");
    expect((await client.devices.retrieve({ key })).rack).toBe(rack.key);
  });

  it("should report a device it connected and leave an empty port unset", async () => {
    const rack = await createRack();
    const scanner = await openScanner(client, rack.key);
    const analytics = createTestSink();
    await renderModalOpener(MQTT.Device.useConnectModal, [{ rack: rack.key }], {
      client,
      analytics,
    });
    fillHost();
    pressSaveTrigger();
    const cmd = await scanner.answer({});
    scanner.close();
    expect(cmd.args.properties).not.toHaveProperty("port");
    expect(cmd.args.properties).not.toHaveProperty("keep_alive");
    await waitFor(() =>
      expect(analytics.capture).toHaveBeenCalledWith("device_connected", {
        integration: "mqtt",
      }),
    );
  });

  it("should leave a port and keep alive the user typed and then cleared unset", async () => {
    const { rack } = await renderConnectModal();
    const scanner = await openScanner(client, rack.key);
    fillHost();
    for (const placeholder of ["1883", "30"]) {
      const input = screen.getByPlaceholderText(placeholder);
      fireEvent.change(input, { target: { value: "42" } });
      fireEvent.blur(input);
      await screen.findByDisplayValue("42");
      fireEvent.change(input, { target: { value: "" } });
      fireEvent.blur(input);
      await waitFor(() => expect(screen.queryByDisplayValue("42")).toBeNull());
    }
    pressSaveTrigger();
    const cmd = await scanner.answer({});
    scanner.close();
    expect(cmd.args.properties).not.toHaveProperty("port");
    expect(cmd.args.properties).not.toHaveProperty("keep_alive");
  });

  it("should not report an edited device as connected", async () => {
    const rack = await createRack();
    const dev = await createBroker(client, { rack: rack.key });
    const scanner = await openScanner(client, rack.key);
    const analytics = createTestSink();
    await renderModalOpener(MQTT.Device.useConnectModal, [{ deviceKey: dev.key }], {
      client,
      analytics,
    });
    await screen.findByDisplayValue(dev.name);
    pressSaveTrigger();
    await scanner.answer({});
    scanner.close();
    await waitFor(() => expect(screen.queryByDisplayValue(dev.name)).toBeNull());
    expect(analytics.capture).not.toHaveBeenCalled();
  });

  it("should show the error a Driver reports for the connection", async () => {
    const { rack } = await renderConnectModal();
    const scanner = await openScanner(client, rack.key);
    fillHost();
    pressSaveTrigger();
    await scanner.answer({ variant: "error", message: "Connection refused" });
    scanner.close();
    expect(await screen.findByText("Connection refused")).toBeTruthy();
  });

  // Two running racks leave the rack field empty, so the save fails validation. That
  // error proves the keys reached the Connect button's save path.
  it("should submit on the shortcut its footer advertises", async () => {
    await createRack();
    await createRack();
    await renderModalOpener(MQTT.Device.useConnectModal, [{}], { client });
    await screen.findByRole("dialog");
    pressSaveTrigger();
    expect(await screen.findByText(/rack is required/i)).toBeTruthy();
  });

  it("should surface a missing scan task on the rack of the broker", async () => {
    const rack = await createRack();
    const dev = await createBroker(client, { rack: rack.key });
    await renderEditModal(dev.key);
    await screen.findByDisplayValue(dev.name);
    pressSaveTrigger();
    expect(await screen.findByText(/not found/i)).toBeTruthy();
  });
});
