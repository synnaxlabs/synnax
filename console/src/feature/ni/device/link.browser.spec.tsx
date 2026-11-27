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
import { id } from "@synnaxlabs/x";
import { uniqueName } from "@synnaxlabs/x/testutil";
import { act, render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { NI } from "@/feature/ni";
import { createConsoleWrapper, renderLinkHook } from "@/testutil";

const client = createTestClient();

describe("NI.Device.useLink", () => {
  it("should open the typed configure form for the linked device", async () => {
    const rack = await client.racks.create({ name: uniqueName("ni_rack") });
    const dev = await client.ni.devices.create({
      rack: rack.key,
      name: "Thermocouple module",
      resourceName: id.create(),
      model: "NI 9211",
    });
    const { handler, modals } = await renderLinkHook(NI.Device.useLink);
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
    expect(await screen.findByDisplayValue("Thermocouple module")).toBeTruthy();
  });
});
