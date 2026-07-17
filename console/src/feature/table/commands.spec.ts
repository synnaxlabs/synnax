// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { renderPalette } from "@/feature/command/testutil";
import { Table } from "@/feature/table";
import { client, project } from "@/feature/table/testutil";
import { createActiveState } from "@/platform/project/testutil";
import { Session } from "@/session";
import { stubGeometry, waitForPlacedLayout } from "@/testutil";

stubGeometry();

describe("Table Commands", () => {
  it("creates a table on the server and places its layout", async () => {
    const proj = await client.projects.retrieve(await project());
    const { store, openCommandPalette, selectCommand } = await renderPalette({
      commands: Table.COMMANDS,
      client,
      preloadedState: { [Session.Project.SLICE_NAME]: createActiveState(proj) },
    });
    await openCommandPalette();
    await selectCommand("Create a table");
    const key = await waitForPlacedLayout(store, Table.LAYOUT_TYPE);
    const created = await client.tables.retrieve({ key });
    expect(created.name).toBe("Table");
  });
});
