// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { HTTP } from "@/feature/http";
import { renderPalette } from "@/platform/palette/testutil";
import { stubGeometry, waitForPlacedLayout } from "@/testutil";

stubGeometry();

const client = createTestClient();

describe("HTTP palette commands", () => {
  it("should open the connect modal from the connect server command", async () => {
    const { openCommandPalette, selectCommand } = await renderPalette({
      commands: HTTP.Device.COMMANDS,
      client,
    });
    await openCommandPalette("Connect an HTTP");
    await selectCommand("Connect an HTTP server");
    expect(await screen.findByPlaceholderText("www.example.com")).toBeTruthy();
  });

  it("should place the read task layout from the create read task command", async () => {
    const { store, openCommandPalette, selectCommand } = await renderPalette({
      commands: HTTP.Task.COMMANDS,
      client,
    });
    await openCommandPalette("Create an HTTP Read");
    await selectCommand("Create an HTTP Read Task");
    await waitForPlacedLayout(store, HTTP.Task.READ_TYPE);
  });

  it("should place the write task layout from the create write task command", async () => {
    const { store, openCommandPalette, selectCommand } = await renderPalette({
      commands: HTTP.Task.COMMANDS,
      client,
    });
    await openCommandPalette("Create an HTTP Write");
    await selectCommand("Create an HTTP Write Task");
    await waitForPlacedLayout(store, HTTP.Task.WRITE_TYPE);
  });
});
