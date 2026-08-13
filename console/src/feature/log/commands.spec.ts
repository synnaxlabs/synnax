// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { log } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { describe, expect, it } from "vitest";

import { renderPalette } from "@/feature/command/testutil";
import { Log } from "@/feature/log";
import { Session } from "@/session";
import { resolveFocusedTab, uniqueName } from "@/testutil";

const client = createTestClient();

describe("Log Commands", () => {
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
    const tab = await resolveFocusedTab(store, client);
    if (tab.variant !== "resource") throw new Error("expected a resource tab");
    expect(tab.resource.type).toBe(log.TYPE_ONTOLOGY_ID.type);
    const created = await client.logs.retrieve(tab.resource.key);
    expect(created.name).toBe("Log");
  });
});
