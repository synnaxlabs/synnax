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

import { OPC } from "@/feature/opc";
import { createOPCDevice } from "@/feature/opc/testutil";
import { renderModalOpener } from "@/platform/modals/testutil";

const client = createTestClient();

describe("OPC.Device.useConnectModal", () => {
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
    await renderModalOpener(OPC.Device.useConnectModal, [{ deviceKey: dev.key }], {
      client,
    });
    await screen.findByDisplayValue(dev.name);
    expect(screen.getByDisplayValue("opc.tcp://existing-server:4840")).toBeTruthy();
    expect(screen.getByDisplayValue("operator")).toBeTruthy();
    await screen.findByText("Client Certificate");
  });

  it("should reveal certificate fields when a security mode is enabled", async () => {
    await renderModalOpener(OPC.Device.useConnectModal, [{}], { client });
    await screen.findByText("Server");
    fireEvent.click(screen.getByText("Sign"));
    await screen.findByText("Client Certificate");
    expect(screen.getByText("Client Private Key")).toBeTruthy();
    expect(screen.getByText("Server Certificate")).toBeTruthy();
    expect(screen.getByText("Basic 256-bit")).toBeTruthy();
    fireEvent.click(screen.getAllByText("None")[0]);
    await waitFor(() => expect(screen.queryByText("Client Certificate")).toBeNull());
  });
});
