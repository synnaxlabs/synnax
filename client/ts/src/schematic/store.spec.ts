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

import { STORE_KEY } from "@/schematic/store";
import { type Key, type Schematic } from "@/schematic/types.gen";
import { createTestClient } from "@/testutil/client";

const client = createTestClient();
afterAll(() => client.close());

describe("schematic store", () => {
  it("tombstones deletes from live delete signals", async () => {
    await client.cache.engine.ensureStreaming();
    const project = await client.projects.create({ name: `sch-${id.create()}` });
    const schematic = await client.schematics.create(project.key, {
      name: `schematic-${id.create()}`,
    });
    client.cache.engine.store<Key, Schematic>(STORE_KEY).set(schematic.key, schematic);
    await client.schematics.delete(schematic.key);
    await expect
      .poll(() => client.schematics.store.status(schematic.key), { timeout: 5000 })
      .toBe("tombstoned");
    expect(client.schematics.store.getTombstone(schematic.key)?.corpse.name).toEqual(
      schematic.name,
    );
  }, 20000);
});
