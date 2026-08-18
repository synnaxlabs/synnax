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

import { Link } from "@/platform/link";
import { Session } from "@/session";
import { renderLinkHook, resolveFocusedTab, uniqueName } from "@/testutil";

const client = createTestClient();

const useOpenTable = Link.createUseOpenResourceTab("table");

describe("Link.createUseOpenResourceTab", () => {
  it("opens the linked key as a resource tab of the factory's type", async () => {
    const project = await client.projects.create({
      name: uniqueName("project"),
      layout: {},
    });
    const created = await client.tables.create(project.key, {
      name: uniqueName("table"),
    });
    const { handler, store } = await renderLinkHook(useOpenTable, { client });
    store.dispatch(Session.Project.select(project.key));
    await handler({ client, key: created.key });
    const tab = await resolveFocusedTab(store, client);
    if (tab.variant !== "resource") throw new Error("expected a resource tab");
    expect(tab.resource).toEqual({ type: "table", key: created.key });
  });
});
