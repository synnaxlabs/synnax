// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client";
import { describe, expect, it } from "vitest";

import { renderPalette } from "@/feature/command/testutil";
import { Log } from "@/feature/log";
import { Session } from "@/session";
import { stubGeometry, uniqueName, waitForPlacedLayout } from "@/testutil";

stubGeometry();

const client = createTestClient();

describe("Log Commands", () => {
  it("should create a log in the active project and place its layout", async () => {
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
    const key = await waitForPlacedLayout(store, Log.LAYOUT_TYPE);
    const created = await client.logs.retrieve({ key });
    expect(created.name).toBe("Log");
  });
});
