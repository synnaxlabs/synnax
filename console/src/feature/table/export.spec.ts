// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError } from "@synnaxlabs/client";
import { describe, expect, it } from "vitest";

import { Table } from "@/feature/table";
import {
  client,
  createCellGrid,
  createPreloadedState,
  project,
} from "@/feature/table/testutil";
import { createTestStore, uniqueName } from "@/testutil";

const extract = Table.EXTRACTORS[Table.LAYOUT_TYPE];

describe("table export", () => {
  it("serializes the table with its layout type, version, and the layout's name", async () => {
    const t = await client.tables.create(await project(), {
      name: uniqueName("table"),
      ...createCellGrid(),
    });
    const store = await createTestStore({
      preloadedState: createPreloadedState(t.key, "Renamed Layout"),
    });
    const file = await extract(t.key, { store, client });
    expect(file.name).toBe("Renamed Layout");
    const data = JSON.parse(file.data);
    expect(data.type).toBe(Table.LAYOUT_TYPE);
    expect(data.version).toBe("1.0.0");
    expect(data.key).toBe(t.key);
    expect(data.rows).toHaveLength(1);
    expect(Object.keys(data.cells).sort()).toEqual(["a", "b"]);
  });

  it("falls back to the server name when no layout exists", async () => {
    const t = await client.tables.create(await project(), {
      name: uniqueName("table"),
    });
    const store = await createTestStore();
    const file = await extract(t.key, { store, client });
    expect(file.name).toBe(t.name);
  });

  it("throws a DisconnectedError without a client", async () => {
    const store = await createTestStore();
    await expect(extract("some-key", { store, client: null })).rejects.toSatisfy((e) =>
      DisconnectedError.matches(e),
    );
  });
});
