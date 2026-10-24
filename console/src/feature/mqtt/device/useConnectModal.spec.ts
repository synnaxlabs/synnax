// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { MQTT } from "@/feature/mqtt";
import { createBroker, openScanner } from "@/feature/mqtt/testutil";
import { createTestSink } from "@/platform/analytics/testutil";
import { pressSaveTrigger, renderModalOpener } from "@/platform/modals/testutil";
import { selectFromDropdown } from "@/platform/task/testutil";
import { getSwitchInput, uniqueName } from "@/testutil";

const client = createTestClient();

const renderConnectModal = async () => {
  const handle = await renderModalOpener(MQTT.Device.useConnectModal, [{}], {
    client,
  });
  await screen.findByText("Broker");
  return handle;
};

describe("useConnectModal", () => {
  it("should populate the form from an existing device", async () => {
    const dev = await createBroker(client, {
      properties: { username: "plant_user", clientId: "console_client", port: 1884 },
    });
    await renderModalOpener(MQTT.Device.useConnectModal, [{ deviceKey: dev.key }], {
      client,
    });
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
    expect(screen.getByText("Skip certificate verification")).toBeTruthy();
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
    await renderModalOpener(MQTT.Device.useConnectModal, [{ deviceKey: dev.key }], {
      client,
    });
    expect(await screen.findByDisplayValue("/etc/ssl/plant_ca.pem")).toBeTruthy();
    expect(getSwitchInput("TLS").checked).toBe(true);
  });

  it("should populate the Sparkplug B host ID and groups of a device", async () => {
    const dev = await createBroker(client, {
      properties: {
        sparkplug: { hostId: "synnax_core", groups: ["plant", "utilities"] },
      },
    });
    await renderModalOpener(MQTT.Device.useConnectModal, [{ deviceKey: dev.key }], {
      client,
    });
    expect(await screen.findByDisplayValue("synnax_core")).toBeTruthy();
    expect(screen.getByDisplayValue("plant, utilities")).toBeTruthy();
  });

  it("should reject a host ID that holds a topic character", async () => {
    await renderConnectModal();
    const input = screen.getByPlaceholderText("Passive host");
    fireEvent.change(input, { target: { value: "core/1" } });
    fireEvent.click(screen.getByRole("button", { name: /Connect/ }));
    expect(await screen.findByText("Host ID must not hold /, +, or #")).toBeTruthy();
  });

  it("should split the groups at the commas and reject a bad group", async () => {
    await renderConnectModal();
    const input = screen.getByPlaceholderText<HTMLInputElement>("All groups");
    fireEvent.focus(input);
    fireEvent.change(input, { target: { value: " plant ,, line#1" } });
    fireEvent.blur(input);
    await waitFor(() => expect(input.value).toBe("plant, line#1"));
    fireEvent.click(screen.getByRole("button", { name: /Connect/ }));
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
    fireEvent.click(screen.getByRole("button", { name: /Connect/ }));
    expect(await screen.findByText("Port must be 1 to 65535")).toBeTruthy();
    expect(screen.getByText("Keep alive must be positive")).toBeTruthy();
  });

  it("should report a device it connected and leave an empty port unset", async () => {
    const rack = await client.racks.create({
      name: uniqueName("mqtt_rack"),
      integrations: [MQTT.Device.MAKE],
    });
    const scanner = await openScanner(client, rack.key);
    const analytics = createTestSink();
    await renderModalOpener(MQTT.Device.useConnectModal, [{}], { client, analytics });
    await selectFromDropdown("Select Driver", rack.name);
    fireEvent.change(screen.getByPlaceholderText("broker.example.com"), {
      target: { value: "localhost" },
    });
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
    const rack = await client.racks.create({
      name: uniqueName("mqtt_rack"),
      integrations: [MQTT.Device.MAKE],
    });
    const scanner = await openScanner(client, rack.key);
    await renderModalOpener(MQTT.Device.useConnectModal, [{}], { client });
    await selectFromDropdown("Select Driver", rack.name);
    fireEvent.change(screen.getByPlaceholderText("broker.example.com"), {
      target: { value: "localhost" },
    });
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
    const dev = await createBroker(client);
    const scanner = await openScanner(client, dev.rack);
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

  // Submitting with no rack chosen fails validation. That error is the proof the keys
  // reached the same save path the Connect button uses.
  it("should submit on the shortcut its footer advertises", async () => {
    await renderModalOpener(MQTT.Device.useConnectModal, [{}], { client });
    await screen.findByRole("dialog");
    pressSaveTrigger();
    expect(await screen.findByText(/rack is required/i)).toBeTruthy();
  });

  it("should surface a missing scan task on the rack of the broker", async () => {
    const dev = await createBroker(client);
    await renderModalOpener(MQTT.Device.useConnectModal, [{ deviceKey: dev.key }], {
      client,
    });
    await screen.findByDisplayValue(dev.name);
    fireEvent.click(screen.getByRole("button", { name: /Connect/ }));
    expect(await screen.findByText(/not found/i)).toBeTruthy();
  });
});
