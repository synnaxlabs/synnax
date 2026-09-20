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
import { pressSaveTrigger, renderModalOpener } from "@/platform/modals/testutil";
import { getSwitchInput } from "@/testutil";

const client = createTestClient();

const renderConnectModal = async () => {
  const handle = await renderModalOpener(HTTP.Device.useConnectModal, [{}], {
    client,
  });
  await screen.findByText("Server");
  return handle;
};

const clickAuthButton = (name: string): void => {
  fireEvent.click(screen.getByRole("button", { name }));
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
    expect(await screen.findByDisplayValue("/health")).toBeTruthy();
    expect(getSwitchInput("Validate response body").checked).toBe(false);
  });

  it("should default max concurrent requests to 6 for a new device", async () => {
    await renderConnectModal();
    expect(screen.getByText("Max concurrent requests")).toBeTruthy();
    expect(await screen.findByDisplayValue("6")).toBeTruthy();
  });

  it("should populate max concurrent requests from an existing device", async () => {
    const dev = await createHTTPDevice(client, {
      properties: { maxConcurrentRequests: 3 },
    });
    await renderModalOpener(HTTP.Device.useConnectModal, [{ deviceKey: dev.key }], {
      client,
    });
    await screen.findByDisplayValue(dev.name);
    expect(await screen.findByDisplayValue("3")).toBeTruthy();
  });

  it("should clamp max concurrent requests to at least 1", async () => {
    const dev = await createHTTPDevice(client, {
      properties: { maxConcurrentRequests: 3 },
    });
    await renderModalOpener(HTTP.Device.useConnectModal, [{ deviceKey: dev.key }], {
      client,
    });
    const input = await screen.findByDisplayValue("3");
    fireEvent.change(input, { target: { value: "0" } });
    fireEvent.blur(input);
    await waitFor(() => expect((input as HTMLInputElement).value).toBe("1"));
  });

  it("should reveal the token field for bearer auth", async () => {
    await renderConnectModal();
    clickAuthButton("Bearer token");
    await screen.findByPlaceholderText(/eyJhbGciOi/);
    clickAuthButton("None");
    await waitFor(() => expect(screen.queryByPlaceholderText(/eyJhbGciOi/)).toBeNull());
  });

  it("should switch API key auth between header and query parameter delivery", async () => {
    await renderConnectModal();
    clickAuthButton("API key");
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
    clickAuthButton("Basic");
    await screen.findByPlaceholderText("user@example.com");
    expect(screen.getByText("Password")).toBeTruthy();
  });

  it("should reveal the body field for a POST health check", async () => {
    await renderConnectModal();
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
    fireEvent.click(getSwitchInput("Validate response body"));
    await screen.findByPlaceholderText("/status");
    await screen.findByPlaceholderText("ok");
    fireEvent.click(screen.getByRole("button", { name: "Number" }));
    await waitFor(() => expect(screen.queryByPlaceholderText("ok")).toBeNull());
    fireEvent.click(screen.getByRole("button", { name: "Boolean" }));
    await waitFor(() => expect(getSwitchInput("Expected value")).toBeTruthy());
    fireEvent.click(getSwitchInput("Validate response body"));
    await waitFor(() => expect(screen.queryByPlaceholderText("/status")).toBeNull());
  });

  // Submitting with no rack chosen fails validation. That error is the proof the keys
  // reached the same save path the Connect button uses.
  it("should submit on the shortcut its footer advertises", async () => {
    await renderModalOpener(HTTP.Device.useConnectModal, [{}], { client });
    await screen.findByRole("dialog");
    pressSaveTrigger();
    expect(await screen.findByText(/rack is required/i)).toBeTruthy();
  });
});
