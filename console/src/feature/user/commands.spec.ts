// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { user } from "@synnaxlabs/client";
import { createTestClient, RoleClients } from "@synnaxlabs/client/testutil";
import { Access } from "@synnaxlabs/pluto";
import { screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { renderPalette } from "@/feature/command/testutil";
import { User } from "@/feature/user";
import { findCommand } from "@/platform/command/testutil";
import { findModalButton } from "@/platform/tree/menuTestutil";

const client = createTestClient();
const roles = new RoleClients(client);

import { assertDefined, renderHookWithConsole } from "@/testutil";

describe("User Commands", () => {
  it("should open the register modal when the command is selected", async () => {
    const { openCommandPalette, selectCommand } = await renderPalette({
      commands: User.COMMANDS,
      client,
    });
    await openCommandPalette();
    await selectCommand("Register a user");
    await screen.findByRole("dialog");
    expect(findModalButton("Register")).toBeTruthy();
  });
});

describe("User Commands permissions", () => {
  it("should offer Register a user to an owner", async () => {
    const gate = findCommand(User.COMMANDS, "Register a user").useVisible;
    assertDefined(gate);
    const { result } = await renderHookWithConsole(gate, {
      client: await roles.get("Owner"),
    });
    await waitFor(() => expect(result.current).toBe(true));
  });

  it("should withhold Register a user from a viewer", async () => {
    const gate = findCommand(User.COMMANDS, "Register a user").useVisible;
    assertDefined(gate);
    const { result } = await renderHookWithConsole(
      () => ({
        visible: gate(),
        loaded: Access.useRetrieveGranted(user.TYPE_ONTOLOGY_ID),
      }),
      { client: await roles.get("Viewer") },
    );
    await waitFor(() => expect(result.current.loaded).toBe(true));
    expect(result.current.visible).toBe(false);
  });
});
