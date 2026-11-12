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

import { Sift } from "@/feature/sift";
import { createAPIKey, createSiftDevice } from "@/feature/sift/testutil";
import { createTestSink } from "@/platform/analytics/testutil";
import { createRunningRack } from "@/platform/device/testutil";
import { pressSaveTrigger, renderModalOpener } from "@/platform/modals/testutil";
import { answerNextCommand } from "@/platform/task/testutil";
import { findDialogTriggerByText, uniqueName } from "@/testutil";

const client = createTestClient();

const createScanTask = async () => {
  const rack = await createRunningRack(client, "sift");
  const scan = await rack.createTask({
    name: uniqueName("sift_scan"),
    type: Sift.Task.SCAN_TYPE,
    config: {},
  });
  return { rack, scan };
};

describe("Sift.Device.useConnectModal", () => {
  it("should populate the form from an existing device", async () => {
    const apiKey = await createAPIKey(client);
    const dev = await createSiftDevice(client, {
      properties: { uri: "sift.example.com:443", apiKey: apiKey.key },
    });
    await renderModalOpener(Sift.Device.useConnectModal, [{ deviceKey: dev.key }], {
      client,
    });
    await screen.findByDisplayValue(dev.name);
    expect(screen.getByDisplayValue("sift.example.com:443")).toBeTruthy();
    expect(await screen.findByText(apiKey.name)).toBeTruthy();
  });

  it("should start with an empty, focused name", async () => {
    await renderModalOpener(Sift.Device.useConnectModal, [{}], { client });
    const name = await screen.findByPlaceholderText("Sift");
    expect(name).toHaveProperty("value", "");
    expect(document.activeElement).toBe(name);
  });

  it("should require a URI and an API key", async () => {
    await createRunningRack(client, "sift");
    await renderModalOpener(Sift.Device.useConnectModal, [{}], { client });
    await screen.findByRole("dialog");
    pressSaveTrigger();
    expect(await screen.findByText("URI is required")).toBeTruthy();
    expect(screen.getByText("API key is required")).toBeTruthy();
  });

  it("should test the connection with a grant for the API key", async () => {
    const { rack, scan } = await createScanTask();
    const apiKey = await createAPIKey(client);
    const dev = await createSiftDevice(client, {
      rack: rack.key,
      properties: { apiKey: apiKey.key },
    });
    const analytics = createTestSink();
    await renderModalOpener(Sift.Device.useConnectModal, [{ deviceKey: dev.key }], {
      client,
      analytics,
    });
    await screen.findByDisplayValue(dev.name);
    const { answered } = await answerNextCommand(client, scan);
    pressSaveTrigger();
    const cmd = await answered;
    expect(cmd.type).toBe(Sift.Task.TEST_CONNECTION_COMMAND_TYPE);
    expect(cmd.args).toMatchObject({
      connection: { uri: "localhost:50051", apiKey: apiKey.key },
      grant: expect.any(String),
    });
    await waitFor(async () =>
      expect((await client.devices.retrieve({ key: dev.key })).location).toBe(
        "localhost:50051",
      ),
    );
    expect(analytics.capture).not.toHaveBeenCalled();
  });

  it("should report a device it connected", async () => {
    const { rack, scan } = await createScanTask();
    const apiKey = await createAPIKey(client);
    const analytics = createTestSink();
    await renderModalOpener(Sift.Device.useConnectModal, [{ rack: rack.key }], {
      client,
      analytics,
    });
    fireEvent.change(await screen.findByPlaceholderText("Sift"), {
      target: { value: uniqueName("sift") },
    });
    fireEvent.change(screen.getByPlaceholderText("grpc-api.siftstack.com:443"), {
      target: { value: "localhost:50051" },
    });
    fireEvent.click(await findDialogTriggerByText("Select a secret"));
    // The list pages, so a search finds the key however many secrets the Core holds.
    fireEvent.change(await screen.findByPlaceholderText("Search secrets..."), {
      target: { value: apiKey.name },
    });
    fireEvent.click(
      await screen.findByRole("option", { name: new RegExp(apiKey.name) }),
    );
    const { answered } = await answerNextCommand(client, scan);
    pressSaveTrigger();
    await answered;
    await waitFor(() =>
      expect(analytics.capture).toHaveBeenCalledWith("device_connected", {
        integration: "sift",
      }),
    );
  });

  it("should show why the connection test failed", async () => {
    const { rack, scan } = await createScanTask();
    const dev = await createSiftDevice(client, { rack: rack.key });
    await renderModalOpener(Sift.Device.useConnectModal, [{ deviceKey: dev.key }], {
      client,
    });
    await screen.findByDisplayValue(dev.name);
    const { answered } = await answerNextCommand(client, scan, {
      variant: "error",
      message: "Sift rejected the API key",
      description: "Check the API key and try again.",
    });
    pressSaveTrigger();
    await answered;
    expect(
      await screen.findByText(
        "Sift rejected the API key: Check the API key and try again.",
      ),
    ).toBeTruthy();
  });

  it("should name a new API key after the device", async () => {
    await renderModalOpener(Sift.Device.useConnectModal, [{}], { client });
    fireEvent.change(await screen.findByPlaceholderText("Sift"), {
      target: { value: "Test stand" },
    });
    fireEvent.click(await screen.findByText("Select a secret"));
    fireEvent.click(await screen.findByText("New secret"));
    expect(await screen.findByDisplayValue("Test stand API key")).toBeTruthy();
  });
});
