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
import { Sift } from "@/feature/sift";
import { findCommand } from "@/platform/command/testutil";
import { assertDefined, renderHookWithConsole } from "@/testutil";

const client = createTestClient();
const roles = new RoleClients(client);

describe("Sift.Device commands", () => {
  it("should open the connect modal from the Connect Sift command", async () => {
    const { openCommandPalette, selectCommand } = await renderPalette({
      commands: Sift.Device.COMMANDS,
      client,
    });
    await openCommandPalette("Connect Sift");
    await selectCommand("Connect Sift");
    expect(
      await screen.findByPlaceholderText("grpc-api.siftstack.com:443"),
    ).toBeTruthy();
  });

  it("should offer Connect Sift to an engineer", async () => {
    const gate = findCommand(Sift.Device.COMMANDS, "Connect Sift").useVisible;
    assertDefined(gate);
    const { result } = await renderHookWithConsole(gate, {
      client: await roles.get("Engineer"),
    });
    await waitFor(() => expect(result.current).toBe(true));
  });

  it("should withhold Connect Sift from a viewer", async () => {
    const gate = findCommand(Sift.Device.COMMANDS, "Connect Sift").useVisible;
    assertDefined(gate);
    const { result } = await renderHookWithConsole(
      () => ({
        visible: gate(),
        loaded: Access.useRetrieveGranted(device.TYPE_RESOURCE_ID),
      }),
      { client: await roles.get("Viewer") },
    );
    await waitFor(() => expect(result.current.loaded).toBe(true));
    expect(result.current.visible).toBe(false);
  });
});
