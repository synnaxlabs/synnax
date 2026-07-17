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

import { STORE_KEY } from "@/label/store";
import { type Key, type Label } from "@/label/types.gen";
import { createTestClient } from "@/testutil/client";

const client = createTestClient();
afterAll(() => client.close());

describe("label store", () => {
  it("caches sets and corpses deletes from live signals", async () => {
    await client.cache.engine.ensureStreaming();
    const store = client.cache.engine.store<Key, Label>(STORE_KEY);
    const label = await client.labels.create({
      name: `label-${id.create()}`,
      color: "#12E774",
    });
    await expect
      .poll(() => store.get(label.key)?.name, { timeout: 5000 })
      .toEqual(label.name);
    await client.labels.delete(label.key);
    await expect
      .poll(() => store.status(label.key), { timeout: 5000 })
      .toBe("tombstoned");
    expect(store.getTombstone(label.key)?.corpse.name).toEqual(label.name);
  }, 20000);
});
