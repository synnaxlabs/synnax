// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, RoleClients } from "@synnaxlabs/client/testutil";
import { screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { renderPalette } from "@/feature/command/testutil";
import { Status } from "@/feature/status";
import { findCommand } from "@/platform/command/testutil";
import { findModalButton } from "@/platform/tree/menuTestutil";
import { Session } from "@/session";
import {
  assertDefined,
  renderHookWithConsole,
  resolveFocusedTab,
  uniqueName,
} from "@/testutil";

const client = createTestClient();
const roles = new RoleClients(client);

describe("Status Commands", () => {
  it("should open the status creation modal when the create command is selected", async () => {
    const { openCommandPalette, selectCommand } = await renderPalette({
      commands: Status.COMMANDS,
      client,
    });
    await openCommandPalette();
    await selectCommand("Create status");
    await screen.findByRole("dialog");
    expect(findModalButton("Create")).toBeTruthy();
  });

  it("should open the status explorer as a tab when the explorer command is selected", async () => {
    const proj = await client.projects.create({
      name: uniqueName("proj"),
      layout: {},
    });
    const { store, openCommandPalette, selectCommand } = await renderPalette({
      commands: Status.COMMANDS,
      client,
    });
    store.dispatch(Session.Project.select(proj.key));
    await openCommandPalette();
    await selectCommand("Open status explorer");
    const tab = await resolveFocusedTab(store, client);
    if (tab.variant !== "view") throw new Error("expected a view tab");
    expect(tab.type).toBe(Status.Explorer.TAB_TYPE);
  });
});

describe("Status Commands permissions", () => {
  it("should offer Create a status to an engineer", async () => {
    const gate = findCommand(Status.COMMANDS, "Create status").useVisible;
    assertDefined(gate);
    const { result } = await renderHookWithConsole(gate, {
      client: await roles.get("Engineer"),
    });
    await waitFor(() => expect(result.current).toBe(true));
  });

  it("should withhold Create a status from a viewer", async () => {
    const gate = findCommand(Status.COMMANDS, "Create status").useVisible;
    assertDefined(gate);
    const read = findCommand(Status.COMMANDS, "Open status explorer").useVisible;
    assertDefined(read);
    const { result } = await renderHookWithConsole(
      () => ({ visible: gate(), readable: read() }),
      { client: await roles.get("Viewer") },
    );
    await waitFor(() => expect(result.current.readable).toBe(true));
    expect(result.current.visible).toBe(false);
  });

  it("should still offer Open the Status Explorer to a viewer", async () => {
    const gate = findCommand(Status.COMMANDS, "Open status explorer").useVisible;
    assertDefined(gate);
    const { result } = await renderHookWithConsole(gate, {
      client: await roles.get("Viewer"),
    });
    await waitFor(() => expect(result.current).toBe(true));
  });
});
