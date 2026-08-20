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
import { pressSaveTrigger, renderModalOpener } from "@/platform/modals/testutil";

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

  // The footer has always advertised Ctrl+Enter, but nothing bound it, so the keys
  // did nothing. Submitting with no rack chosen fails validation, and that error is
  // the proof the shortcut reached the same save path the Connect button uses.
  it("should submit on the shortcut its footer advertises", async () => {
    await renderModalOpener(OPCUA.Device.useConnectModal, [{}], { client });
    await screen.findByRole("dialog");
    pressSaveTrigger();
    expect(await screen.findByText(/rack is required/i)).toBeTruthy();
  });
});
