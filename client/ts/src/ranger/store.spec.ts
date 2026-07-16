// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id, TimeRange, TimeStamp } from "@synnaxlabs/x";
import { afterAll, describe, expect, it } from "vitest";

import { createTestClient } from "@/testutil/client";

const client = createTestClient();
afterAll(() => client.close());

describe("ranger store", () => {
  it("caches sugared sets and corpses deletes from live signals", async () => {
    await client.cache.engine.ensureStreaming();
    const range = await client.ranges.create({
      name: `range-${id.create()}`,
      timeRange: new TimeRange(TimeStamp.now(), TimeStamp.now().add(TimeStamp.SECOND)),
    });
    await expect
      .poll(() => client.ranges.store.get(range.key)?.name, { timeout: 5000 })
      .toEqual(range.name);
    await client.ranges.delete(range.key);
    await expect
      .poll(() => client.ranges.store.status(range.key), { timeout: 5000 })
      .toBe("tombstoned");
    expect(client.ranges.store.getTombstone(range.key)?.corpse.name).toEqual(
      range.name,
    );
  }, 20000);
});
