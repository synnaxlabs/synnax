// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { describe, expect, it } from "vitest";

import { renderPalette } from "@/feature/command/testutil";
import { Docs } from "@/feature/docs";
import { Session } from "@/session";
import { resolveFocusedTab, stubGeometry, uniqueName } from "@/testutil";

stubGeometry();

const client = createTestClient();

describe("docs palette", () => {
  it("should open the docs view as a tab when the read command is selected", async () => {
    const proj = await client.projects.create({
      name: uniqueName("proj"),
      layout: {},
    });
    const { store, openCommandPalette, selectCommand } = await renderPalette({
      commands: Docs.COMMANDS,
      client,
    });
    store.dispatch(Session.Project.select(proj.key));
    await openCommandPalette();
    await selectCommand("Read the documentation");
    const tab = await resolveFocusedTab(store, client);
    if (tab.variant !== "view") throw new Error("expected a view tab");
    expect(tab.type).toBe(Docs.TAB_TYPE);
  });
});
