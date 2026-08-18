// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { table } from "@synnaxlabs/client";
import { RoleClients } from "@synnaxlabs/client/testutil";
import { Access } from "@synnaxlabs/pluto";
import { waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { renderPalette } from "@/feature/command/testutil";
import { Table } from "@/feature/table";
import { client, project } from "@/feature/table/testutil";
import { findCommand } from "@/platform/command/testutil";
import { createActiveState } from "@/platform/project/testutil";
import { Session } from "@/session";
import { assertDefined, renderHookWithConsole, resolveFocusedTab } from "@/testutil";

const roles = new RoleClients(client);

describe("Table Commands", () => {
  it("creates a table on the server and opens it as a tab", async () => {
    const proj = await client.projects.retrieve(await project());
    const { store, openCommandPalette, selectCommand } = await renderPalette({
      commands: Table.COMMANDS,
      client,
      preloadedState: { [Session.Project.SLICE_NAME]: createActiveState(proj) },
    });
    await openCommandPalette();
    await selectCommand("Create table");
    const tab = await resolveFocusedTab(store, client);
    if (tab.variant !== "resource") throw new Error("expected a resource tab");
    expect(tab.resource.type).toBe(table.TYPE_ONTOLOGY_ID.type);
    const created = await client.tables.retrieve(tab.resource.key);
    expect(created.name).toBe("Table");
  });
});

describe("Table Commands permissions", () => {
  it("should offer Create a table to an engineer", async () => {
    const gate = findCommand(Table.COMMANDS, "Create table").useVisible;
    assertDefined(gate);
    const { result } = await renderHookWithConsole(gate, {
      client: await roles.get("Engineer"),
    });
    await waitFor(() => expect(result.current).toBe(true));
  });

  it("should withhold Create a table from a viewer", async () => {
    const gate = findCommand(Table.COMMANDS, "Create table").useVisible;
    assertDefined(gate);
    const { result } = await renderHookWithConsole(
      () => ({
        visible: gate(),
        loaded: Access.useRetrieveGranted(table.TYPE_ONTOLOGY_ID),
      }),
      { client: await roles.get("Viewer") },
    );
    await waitFor(() => expect(result.current.loaded).toBe(true));
    expect(result.current.visible).toBe(false);
  });
});
