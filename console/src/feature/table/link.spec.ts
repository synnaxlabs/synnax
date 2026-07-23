// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { id } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { Table } from "@/feature/table";
import { Session } from "@/session";
import { renderLinkHook, resolveFocusedTab } from "@/testutil";

const client = createTestClient();

describe("Table.useLink", () => {
  it("should open the table as a tab", async () => {
    const { layout: _, ...project } = await client.projects.create({
      name: id.create(),
      layout: {},
    });
    const table = await client.tables.create(project.key, { name: "Sensor Table" });
    const { handler, store } = await renderLinkHook(Table.useLink, { client });
    store.dispatch(Session.Project.select(project.key));
    await handler({ client, key: table.key });
    const tab = await resolveFocusedTab(store, client);
    if (tab.variant !== "resource") throw new Error("expected a resource tab");
    expect(tab.resource.type).toBe("table");
    const retrieved = await client.tables.retrieve({ key: tab.resource.key });
    expect(retrieved.name).toBe("Sensor Table");
  });
});
