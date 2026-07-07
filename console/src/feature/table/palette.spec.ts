// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { panel, table } from "@synnaxlabs/client";
import { describe, expect, it } from "vitest";

import { Table } from "@/feature/table";
import { client, project } from "@/feature/table/testutil";
import { renderPalette } from "@/platform/palette/testutil";
import { createActiveState } from "@/platform/project/testutil";
import { Session } from "@/session";
import { assertDefined, stubGeometry, waitForFocusedTab } from "@/testutil";

stubGeometry();

describe("table palette", () => {
  it("creates a table on the server and opens it as a tab", async () => {
    const proj = await client.projects.retrieve(await project());
    const { store, openCommandPalette, selectCommand } = await renderPalette({
      commands: Table.COMMANDS,
      client,
      preloadedState: { [Session.Project.SLICE_NAME]: createActiveState(proj) },
    });
    await openCommandPalette();
    await selectCommand("Create a table");
    const focused = await waitForFocusedTab(store);
    const panelKey = Session.Panel.selectSelected(store.getState());
    assertDefined(panelKey);
    const doc = await client.panels.retrieve(panelKey);
    const tab = panel.findTab(doc.root, focused);
    if (tab?.variant !== "resource") throw new Error("expected a resource tab");
    expect(tab.resource.type).toBe(table.TYPE_ONTOLOGY_ID.type);
    const created = await client.tables.retrieve({ key: tab.resource.key });
    expect(created.name).toBe("Table");
  });
});
