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
import { createBroker } from "@/feature/mqtt/testutil";
import { pressSaveTrigger, renderModalOpener } from "@/platform/modals/testutil";
import { getSwitchInput } from "@/testutil";

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
    expect(screen.getByPlaceholderText("1883")).toBeTruthy();
    expect(screen.getByPlaceholderText("30")).toBeTruthy();
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

  it("should clamp the port to 65535", async () => {
    await renderConnectModal();
    const input = screen.getByPlaceholderText<HTMLInputElement>("1883");
    fireEvent.change(input, { target: { value: "70000" } });
    fireEvent.blur(input);
    await waitFor(() => expect(input.value).toBe("65535"));
  });

  // Submitting with no rack chosen fails validation. That error is the proof the keys
  // reached the same save path the Connect button uses.
  it("should submit on the shortcut its footer advertises", async () => {
    await renderModalOpener(MQTT.Device.useConnectModal, [{}], { client });
    await screen.findByRole("dialog");
    pressSaveTrigger();
    expect(await screen.findByText(/rack is required/i)).toBeTruthy();
  });

  it("should surface a missing scan task when the rack has no MQTT driver", async () => {
    const dev = await createBroker(client);
    await renderModalOpener(MQTT.Device.useConnectModal, [{ deviceKey: dev.key }], {
      client,
    });
    await screen.findByDisplayValue(dev.name);
    fireEvent.click(screen.getByRole("button", { name: /Connect/ }));
    expect(await screen.findByText(/not found/i)).toBeTruthy();
  });
});
