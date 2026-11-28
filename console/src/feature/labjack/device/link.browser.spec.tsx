// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { Dialog } from "@synnaxlabs/lyra/dialog";
import { uniqueName } from "@synnaxlabs/x/testutil";
import { act, render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { LabJack } from "@/feature/labjack";
import { createConsoleWrapper, renderLinkHook } from "@/testutil";

const client = createTestClient();

describe("LabJack.Device.useLink", () => {
  it("should open the typed configure form for the linked device", async () => {
    const rack = await client.racks.create({ name: uniqueName("labjack_rack") });
    const dev = await client.labjack.devices.create({
      rack: rack.key,
      name: "Test stand T7",
      serial: Math.floor(Math.random() * 400_000) * 10_000 + 3,
      connection: "LJM_ctUSB",
      model: "t7",
      ports: { model: "t7" },
    });
    const { handler, modals } = await renderLinkHook(LabJack.Device.useLink);
    await handler({ client, key: dev.key });
    const [entry] = modals.getState();
    expect(entry).toBeDefined();
    const { wrapper } = await createConsoleWrapper({ client });
    await act(async () => {
      render(
        <Dialog.Frame variant="modal" visible>
          {entry.render()}
        </Dialog.Frame>,
        { wrapper },
      );
    });
    expect(await screen.findByDisplayValue("Test stand T7")).toBeTruthy();
  });
});
