// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { query, table } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { waitFor } from "@testing-library/react";
import { assert, describe, expect, it } from "vitest";

import { Table } from "@/feature/table";
import { uniqueName } from "@/testutil";

const client = createTestClient();

describe("Table.TABS restore", () => {
  it("should restore a deleted table into the given project", async () => {
    const { restore } = Table.TABS[table.TYPE_RESOURCE_ID.type];
    assert(restore != null);
    const from = await client.projects.create({ name: uniqueName("from") });
    const to = await client.projects.create({ name: uniqueName("to") });
    const created = await client.tables.create({
      project: from.key,
      name: uniqueName("table"),
    });
    await client.tables.retrieve(created.key);
    await client.tables.delete(created.key);
    await waitFor(() =>
      expect(query.Deleted.matches(client.tables.watch(created.key).getCached())).toBe(
        true,
      ),
    );
    await restore({ client, project: to.key, resource: table.resourceID(created.key) });
    const restored = await createTestClient().tables.retrieve(created.key);
    expect(restored.project).toEqual(to.key);
  });
});
