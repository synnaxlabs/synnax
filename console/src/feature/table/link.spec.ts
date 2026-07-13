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
import { renderLinkHook } from "@/testutil";

const client = createTestClient();

describe("Table.useLink", () => {
  it("should place a table layout for the retrieved table", async () => {
    const project = await client.projects.create({ name: id.create(), layout: {} });
    const table = await client.tables.create(project.key, { name: "Sensor Table" });
    const { handler, store } = await renderLinkHook(Table.useLink);
    await handler({ client, key: table.key });
    expect(Session.Layout.select(store.getState(), table.key)?.name).toBe(
      "Sensor Table",
    );
  });
});
