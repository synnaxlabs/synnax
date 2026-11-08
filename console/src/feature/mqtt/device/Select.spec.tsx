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

import { MQTT } from "@/feature/mqtt";
import { createBroker } from "@/feature/mqtt/testutil";
import { renderWithDeviceForm } from "@/platform/device/testutil";
import { Modals } from "@/platform/modals";
import { findDialogTrigger } from "@/testutil";

const client = createTestClient();

const renderSelect = async () =>
  await renderWithDeviceForm(
    <>
      <MQTT.Device.Select />
      <Modals.Stack />
    </>,
    { deviceKey: "", client },
  );

describe("MQTT.Device.Select", () => {
  it("should open the connect modal from the connect action", async () => {
    await renderSelect();
    fireEvent.click(await findDialogTrigger());
    fireEvent.click(await screen.findByText("Connect MQTT broker"));
    expect(await screen.findByPlaceholderText("broker.example.com")).toBeTruthy();
  });

  it("should open the connect modal when an unconfigured broker is selected", async () => {
    const dev = await createBroker(client, { configured: false });
    await renderSelect();
    fireEvent.click(await findDialogTrigger());
    fireEvent.change(await screen.findByPlaceholderText("Search devices..."), {
      target: { value: dev.name },
    });
    fireEvent.click(await screen.findByText(dev.name));
    await screen.findByPlaceholderText("broker.example.com");
    await waitFor(() => expect(screen.getByDisplayValue(dev.name)).toBeTruthy());
  });
});
