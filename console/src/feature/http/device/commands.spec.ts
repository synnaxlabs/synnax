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

import { renderPalette } from "@/feature/command/testutil";
import { HTTP } from "@/feature/http";

const client = createTestClient();

describe("HTTP.Device Commands", () => {
  it("should open the connect modal from the connect server command", async () => {
    const { openCommandPalette, selectCommand } = await renderPalette({
      commands: HTTP.Device.COMMANDS,
      client,
    });
    await openCommandPalette("Connect an HTTP");
    await selectCommand("Connect an HTTP server");
    expect(await screen.findByPlaceholderText("www.example.com")).toBeTruthy();
  });
});
