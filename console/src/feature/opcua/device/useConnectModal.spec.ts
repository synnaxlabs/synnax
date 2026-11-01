// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type rack, type task, TimeSpan } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { OPCUA } from "@/feature/opcua";
import { createOPCDevice, SERVER_CERTIFICATE } from "@/feature/opcua/testutil";
import { createTestSink } from "@/platform/analytics/testutil";
import { createRunningRack } from "@/platform/device/testutil";
import {
  findButton,
  findLastButton,
  getSwitch,
  pressSaveTrigger,
  renderModalOpener,
} from "@/platform/modals/testutil";
import {
  answerNextCommand,
  commitFieldInput,
  selectFromDropdown,
} from "@/platform/task/testutil";
import {
  captureBrowserDownloads,
  findDialogTriggerByText,
  uniqueName,
} from "@/testutil";

const client = createTestClient();

const DRIVER_CERTIFICATE: OPCUA.Task.CertificateInfo = {
  ...SERVER_CERTIFICATE,
  subject: "CN=Synnax Driver",
  thumbprint: "0123456789ABCDEF0123456789ABCDEF01234567",
  notAfter: "2029-01-15T00:00:00Z",
};

const NEXT_DRIVER_CERTIFICATE: OPCUA.Task.CertificateInfo = {
  ...DRIVER_CERTIFICATE,
  thumbprint: "89ABCDEF0123456789ABCDEF0123456789ABCDEF",
  notAfter: "2032-01-15T00:00:00Z",
};

const TEST_CONNECTION = { type: OPCUA.Task.TEST_CONNECTION_COMMAND_TYPE };

const ENDPOINTS: OPCUA.Task.Endpoint[] = [
  {
    url: "opc.tcp://localhost:4840",
    securityMode: "None",
    securityPolicy: "None",
    userTokenTypes: ["anonymous"],
    securityLevel: 0,
  },
  {
    url: "opc.tcp://localhost:4840",
    securityMode: "Sign",
    securityPolicy: "ECC_nistP256",
    userTokenTypes: ["anonymous"],
    securityLevel: 2,
  },
  {
    url: "opc.tcp://localhost:4840",
    securityMode: "SignAndEncrypt",
    securityPolicy: "Aes256_Sha256_RsaPss",
    userTokenTypes: ["anonymous", "username"],
    securityLevel: 3,
  },
  {
    url: "opc.tcp://127.0.0.1:4840",
    securityMode: "SignAndEncrypt",
    securityPolicy: "Aes256_Sha256_RsaPss",
    userTokenTypes: ["username", "certificate"],
    securityLevel: 3,
  },
];

const GET_ENDPOINTS = { type: OPCUA.Task.GET_ENDPOINTS_COMMAND_TYPE };

const createScanTask = async (rck: rack.Rack) =>
  await rck.createTask({
    name: uniqueName("opcua_scan"),
    type: OPCUA.Task.SCAN_TYPE,
    config: {},
  });

const answerGetCertificate = async (
  scan: task.Task,
  next?: OPCUA.Task.CertificateInfo,
) =>
  await answerNextCommand(client, scan, {
    type: OPCUA.Task.GET_CERTIFICATE_COMMAND_TYPE,
    data: { current: DRIVER_CERTIFICATE, next },
  });

const toDER = (pem: string): Uint8Array =>
  new Uint8Array(
    Buffer.from(pem.replace(/-----(BEGIN|END) CERTIFICATE-----|\s/g, ""), "base64"),
  );

afterEach(() => {
  vi.restoreAllMocks();
});

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
          certificateValidationDisabled: false,
          clientTimeoutMs: 2000,
          secureChannelLifetimeMs: 60_000,
          sessionTimeoutMs: 120_000,
        },
      },
    });
    await renderModalOpener(OPCUA.Device.useConnectModal, [{ deviceKey: dev.key }], {
      client,
    });
    await screen.findByDisplayValue(dev.name);
    expect(screen.getByDisplayValue("2000")).toBeTruthy();
    expect(screen.getByDisplayValue("60000")).toBeTruthy();
    expect(screen.getByDisplayValue("120000")).toBeTruthy();
    expect(screen.getByDisplayValue("opc.tcp://existing-server:4840")).toBeTruthy();
    expect(screen.getByDisplayValue("operator")).toBeTruthy();
    await screen.findByText("Client certificate");
  });

  it("should reveal certificate fields when a security mode is enabled", async () => {
    await renderModalOpener(OPCUA.Device.useConnectModal, [{}], { client });
    await screen.findByText("OPC UA server");
    fireEvent.click(screen.getByText("Sign"));
    await screen.findByText("Client certificate");
    expect(screen.getByText("Server certificate")).toBeTruthy();
    expect(screen.getByText("Validate server certificate")).toBeTruthy();
    fireEvent.click(screen.getByText("None"));
    await waitFor(() => expect(screen.queryByText("Client certificate")).toBeNull());
  });

  it("should save a policy other than None with a security mode", async () => {
    const rack = await createRunningRack(client, "opc");
    const scan = await createScanTask(rack);
    const { result } = await renderModalOpener(
      OPCUA.Device.useConnectModal,
      [{ rack: rack.key }],
      { client },
    );
    fireEvent.click(await screen.findByText("Sign"));
    await screen.findByText("Basic 256-bit with SHA-256");
    const { answered } = await answerNextCommand(client, scan, TEST_CONNECTION);
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
    const scan = await createScanTask(rack);
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
    const { answered } = await answerNextCommand(client, scan, TEST_CONNECTION);
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
    const scan = await createScanTask(rack);
    const { result } = await renderModalOpener(
      OPCUA.Device.useConnectModal,
      [{ rack: rack.key }],
      { client },
    );
    fireEvent.change(screen.getByPlaceholderText("opc.tcp://localhost:4840"), {
      target: { value: "opc.tcp://localhost:4840" },
    });
    const { answered } = await answerNextCommand(client, scan, TEST_CONNECTION);
    pressSaveTrigger();
    await answered;
    const key = await result();
    if (key == null) throw new Error("the modal closed without a device");
    expect((await client.devices.retrieve({ key })).rack).toBe(rack.key);
  });

  it("should report a device it connected", async () => {
    const rack = await createRunningRack(client, "opc");
    const scan = await createScanTask(rack);
    const analytics = createTestSink();
    await renderModalOpener(OPCUA.Device.useConnectModal, [{ rack: rack.key }], {
      client,
      analytics,
    });
    fireEvent.change(screen.getByPlaceholderText("opc.tcp://localhost:4840"), {
      target: { value: "opc.tcp://localhost:4840" },
    });
    const { answered } = await answerNextCommand(client, scan, TEST_CONNECTION);
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
    const scan = await createScanTask(rack);
    await renderModalOpener(OPCUA.Device.useConnectModal, [{ rack: rack.key }], {
      client,
    });
    fireEvent.change(screen.getByPlaceholderText("opc.tcp://localhost:4840"), {
      target: { value: "opc.tcp://localhost:4840" },
    });
    const { answered } = await answerNextCommand(client, scan, {
      ...TEST_CONNECTION,
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

  it("should caution against a deprecated security policy", async () => {
    const dev = await createOPCDevice(client, {
      properties: {
        connection: {
          ...OPCUA.Device.ZERO_PROPERTIES.connection,
          securityMode: "Sign",
          securityPolicy: "Basic256",
        },
      },
    });
    await renderModalOpener(OPCUA.Device.useConnectModal, [{ deviceKey: dev.key }], {
      client,
    });
    expect(
      await screen.findByText("Use only if the server supports nothing newer."),
    ).toBeTruthy();
  });

  it("should show the Driver certificate of the selected rack", async () => {
    const rack = await createRunningRack(client, "opc");
    const scan = await createScanTask(rack);
    const { answered } = await answerGetCertificate(scan, NEXT_DRIVER_CERTIFICATE);
    await renderModalOpener(OPCUA.Device.useConnectModal, [{ rack: rack.key }], {
      client,
    });
    fireEvent.click(await screen.findByText("Sign"));
    await answered;
    expect(
      await screen.findByText(
        "01:23:45:67:89:AB:CD:EF:01:23:45:67:89:AB:CD:EF:01:23:45:67",
      ),
    ).toBeTruthy();
    expect(screen.getByText("Expires 2029-01-15")).toBeTruthy();
    expect(screen.getByText("Next certificate")).toBeTruthy();
    expect(
      screen.getByText("89:AB:CD:EF:01:23:45:67:89:AB:CD:EF:01:23:45:67:89:AB:CD:EF"),
    ).toBeTruthy();
    expect(screen.getByText("Expires 2032-01-15")).toBeTruthy();
  });

  it("should download the Driver certificate as DER", async () => {
    const rack = await createRunningRack(client, "opc");
    const scan = await createScanTask(rack);
    const { answered } = await answerGetCertificate(scan);
    await renderModalOpener(OPCUA.Device.useConnectModal, [{ rack: rack.key }], {
      client,
    });
    fireEvent.click(await screen.findByText("Sign"));
    await answered;
    await screen.findByText("Expires 2029-01-15");
    const downloads = captureBrowserDownloads();
    fireEvent.click(findButton("Download"));
    await waitFor(() => expect(downloads.anchors).toHaveLength(1));
    expect(downloads.anchors[0].download).toBe("driver.der");
    expect(new Uint8Array(await downloads.blobs[0].arrayBuffer())).toEqual(
      toDER(DRIVER_CERTIFICATE.pem),
    );
  });

  it("should show the new Driver certificate after a confirmed regenerate", async () => {
    const rack = await createRunningRack(client, "opc");
    const scan = await createScanTask(rack);
    const loaded = await answerGetCertificate(scan);
    await renderModalOpener(OPCUA.Device.useConnectModal, [{ rack: rack.key }], {
      client,
    });
    fireEvent.click(await screen.findByText("Sign"));
    await loaded.answered;
    await screen.findByText("Expires 2029-01-15");
    const regenerated = await answerNextCommand(client, scan, {
      type: OPCUA.Task.REGENERATE_CERTIFICATE_COMMAND_TYPE,
      data: { current: NEXT_DRIVER_CERTIFICATE },
    });
    fireEvent.click(findButton("Regenerate"));
    await screen.findByText(
      "Are you sure you want to regenerate the Driver certificate?",
    );
    fireEvent.click(findLastButton("Regenerate"));
    await regenerated.answered;
    expect(await screen.findByText("Expires 2032-01-15")).toBeTruthy();
    expect(screen.queryByText("Expires 2029-01-15")).toBeNull();
  });

  it("should show the thumbprint of the pinned server certificate", async () => {
    const dev = await createOPCDevice(client, {
      properties: {
        connection: {
          ...OPCUA.Device.ZERO_PROPERTIES.connection,
          securityMode: "Sign",
          securityPolicy: "Basic256Sha256",
          serverCertificate: SERVER_CERTIFICATE.pem,
        },
      },
    });
    await renderModalOpener(OPCUA.Device.useConnectModal, [{ deviceKey: dev.key }], {
      client,
    });
    expect(
      await screen.findByText(
        "FF:65:16:FC:E0:22:F9:7F:23:B6:F6:79:17:9D:B7:FE:68:CB:2D:9C",
      ),
    ).toBeTruthy();
  });

  it("should trust the certificate the server sent and connect again", async () => {
    const rack = await createRunningRack(client, "opc");
    const scan = await createScanTask(rack);
    const loaded = await answerGetCertificate(scan);
    const { result } = await renderModalOpener(
      OPCUA.Device.useConnectModal,
      [{ rack: rack.key }],
      { client },
    );
    fireEvent.click(await screen.findByText("Sign"));
    await loaded.answered;
    expect(await screen.findByText("Not trusted")).toBeTruthy();
    const rejected = await answerNextCommand(client, scan, {
      ...TEST_CONNECTION,
      variant: "error",
      message: "Server certificate not trusted",
      data: { serverCertificate: SERVER_CERTIFICATE },
    });
    pressSaveTrigger();
    await rejected.answered;
    expect(await screen.findByText(SERVER_CERTIFICATE.subject)).toBeTruthy();
    const accepted = await answerNextCommand(client, scan, TEST_CONNECTION);
    fireEvent.click(findButton("Trust"));
    await accepted.answered;
    const key = await result();
    if (key == null) throw new Error("the modal closed without a device");
    const dev = await client.devices.retrieve({ key, schemas: OPCUA.Device.SCHEMAS });
    expect(dev.properties.connection.serverCertificate).toBe(SERVER_CERTIFICATE.pem);
  });

  it("should show the file paths of a device that names them", async () => {
    const dev = await createOPCDevice(client, {
      properties: {
        connection: {
          ...OPCUA.Device.ZERO_PROPERTIES.connection,
          securityMode: "Sign",
          securityPolicy: "Basic256Sha256",
          clientCertificate: "/etc/opcua/client.der",
          clientPrivateKey: "/etc/opcua/client.key",
        },
      },
    });
    await renderModalOpener(OPCUA.Device.useConnectModal, [{ deviceKey: dev.key }], {
      client,
    });
    expect(await screen.findByDisplayValue("/etc/opcua/client.der")).toBeTruthy();
    expect(screen.getByDisplayValue("/etc/opcua/client.key")).toBeTruthy();
  });

  it("should clear the file paths when the Driver certificate is chosen again", async () => {
    const rack = await createRunningRack(client, "opc");
    const scan = await createScanTask(rack);
    const { result } = await renderModalOpener(
      OPCUA.Device.useConnectModal,
      [{ rack: rack.key }],
      { client },
    );
    fireEvent.click(await screen.findByText("Sign"));
    fireEvent.click(await screen.findByText("Files on the Driver computer"));
    const [certificate, privateKey] = screen.getAllByPlaceholderText<HTMLInputElement>(
      "Path on the Driver computer",
    );
    commitFieldInput(certificate, "/etc/opcua/client.der");
    commitFieldInput(privateKey, "/etc/opcua/client.key");
    fireEvent.click(screen.getByText("Driver certificate"));
    await waitFor(() =>
      expect(screen.queryByPlaceholderText("Path on the Driver computer")).toBeNull(),
    );
    const { answered } = await answerNextCommand(client, scan, TEST_CONNECTION);
    pressSaveTrigger();
    await answered;
    const key = await result();
    if (key == null) throw new Error("the modal closed without a device");
    const dev = await client.devices.retrieve({ key, schemas: OPCUA.Device.SCHEMAS });
    expect(dev.properties.connection).toMatchObject({
      clientCertificate: "",
      clientPrivateKey: "",
    });
  });

  it("should save certificate validation as disabled when its switch is off", async () => {
    const rack = await createRunningRack(client, "opc");
    const scan = await createScanTask(rack);
    const { result } = await renderModalOpener(
      OPCUA.Device.useConnectModal,
      [{ rack: rack.key }],
      { client },
    );
    fireEvent.click(await screen.findByText("Sign"));
    await screen.findByText("Validate server certificate");
    const validation = getSwitch("Validate server certificate");
    expect(validation.checked).toBe(true);
    fireEvent.click(validation);
    const { answered } = await answerNextCommand(client, scan, TEST_CONNECTION);
    pressSaveTrigger();
    await answered;
    const key = await result();
    if (key == null) throw new Error("the modal closed without a device");
    const dev = await client.devices.retrieve({ key, schemas: OPCUA.Device.SCHEMAS });
    expect(dev.properties.connection.certificateValidationDisabled).toBe(true);
  });

  it("should set the security mode and policy from an endpoint the server lists", async () => {
    const rack = await createRunningRack(client, "opc");
    const scan = await createScanTask(rack);
    const { result } = await renderModalOpener(
      OPCUA.Device.useConnectModal,
      [{ rack: rack.key }],
      { client },
    );
    const listed = await answerNextCommand(client, scan, {
      ...GET_ENDPOINTS,
      data: { endpoints: ENDPOINTS },
    });
    fireEvent.click(await screen.findByText("Find endpoints"));
    await listed.answered;
    await selectFromDropdown("None", "Sign and encrypt, AES 256-bit with SHA-256");
    await screen.findByText("Client certificate");
    const { answered } = await answerNextCommand(client, scan, TEST_CONNECTION);
    pressSaveTrigger();
    await answered;
    const key = await result();
    if (key == null) throw new Error("the modal closed without a device");
    const dev = await client.devices.retrieve({ key, schemas: OPCUA.Device.SCHEMAS });
    expect(dev.properties.connection).toMatchObject({
      securityMode: "SignAndEncrypt",
      securityPolicy: "Aes256_Sha256_RsaPss",
    });
  });

  it("should merge the endpoints, sort them, and mark unsupported policies", async () => {
    const rack = await createRunningRack(client, "opc");
    const scan = await createScanTask(rack);
    await renderModalOpener(OPCUA.Device.useConnectModal, [{ rack: rack.key }], {
      client,
    });
    const listed = await answerNextCommand(client, scan, {
      ...GET_ENDPOINTS,
      data: { endpoints: ENDPOINTS },
    });
    fireEvent.click(await screen.findByText("Find endpoints"));
    await listed.answered;
    fireEvent.click(await findDialogTriggerByText("None"));
    const list = await waitFor(() => screen.getAllByRole("dialog").at(-1));
    if (list == null) throw new Error("the endpoint list did not open");
    const names = within(list)
      .getAllByRole("option")
      .map((option) => option.textContent);
    expect(names).toEqual([
      "Sign and encrypt, AES 256-bit with SHA-256Anonymous, Username, Certificate",
      "Sign, ECC_nistP256 (not supported)Anonymous",
      "NoneAnonymous",
    ]);
  });

  it("should show why the server endpoints could not be listed", async () => {
    const rack = await createRunningRack(client, "opc");
    const scan = await createScanTask(rack);
    await renderModalOpener(OPCUA.Device.useConnectModal, [{ rack: rack.key }], {
      client,
    });
    const failed = await answerNextCommand(client, scan, {
      ...GET_ENDPOINTS,
      variant: "error",
      message: "Failed to get endpoints",
      description: "Could not establish a network connection to remote server.",
    });
    fireEvent.click(await screen.findByText("Find endpoints"));
    await failed.answered;
    expect(
      await screen.findByText(
        "Could not establish a network connection to remote server.",
      ),
    ).toBeTruthy();
  });

  it("should hide the endpoints of a rack the user moves away from", async () => {
    const first = await createRunningRack(client, "opc");
    const second = await createRunningRack(client, "opc");
    const scan = await createScanTask(first);
    await renderModalOpener(OPCUA.Device.useConnectModal, [{ rack: first.key }], {
      client,
    });
    const listed = await answerNextCommand(client, scan, {
      ...GET_ENDPOINTS,
      data: { endpoints: ENDPOINTS },
    });
    fireEvent.click(await screen.findByText("Find endpoints"));
    await listed.answered;
    await findDialogTriggerByText("None");
    fireEvent.click(await findDialogTriggerByText(first.name));
    const search = await waitFor(() => {
      const dialog = screen.getAllByRole("dialog").at(-1);
      if (dialog == null) throw new Error("the rack list did not open");
      return within(dialog).getByRole("textbox");
    });
    fireEvent.change(search, { target: { value: second.name } });
    fireEvent.click(await screen.findByText(second.name));
    await findDialogTriggerByText(second.name);
    expect(
      screen
        .getAllByRole("button")
        .some(
          (b) =>
            b.getAttribute("aria-haspopup") === "dialog" &&
            b.textContent?.includes("None"),
        ),
    ).toBe(false);
  });

  it("should save the timeouts the user enters", async () => {
    const rack = await createRunningRack(client, "opc");
    const scan = await createScanTask(rack);
    const { result } = await renderModalOpener(
      OPCUA.Device.useConnectModal,
      [{ rack: rack.key }],
      { client },
    );
    commitFieldInput(await screen.findByDisplayValue<HTMLInputElement>("5000"), "2500");
    commitFieldInput(screen.getByDisplayValue<HTMLInputElement>("1200000"), "90000");
    const { answered } = await answerNextCommand(client, scan, TEST_CONNECTION);
    pressSaveTrigger();
    await answered;
    const key = await result();
    if (key == null) throw new Error("the modal closed without a device");
    const dev = await client.devices.retrieve({ key, schemas: OPCUA.Device.SCHEMAS });
    expect(dev.properties.connection).toMatchObject({
      clientTimeoutMs: 2500,
      secureChannelLifetimeMs: 600_000,
      sessionTimeoutMs: 90_000,
    });
  });

  it("should wait twice the request timeout and a margin for the connection test", async () => {
    const rack = await createRunningRack(client, "opc");
    await createScanTask(rack);
    await renderModalOpener(OPCUA.Device.useConnectModal, [{ rack: rack.key }], {
      client,
    });
    fireEvent.change(screen.getByPlaceholderText("opc.tcp://localhost:4840"), {
      target: { value: "opc.tcp://localhost:4840" },
    });
    commitFieldInput(await screen.findByDisplayValue<HTMLInputElement>("5000"), "100");
    pressSaveTrigger();
    const message = `timed out after ${TimeSpan.milliseconds(2200).toString()}`;
    expect(await screen.findByText(message, { exact: false })).toBeTruthy();
  });
});
