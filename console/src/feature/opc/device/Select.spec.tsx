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
import { renderWithDeviceForm } from "@/platform/device/testutil";
import { Modals } from "@/platform/modals";
import { findDialogTrigger } from "@/testutil";

const client = createTestClient();

describe("OPC.Device.Select", () => {
  it("should display the selected server in the trigger", async () => {
    const dev = await createOPCDevice(client);
    await renderWithDeviceForm(<OPC.Device.Select />, {
      deviceKey: dev.key,
      client,
    });
    expect(screen.getByText("OPC UA Server")).toBeTruthy();
    const trigger = await findDialogTrigger();
    await waitFor(() => expect(trigger.textContent).toContain(dev.name));
  });

  it("should open the connect modal when an unconfigured server is selected", async () => {
    const dev = await createOPCDevice(client, { configured: false });
    await renderWithDeviceForm(
      <>
        <OPC.Device.Select />
        <Modals.Stack />
      </>,
      { deviceKey: "", client },
    );
    fireEvent.click(await findDialogTrigger());
    fireEvent.change(await screen.findByPlaceholderText("Search devices..."), {
      target: { value: dev.name },
    });
    fireEvent.click(await screen.findByText(dev.name));
    await screen.findByText("Server");
    await waitFor(() => expect(screen.getByDisplayValue(dev.name)).toBeTruthy());
  });
});
