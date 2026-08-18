// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { device } from "@synnaxlabs/client";
import { createTestClient, RoleClients } from "@synnaxlabs/client/testutil";
import { Access } from "@synnaxlabs/pluto";
import { screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { renderPalette } from "@/feature/command/testutil";
import { HTTP } from "@/feature/http";
import { findCommand } from "@/platform/command/testutil";

const client = createTestClient();
const roles = new RoleClients(client);

import { assertDefined, renderHookWithConsole } from "@/testutil";

describe("HTTP.Device Commands", () => {
  it("should open the connect modal from the connect server command", async () => {
    const { openCommandPalette, selectCommand } = await renderPalette({
      commands: HTTP.Device.COMMANDS,
      client,
    });
    await openCommandPalette("Connect an HTTP");
    await selectCommand("Connect HTTP server");
    expect(await screen.findByPlaceholderText("www.example.com")).toBeTruthy();
  });
});

describe("HTTP.Device Commands permissions", () => {
  it("should offer Connect an HTTP server to an engineer", async () => {
    const gate = findCommand(HTTP.Device.COMMANDS, "Connect HTTP server").useVisible;
    assertDefined(gate);
    const { result } = await renderHookWithConsole(gate, {
      client: await roles.get("Engineer"),
    });
    await waitFor(() => expect(result.current).toBe(true));
  });

  it("should withhold Connect an HTTP server from a viewer", async () => {
    const gate = findCommand(HTTP.Device.COMMANDS, "Connect HTTP server").useVisible;
    assertDefined(gate);
    const { result } = await renderHookWithConsole(
      () => ({
        visible: gate(),
        loaded: Access.useRetrieveGranted(device.TYPE_ONTOLOGY_ID),
      }),
      { client: await roles.get("Viewer") },
    );
    await waitFor(() => expect(result.current.loaded).toBe(true));
    expect(result.current.visible).toBe(false);
  });
});
