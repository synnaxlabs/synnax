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
import { createSiftDevice } from "@/feature/sift/testutil";
import { renderWithDeviceForm } from "@/platform/device/testutil";
import { Modals } from "@/platform/modals";
import { findDialogTrigger } from "@/testutil";

const client = createTestClient();

describe("Sift.Device.Select", () => {
  it("should display the selected connection in the trigger", async () => {
    const dev = await createSiftDevice(client);
    await renderWithDeviceForm(<Sift.Device.Select />, { deviceKey: dev.key, client });
    expect(screen.getByText("Sift connection")).toBeTruthy();
    const trigger = await findDialogTrigger();
    await waitFor(() => expect(trigger.textContent).toContain(dev.name));
  });

  it("should open the connect modal from the footer", async () => {
    await renderWithDeviceForm(
      <>
        <Sift.Device.Select />
        <Modals.Stack />
      </>,
      { deviceKey: "", client },
    );
    fireEvent.click(await findDialogTrigger());
    fireEvent.click(await screen.findByText("Connect Sift"));
    expect(
      await screen.findByPlaceholderText("grpc-api.siftstack.com:443"),
    ).toBeTruthy();
  });
});
