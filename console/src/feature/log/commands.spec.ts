// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, log, panel } from "@synnaxlabs/client";
import { describe, expect, it } from "vitest";

import { renderPalette } from "@/feature/command/testutil";
import { Log } from "@/feature/log";
import { Session } from "@/session";
import { assertDefined, stubGeometry, uniqueName, waitForFocusedTab } from "@/testutil";

stubGeometry();

const client = createTestClient();

describe("log palette", () => {
  it("creates a log in the active project and opens it as a tab", async () => {
    const project = await client.projects.create({
      name: uniqueName("project"),
      layout: {},
    });
    const { store, openCommandPalette, selectCommand } = await renderPalette({
      commands: Log.COMMANDS,
      client,
      preloadedState: {
        [Session.Project.SLICE_NAME]: { version: 0, selected: project.key },
      },
    });
    await openCommandPalette();
    await selectCommand("Create a log");
    const focused = await waitForFocusedTab(store);
    const panelKey = Session.Panel.selectSelected(store.getState());
    assertDefined(panelKey);
    const doc = await client.panels.retrieve(panelKey);
    const tab = panel.findTab(doc.root, focused);
    if (tab?.variant !== "resource") throw new Error("expected a resource tab");
    expect(tab.resource.type).toBe(log.TYPE_ONTOLOGY_ID.type);
    const created = await client.logs.retrieve({ key: tab.resource.key });
    expect(created.name).toBe("Log");
  });
});
