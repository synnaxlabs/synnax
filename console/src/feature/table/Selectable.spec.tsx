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
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Table } from "@/feature/table";
import { client, project } from "@/feature/table/testutil";
import { createActiveState } from "@/platform/project/testutil";
import { Session } from "@/session";
import { assertDefined, createConsoleWrapper, resolveFocusedTab } from "@/testutil";

const roles = new RoleClients(client);

describe("table/Selectable", () => {
  it("creates a table in the active project and opens its tab when clicked", async () => {
    const proj = await client.projects.retrieve(await project());
    const { wrapper, store } = await createConsoleWrapper({
      client,
      preloadedState: { [Session.Project.SLICE_NAME]: createActiveState(proj) },
    });
    const Selectable = Table.SELECTABLES[0];
    expect(Selectable.type).toBe(table.TYPE_ONTOLOGY_ID.type);
    render(<Selectable />, { wrapper });
    fireEvent.click(await screen.findByText("Table"));
    const tab = await resolveFocusedTab(store, client, (t) => t.variant === "resource");
    if (tab.variant !== "resource") throw new Error("expected a resource tab");
    expect(tab.resource.type).toBe(table.TYPE_ONTOLOGY_ID.type);
    const key = tab.resource.key;
    const created = await client.tables.retrieve(key);
    expect(created.name).toBe("Table");
    await waitFor(() =>
      expect(Session.Table.selectEditable({ state: store.getState(), key })).toBe(true),
    );
  });
});

describe("table/Selectable permissions", () => {
  const findSelectable = () => {
    const Selectable = Table.SELECTABLES.find(
      (s) => s.type === table.TYPE_ONTOLOGY_ID.type,
    );
    assertDefined(Selectable, "no selectable registered for table");
    return Selectable;
  };

  it("should offer the tile to an engineer", async () => {
    const Selectable = findSelectable();
    const { wrapper } = await createConsoleWrapper({
      client: await roles.get("Engineer"),
    });
    render(<Selectable />, { wrapper });
    expect(await screen.findByText("Table")).toBeTruthy();
  });

  it("should withhold the tile from a viewer", async () => {
    const Selectable = findSelectable();
    const { wrapper } = await createConsoleWrapper({
      client: await roles.get("Viewer"),
    });
    const { container } = render(<Selectable />, { wrapper });
    await waitFor(() => expect(container.textContent).toBe(""));
  });
});
