// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id } from "@synnaxlabs/x";
import { afterAll, describe, expect, it } from "vitest";

import { STORE_KEY } from "@/table/store";
import { type Key, type Table } from "@/table/types.gen";
import { createTestClient } from "@/testutil/client";

const client = createTestClient();
afterAll(() => client.close());

describe("table store", () => {
  it("tombstones deletes from live delete signals", async () => {
    await client.cache.ensureStreaming();
    const project = await client.projects.create({ name: `tbl-${id.create()}` });
    const table = await client.tables.create(project.key, {
      name: `table-${id.create()}`,
    });
    const store = client.cache.table<Key, Table>(STORE_KEY);
    store.set(table.key, table);
    await client.tables.delete(table.key);
    await expect
      .poll(() => store.status(table.key), { timeout: 5000 })
      .toBe("tombstoned");
    expect(store.getTombstone(table.key)?.corpse.name).toEqual(table.name);
  }, 20000);
});
