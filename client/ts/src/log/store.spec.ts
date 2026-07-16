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

import { STORE_KEY } from "@/log/store";
import { type Key, type Log } from "@/log/types.gen";
import { createTestClient } from "@/testutil/client";

const client = createTestClient();
afterAll(() => client.close());

describe("log store", () => {
  it("tombstones deletes from live delete signals", async () => {
    await client.cache.engine.ensureStreaming();
    const project = await client.projects.create({ name: `log-${id.create()}` });
    const log = await client.logs.create(project.key, { name: `log-${id.create()}` });
    client.cache.engine.store<Key, Log>(STORE_KEY).set(log.key, log);
    await client.logs.delete(log.key);
    await expect
      .poll(() => client.logs.store.status(log.key), { timeout: 5000 })
      .toBe("tombstoned");
    expect(client.logs.store.getTombstone(log.key)?.corpse.name).toEqual(log.name);
  }, 20000);
});
