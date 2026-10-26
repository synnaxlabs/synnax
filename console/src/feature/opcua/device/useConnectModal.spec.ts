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
import { createRunningRack } from "@/platform/device/testutil";
import {
  findLastButton,
  pressSaveTrigger,
  renderModalOpener,
} from "@/platform/modals/testutil";
import { answerNextCommand } from "@/platform/task/testutil";
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
    await screen.findByText("OPC UA server");
    fireEvent.click(screen.getByText("Sign"));
    await screen.findByText("Client certificate");
    expect(screen.getByText("Client private key")).toBeTruthy();
    expect(screen.getByText("Server certificate")).toBeTruthy();
    fireEvent.click(screen.getByText("None"));
    await waitFor(() => expect(screen.queryByText("Client certificate")).toBeNull());
  });

  it("should save a policy other than None with a security mode", async () => {
    const rack = await createRunningRack(client, "opc");
    const scan = await rack.createTask({
      name: uniqueName("opcua_scan"),
      type: OPCUA.Task.SCAN_TYPE,
      config: {},
    });
    const { result } = await renderModalOpener(
      OPCUA.Device.useConnectModal,
      [{ rack: rack.key }],
      { client },
    );
    fireEvent.click(await screen.findByText("Sign"));
    await screen.findByText("Basic 256-bit with SHA-256");
    const { answered } = await answerNextCommand(client, scan);
    pressSaveTrigger();
    await answered;
    const key = await result();
    if (key == null) throw new Error("the modal closed without a device");
    const dev = await client.devices.retrieve({ key, schemas: OPCUA.Device.SCHEMAS });
    expect(dev.properties.connection).toMatchObject({
      securityMode: "Sign",
      securityPolicy: "Basic256Sha256",
    });
  });

  it("should save policy None when the security mode returns to None", async () => {
    const rack = await createRunningRack(client, "opc");
    const scan = await rack.createTask({
      name: uniqueName("opcua_scan"),
      type: OPCUA.Task.SCAN_TYPE,
      config: {},
    });
    const { result } = await renderModalOpener(
      OPCUA.Device.useConnectModal,
      [{ rack: rack.key }],
      { client },
    );
    fireEvent.click(await screen.findByText("Sign"));
    await screen.findByText("Basic 256-bit with SHA-256");
    fireEvent.click(screen.getByText("None"));
    await waitFor(() =>
      expect(screen.queryByText("Basic 256-bit with SHA-256")).toBeNull(),
    );
    const { answered } = await answerNextCommand(client, scan);
    pressSaveTrigger();
    await answered;
    const key = await result();
    if (key == null) throw new Error("the modal closed without a device");
    const dev = await client.devices.retrieve({ key, schemas: OPCUA.Device.SCHEMAS });
    expect(dev.properties.connection).toMatchObject({
      securityMode: "None",
      securityPolicy: "None",
    });
  });

  it("should resolve with the device it connected", async () => {
    const rack = await createRunningRack(client, "opc");
    const scan = await rack.createTask({
      name: uniqueName("opcua_scan"),
      type: OPCUA.Task.SCAN_TYPE,
      config: {},
    });
    const { result } = await renderModalOpener(
      OPCUA.Device.useConnectModal,
      [{ rack: rack.key }],
      { client },
    );
    fireEvent.change(screen.getByPlaceholderText("opc.tcp://localhost:4840"), {
      target: { value: "opc.tcp://localhost:4840" },
    });
    const { answered } = await answerNextCommand(client, scan);
    pressSaveTrigger();
    await answered;
    const key = await result();
    if (key == null) throw new Error("the modal closed without a device");
    expect((await client.devices.retrieve({ key })).rack).toBe(rack.key);
  });

  it("should report a device it connected", async () => {
    const rack = await createRunningRack(client, "opc");
    const scan = await rack.createTask({
      name: uniqueName("opcua_scan"),
      type: OPCUA.Task.SCAN_TYPE,
      config: {},
    });
    const analytics = createTestSink();
    await renderModalOpener(OPCUA.Device.useConnectModal, [{ rack: rack.key }], {
      client,
      analytics,
    });
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

  it("should start from the rack it opens with and block Connect while it is not running", async () => {
    const rack = await client.racks.create({
      name: uniqueName("opcua_rack"),
      integrations: ["opc"],
    });
    await renderModalOpener(OPCUA.Device.useConnectModal, [{ rack: rack.key }], {
      client,
    });
    await screen.findByText(`${rack.name} is not running`);
    expect(findLastButton("Connect").getAttribute("aria-disabled")).toBe("true");
  });

  it("should show a failure from a Driver that sends no description", async () => {
    const rack = await createRunningRack(client, "opc");
    const scan = await rack.createTask({
      name: uniqueName("opcua_scan"),
      type: OPCUA.Task.SCAN_TYPE,
      config: {},
    });
    await renderModalOpener(OPCUA.Device.useConnectModal, [{ rack: rack.key }], {
      client,
    });
    fireEvent.change(screen.getByPlaceholderText("opc.tcp://localhost:4840"), {
      target: { value: "opc.tcp://localhost:4840" },
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
    await createRunningRack(client, "opc");
    await renderModalOpener(OPCUA.Device.useConnectModal, [{}], { client });
    await screen.findByRole("dialog");
    pressSaveTrigger();
    expect(await screen.findByText(/rack is required/i)).toBeTruthy();
  });
});
