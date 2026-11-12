// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { task } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { describe, expect, it } from "vitest";

import { renderPalette } from "@/feature/command/testutil";
import { Sift } from "@/feature/sift";
import { Session } from "@/session";
import { resolveFocusedTab, uniqueName } from "@/testutil";

const client = createTestClient();

describe("Sift.Task commands", () => {
  it("should create a stream task draft and open its tab from the command", async () => {
    const proj = await client.projects.create({ name: uniqueName("proj") });
    const { store, openCommandPalette, selectCommand } = await renderPalette({
      commands: Sift.Task.COMMANDS,
      client,
    });
    store.dispatch(Session.Project.select(proj.key));
    await openCommandPalette("Create Sift");
    await selectCommand("Create Sift stream task");
    const tab = await resolveFocusedTab(store, client);
    if (tab.variant !== "resource") throw new Error("expected a resource tab");
    expect(tab.resource.type).toBe(task.TYPE_RESOURCE_ID.type);
    const created = await client.tasks.retrieve({ key: tab.resource.key });
    expect(created.type).toBe(Sift.Task.STREAM_TYPE);
  });
});
