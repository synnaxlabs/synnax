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

import { HTTP } from "@/feature/http";
import { createHTTPDevice } from "@/feature/http/testutil";
import { createTestSink } from "@/platform/analytics/testutil";
import { createRunningRack } from "@/platform/device/testutil";
import { pressSaveTrigger, renderModalOpener } from "@/platform/modals/testutil";
import { answerNextCommand } from "@/platform/task/testutil";
import { getSwitchInput, uniqueName } from "@/testutil";

const client = createTestClient();

const renderConnectModal = async () => {
  const handle = await renderModalOpener(HTTP.Device.useConnectModal, [{}], {
    client,
  });
  await screen.findByText("HTTP server");
  return handle;
};

const clickButton = (name: string): void => {
  fireEvent.click(screen.getByRole("button", { name }));
};

const clickTab = (name: string): void => {
  fireEvent.click(screen.getByRole("tab", { name }));
};

describe("useConnectModal", () => {
  it("should populate the form from an existing device", async () => {
    const dev = await createHTTPDevice(client, {
      properties: { auth: { type: "bearer", token: "tok_existing_123" } },
    });
    await renderModalOpener(HTTP.Device.useConnectModal, [{ deviceKey: dev.key }], {
      client,
    });
    await screen.findByDisplayValue(dev.name);
    expect(screen.getByDisplayValue(dev.location)).toBeTruthy();
    await screen.findByDisplayValue("tok_existing_123");
  });

  it("should open a device saved without a validateResponse flag", async () => {
    const dev = await createHTTPDevice(client, {
      properties: {
        healthCheck: { method: "GET", path: "/health" } as HTTP.Device.HealthCheck,
      },
    });
    await renderModalOpener(HTTP.Device.useConnectModal, [{ deviceKey: dev.key }], {
      client,
    });
    await screen.findByDisplayValue(dev.name);
    clickTab("Health check");
    expect(await screen.findByDisplayValue("/health")).toBeTruthy();
    expect(getSwitchInput("Validate body").checked).toBe(false);
  });

  it("should default concurrent requests to 6 for a new device", async () => {
    await renderConnectModal();
    expect(screen.queryByText("Concurrent requests")).toBeNull();
    clickTab("Advanced");
    expect(screen.getByText("Concurrent requests")).toBeTruthy();
    expect(await screen.findByDisplayValue("6")).toBeTruthy();
  });

  it("should populate concurrent requests from an existing device", async () => {
    const dev = await createHTTPDevice(client, {
      properties: { maxConcurrentRequests: 3 },
    });
    await renderModalOpener(HTTP.Device.useConnectModal, [{ deviceKey: dev.key }], {
      client,
    });
    await screen.findByDisplayValue(dev.name);
    clickTab("Advanced");
    expect(await screen.findByDisplayValue("3")).toBeTruthy();
  });

  it("should clamp concurrent requests to at least 1", async () => {
    const dev = await createHTTPDevice(client, {
      properties: { maxConcurrentRequests: 3 },
    });
    await renderModalOpener(HTTP.Device.useConnectModal, [{ deviceKey: dev.key }], {
      client,
    });
    await screen.findByDisplayValue(dev.name);
    clickTab("Advanced");
    const input = await screen.findByDisplayValue("3");
    fireEvent.change(input, { target: { value: "0" } });
    fireEvent.blur(input);
    await waitFor(() => expect((input as HTMLInputElement).value).toBe("1"));
  });

  it("should hide certificate verification while HTTPS is off", async () => {
    await renderConnectModal();
    expect(getSwitchInput("Verify certificate").checked).toBe(true);
    fireEvent.click(getSwitchInput("HTTPS"));
    await waitFor(() => expect(screen.queryByText("Verify certificate")).toBeNull());
  });

  it("should reveal the token field for bearer auth", async () => {
    await renderConnectModal();
    clickButton("Bearer token");
    await screen.findByPlaceholderText(/eyJhbGciOi/);
    clickButton("None");
    await waitFor(() => expect(screen.queryByPlaceholderText(/eyJhbGciOi/)).toBeNull());
  });

  it("should switch API key auth between header and query parameter delivery", async () => {
    await renderConnectModal();
    clickButton("API key");
    await screen.findByPlaceholderText("X-API-Key");
    expect(screen.getByPlaceholderText(/sk_live/)).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Query parameter" }));
    await screen.findByPlaceholderText("key");
    expect(screen.queryByPlaceholderText("X-API-Key")).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Header" }));
    await screen.findByPlaceholderText("X-API-Key");
  });

  it("should reveal username and password fields for basic auth", async () => {
    await renderConnectModal();
    clickButton("Basic");
    await screen.findByPlaceholderText("user@example.com");
    expect(screen.getByText("Password")).toBeTruthy();
  });

  it("should reveal the body field for a POST health check", async () => {
    await renderConnectModal();
    clickTab("Health check");
    expect(screen.queryByPlaceholderText('{"check": "ping"}')).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "POST" }));
    await screen.findByPlaceholderText('{"check": "ping"}');
    fireEvent.click(screen.getByRole("button", { name: "GET" }));
    await waitFor(() =>
      expect(screen.queryByPlaceholderText('{"check": "ping"}')).toBeNull(),
    );
  });

  it("should reveal typed expected-value inputs when response validation is enabled", async () => {
    await renderConnectModal();
    clickTab("Health check");
    fireEvent.click(getSwitchInput("Validate body"));
    await screen.findByPlaceholderText("/status");
    await screen.findByPlaceholderText("ok");
    fireEvent.click(screen.getByRole("button", { name: "Number" }));
    await waitFor(() => expect(screen.queryByPlaceholderText("ok")).toBeNull());
    fireEvent.click(screen.getByRole("button", { name: "Boolean" }));
    await waitFor(() => expect(getSwitchInput("Expected value")).toBeTruthy());
    fireEvent.click(getSwitchInput("Validate body"));
    await waitFor(() => expect(screen.queryByPlaceholderText("/status")).toBeNull());
  });

  it("should open the tab with the field that failed validation", async () => {
    const rack = await createRunningRack(client, "http");
    await renderModalOpener(HTTP.Device.useConnectModal, [{ rack: rack.key }], {
      client,
    });
    fireEvent.change(screen.getByPlaceholderText("www.example.com"), {
      target: { value: "localhost:8080" },
    });
    clickButton("API key");
    clickTab("Health check");
    pressSaveTrigger();
    expect(await screen.findByText("Key is required")).toBeTruthy();
    expect(screen.getByText("Header is required")).toBeTruthy();
  });

  it("should resolve with the device it connected", async () => {
    const rack = await createRunningRack(client, "http");
    const scan = await rack.createTask({
      name: uniqueName("http_scan"),
      type: HTTP.Device.SCAN_TYPE,
      config: {},
    });
    const { result } = await renderModalOpener(
      HTTP.Device.useConnectModal,
      [{ rack: rack.key }],
      { client },
    );
    fireEvent.change(screen.getByPlaceholderText("www.example.com"), {
      target: { value: "localhost:8080" },
    });
    const { answered } = await answerNextCommand(client, scan);
    pressSaveTrigger();
    await answered;
    const key = await result();
    if (key == null) throw new Error("the modal closed without a device");
    expect((await client.devices.retrieve({ key })).rack).toBe(rack.key);
  });

  it("should report a device it connected", async () => {
    const rack = await createRunningRack(client, "http");
    const scan = await rack.createTask({
      name: uniqueName("http_scan"),
      type: HTTP.Device.SCAN_TYPE,
      config: {},
    });
    const analytics = createTestSink();
    await renderModalOpener(HTTP.Device.useConnectModal, [{ rack: rack.key }], {
      client,
      analytics,
    });
    fireEvent.change(screen.getByPlaceholderText("www.example.com"), {
      target: { value: "localhost:8080" },
    });
    const { answered } = await answerNextCommand(client, scan);
    pressSaveTrigger();
    await answered;
    await waitFor(() =>
      expect(analytics.capture).toHaveBeenCalledWith("device_connected", {
        integration: "http",
      }),
    );
  });

  it("should not report an edited device as connected", async () => {
    const rack = await createRunningRack(client, "http");
    const dev = await createHTTPDevice(client, { rack: rack.key });
    const scan = await rack.createTask({
      name: uniqueName("http_scan"),
      type: HTTP.Device.SCAN_TYPE,
      config: {},
    });
    const analytics = createTestSink();
    await renderModalOpener(HTTP.Device.useConnectModal, [{ deviceKey: dev.key }], {
      client,
      analytics,
    });
    await screen.findByDisplayValue(dev.name);
    const { answered } = await answerNextCommand(client, scan);
    pressSaveTrigger();
    await answered;
    await waitFor(() => expect(screen.queryByDisplayValue(dev.name)).toBeNull());
    expect(analytics.capture).not.toHaveBeenCalled();
  });

  it("should show a failure from a Driver that sends no description", async () => {
    const rack = await createRunningRack(client, "http");
    const scan = await rack.createTask({
      name: uniqueName("http_scan"),
      type: HTTP.Device.SCAN_TYPE,
      config: {},
    });
    await renderModalOpener(HTTP.Device.useConnectModal, [{ rack: rack.key }], {
      client,
    });
    fireEvent.change(screen.getByPlaceholderText("www.example.com"), {
      target: { value: "localhost:8080" },
    });
    const { answered } = await answerNextCommand(client, scan, {
      variant: "error",
      message: "Connection refused",
    });
    pressSaveTrigger();
    await answered;
    expect(await screen.findByText("Connection refused")).toBeTruthy();
  });

  // A running rack keeps Connect enabled while none is chosen, so the save fails
  // validation. That error proves the keys reached the Connect button's save path.
  it("should submit on the shortcut its footer advertises", async () => {
    await createRunningRack(client, "http");
    await renderModalOpener(HTTP.Device.useConnectModal, [{}], { client });
    await screen.findByRole("dialog");
    pressSaveTrigger();
    expect(await screen.findByText(/rack is required/i)).toBeTruthy();
  });
});
