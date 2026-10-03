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

import { OPCUA } from "@/feature/opcua";
import { createOPCDevice } from "@/feature/opcua/testutil";
import { createTestSink } from "@/platform/analytics/testutil";
import { pressSaveTrigger, renderModalOpener } from "@/platform/modals/testutil";
import { answerNextCommand, selectFromDropdown } from "@/platform/task/testutil";
import { uniqueName } from "@/testutil";

const client = createTestClient();

describe("OPCUA.Device.useConnectModal", () => {
  it("should populate the form from an existing device", async () => {
    const dev = await createOPCDevice(client, {
      properties: {
        connection: {
          endpoint: "opc.tcp://existing-server:4840",
          username: "operator",
          password: "secret",
          securityMode: "Sign",
          securityPolicy: "Basic256",
          clientCertificate: "",
          clientPrivateKey: "",
          serverCertificate: "",
        },
      },
    });
    await renderModalOpener(OPCUA.Device.useConnectModal, [{ deviceKey: dev.key }], {
      client,
    });
    await screen.findByDisplayValue(dev.name);
    expect(screen.getByDisplayValue("opc.tcp://existing-server:4840")).toBeTruthy();
    expect(screen.getByDisplayValue("operator")).toBeTruthy();
    await screen.findByText("Client certificate");
  });

  it("should reveal certificate fields when a security mode is enabled", async () => {
    await renderModalOpener(OPCUA.Device.useConnectModal, [{}], { client });
    await screen.findByText("Server");
    fireEvent.click(screen.getByText("Sign"));
    await screen.findByText("Client certificate");
    expect(screen.getByText("Client private key")).toBeTruthy();
    expect(screen.getByText("Server certificate")).toBeTruthy();
    expect(screen.getByText("Basic 256-bit")).toBeTruthy();
    fireEvent.click(screen.getAllByText("None")[0]);
    await waitFor(() => expect(screen.queryByText("Client certificate")).toBeNull());
  });

  it("should report a device it connected", async () => {
    const rack = await client.racks.create({
      name: uniqueName("opcua_rack"),
      integrations: ["opc"],
    });
    const scan = await rack.createTask({
      name: uniqueName("opcua_scan"),
      type: OPCUA.Task.SCAN_TYPE,
      config: {},
    });
    const analytics = createTestSink();
    await renderModalOpener(OPCUA.Device.useConnectModal, [{}], { client, analytics });
    await selectFromDropdown("Select Driver", rack.name);
    fireEvent.change(screen.getByPlaceholderText("opc.tcp://localhost:4840"), {
      target: { value: "opc.tcp://localhost:4840" },
    });
    const { answered } = await answerNextCommand(client, scan);
    pressSaveTrigger();
    await answered;
    await waitFor(() =>
      expect(analytics.capture).toHaveBeenCalledWith("device_connected", {
        integration: "opcua",
      }),
    );
  });

  // Submitting with no rack chosen fails validation. That error is the proof the keys
  // reached the same save path the Connect button uses.
  it("should submit on the shortcut its footer advertises", async () => {
    await renderModalOpener(OPCUA.Device.useConnectModal, [{}], { client });
    await screen.findByRole("dialog");
    pressSaveTrigger();
    expect(await screen.findByText(/rack is required/i)).toBeTruthy();
  });
});
