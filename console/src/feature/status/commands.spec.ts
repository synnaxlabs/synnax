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
import { Status } from "@/feature/status";
import { findModalButton } from "@/platform/tree/menuTestutil";
import { Session } from "@/session";
import { resolveFocusedTab, stubGeometry, uniqueName } from "@/testutil";

stubGeometry();

const client = createTestClient();

describe("Status Commands", () => {
  it("should open the status creation modal when the create command is selected", async () => {
    const { openCommandPalette, selectCommand } = await renderPalette({
      commands: Status.COMMANDS,
      client,
    });
    await openCommandPalette();
    await selectCommand("Create a status");
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
    await selectCommand("Open the Status Explorer");
    const tab = await resolveFocusedTab(store, client);
    if (tab.variant !== "view") throw new Error("expected a view tab");
    expect(tab.type).toBe(Status.Explorer.TAB_TYPE);
  });
});
