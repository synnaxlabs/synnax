// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { HTTP } from "@/feature/http";
import { createTestDevice, renderWithDeviceForm } from "@/platform/device/testutil";
import { Modals } from "@/platform/modals";
import { findDialogTrigger, uniqueName } from "@/testutil";

const renderSelect = async (client: Synnax | null) =>
  await renderWithDeviceForm(
    <>
      <HTTP.Device.Select />
      <Modals.Stack />
    </>,
    { deviceKey: "", client },
  );

describe("HTTP Device.Select", () => {
  it("should open the connect modal from the empty content", async () => {
    const client = createTestClient();
    await renderSelect(client);
    fireEvent.click(await findDialogTrigger());
    fireEvent.change(await screen.findByPlaceholderText("Search devices..."), {
      target: { value: uniqueName("no_such_device") },
    });
    await screen.findByText("No HTTP servers connected.");
    fireEvent.click(screen.getByText("Connect a new server"));
    await screen.findByText("Server");
    expect(screen.getByPlaceholderText("www.example.com")).toBeTruthy();
  });

  it("should open the connect modal when an unconfigured HTTP server is selected", async () => {
    const client = createTestClient();
    const dev = await createTestDevice(client, {
      name: uniqueName("http_server"),
      make: HTTP.Device.MAKE,
      model: "HTTP server",
      properties: HTTP.Device.ZERO_PROPERTIES,
      configured: false,
    });
    await renderSelect(client);
    fireEvent.click(await findDialogTrigger());
    fireEvent.change(await screen.findByPlaceholderText("Search devices..."), {
      target: { value: dev.name },
    });
    fireEvent.click(await screen.findByText(dev.name));
    await screen.findByText("Server");
    const nameInput = await screen.findByDisplayValue(dev.name);
    expect(nameInput).toBeTruthy();
  });
});
